//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang execsnoop ./execsnoop.bpf.c -- -I${GOPATH}/pkg/mod/github.com/cilium/ebpf@v0.21.0/examples/headers -I/usr/include/x86_64-linux-gnu -O2 -g -Wall -Werror

// execsnoop traces execve(2) and execveat(2) system calls and prints the
// command line, process name, PID, PPID, UID, inode, and return value for
// every exec attempt, modelled after the iovisor/bcc execsnoop tool.
//
// Inode resolution strategy
//
// At startup, Go queries the running kernel's BTF (/sys/kernel/btf/vmlinux)
// for the exact byte offsets of four fields:
//
//	task_struct.mm  →  mm_struct.exe_file  →  file.f_inode  →  inode.i_ino
//
// Those offsets are injected into the BPF program before it is loaded; the
// kernel-side handle_exit() then walks the pointer chain with
// bpf_probe_read_kernel() and stores the result in the ring-buffer event.
//
// On kernels without BTF (CONFIG_DEBUG_INFO_BTF not set) the offsets cannot
// be resolved and execsnoop exits with an error rather than running with
// inode/dev fields silently unset.
//
// Requirements:
//   - Linux 5.8+ (BPF ring buffer support)
//   - CAP_BPF and CAP_PERFMON, or CAP_SYS_ADMIN
//   - Kernel built with CONFIG_BPF_SYSCALL, CONFIG_FTRACE_SYSCALLS
//
// Usage:
//
//	sudo execsnoop [-p pid] [-u uid] [-d duration] [-x] [-T]
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

const (
	argSize = 128
	maxArgs = 20
)

// event mirrors struct event in execsnoop.bpf.c exactly.
// inode is first so the __u64 sits at offset 0 with no padding.
// Layout: inode(8)+cgroup_id(8)+dev(4)+pid(4)+ppid(4)+uid(4)+ret(4)+args_count(4)+
//
//	comm(16)+filename(256)+args(2560) = 2872 bytes.
type event struct {
	Inode     uint64
	CgroupId  uint64
	Dev       uint32
	Pid       uint32
	Ppid      uint32
	Uid       uint32
	Ret       int32
	ArgsCount uint32
	Comm      [16]byte
	Filename  [256]byte
	Args      [argSize * maxArgs]byte
}

func nullStr(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

// reContainerID matches a 64-hex-char Docker/OCI container ID in a cgroup path.
var reContainerID = regexp.MustCompile(`[0-9a-f]{64}`)

// cgroupCache maps cgroup inode ID → short container ID (or "-").
// The cgroup outlives all its processes, so this lookup is reliable even for
// processes that have already exited.
var cgroupCache = map[uint64]string{}

// resolveContainer returns the 12-char container short ID for the given BPF
// cgroup ID, or "-" for host processes.  Results are cached by cgroup ID so
// the /sys/fs/cgroup walk happens at most once per container.
func resolveContainer(cgroupId uint64) string {
	if id, ok := cgroupCache[cgroupId]; ok {
		return id
	}
	id := findContainerByCgroupId(cgroupId)
	cgroupCache[cgroupId] = id
	return id
}

// findContainerByCgroupId walks /sys/fs/cgroup looking for a directory whose
// inode equals cgroupId, then extracts the Docker container ID from its path.
func findContainerByCgroupId(cgroupId uint64) string {
	var found string
	filepath.WalkDir("/sys/fs/cgroup", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Ino == cgroupId {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if m := reContainerID.FindString(found); m != "" {
		return m[:12]
	}
	return "-"
}

// parseArgs reconstructs the command line from the fixed-size argv slots.
func parseArgs(args []byte, count uint32) string {
	parts := make([]string, 0, count)
	for i := uint32(0); i < count && i < maxArgs; i++ {
		start := i * argSize
		s := nullStr(args[start : start+argSize])
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// setInodeOffsets loads the running kernel's BTF and injects the byte
// offsets needed to walk task_struct → mm_struct → file → inode → i_ino
// (and inode → super_block → s_dev) into the BPF CollectionSpec.  Must be
// called before spec.LoadAndAssign.  Returns an error if BTF is unavailable
// or any required field cannot be resolved.
func setInodeOffsets(spec *ebpf.CollectionSpec) error {
	kspec, err := btf.LoadKernelSpec()
	if err != nil {
		return fmt.Errorf("load kernel BTF: %w", err)
	}

	type lookup struct {
		structName string
		fieldName  string
		varName    string
	}
	chain := []lookup{
		{"task_struct", "mm",       "off_task_mm"},
		{"mm_struct",   "exe_file", "off_mm_exefile"},
		{"file",        "f_inode",  "off_file_inode"},
		{"inode",       "i_ino",    "off_inode_ino"},
		{"inode",       "i_sb",     "off_inode_sb"},
		{"super_block", "s_dev",    "off_sb_dev"},
	}

	for _, l := range chain {
		off, ok := structFieldOffset(kspec, l.structName, l.fieldName)
		if !ok {
			return fmt.Errorf("resolve %s.%s from kernel BTF", l.structName, l.fieldName)
		}
		if err := spec.Variables[l.varName].Set(off); err != nil {
			return fmt.Errorf("set %s: %w", l.varName, err)
		}
	}
	return nil
}

// structFieldOffset returns the byte offset of fieldName within the named
// struct type from the given BTF spec, recursing into anonymous nested
// struct/union members (kernels group related fields this way, e.g.
// mm_struct wraps most of its members in an unnamed struct for layout
// randomization).
func structFieldOffset(kspec *btf.Spec, structName, fieldName string) (uint32, bool) {
	var s *btf.Struct
	if err := kspec.TypeByName(structName, &s); err != nil {
		return 0, false
	}
	return memberOffset(s, fieldName, 0)
}

// memberOffset searches the members of a struct or union for fieldName,
// recursing into anonymous nested structs/unions and accumulating their
// base offset.
func memberOffset(t btf.Type, fieldName string, base uint32) (uint32, bool) {
	var members []btf.Member
	switch v := t.(type) {
	case *btf.Struct:
		members = v.Members
	case *btf.Union:
		members = v.Members
	default:
		return 0, false
	}
	for _, m := range members {
		off := base + m.Offset.Bytes()
		if m.Name == fieldName {
			return off, true
		}
		if m.Name == "" {
			if resolved, ok := memberOffset(m.Type, fieldName, off); ok {
				return resolved, true
			}
		}
	}
	return 0, false
}

func main() {
	pid := flag.Int("p", 0, "trace only `PID`")
	uid := flag.Uint("u", 0xFFFFFFFF, "trace only `UID` (default: all users)")
	duration := flag.Duration("d", 0, "stop after `duration` (e.g. 10s)")
	failOnly := flag.Bool("x", false, "only print failed execs")
	showTS := flag.Bool("T", false, "include absolute HH:MM:SS.nsec timestamp")
	containerOnly := flag.Bool("c", false, "only print execs from containers")
	flag.Parse()

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("remove memlock limit: %v", err)
	}

	spec, err := loadExecsnoop()
	if err != nil {
		log.Fatalf("load BPF spec: %v", err)
	}

	// Inject kernel struct offsets from BTF so the BPF program can resolve
	// the inode and device itself.  BTF is required; without it the
	// program cannot resolve these fields, so fail rather than running
	// with them silently unset.
	if err := setInodeOffsets(spec); err != nil {
		log.Fatalf("resolve inode offsets from kernel BTF: %v\n"+
			"Hint: kernel must be built with CONFIG_DEBUG_INFO_BTF=y "+
			"(check that /sys/kernel/btf/vmlinux exists).", err)
	}

	if *pid != 0 {
		if err := spec.Variables["target_pid"].Set(uint32(*pid)); err != nil {
			log.Fatalf("set target_pid constant: %v", err)
		}
	}
	if *uid != 0xFFFFFFFF {
		if err := spec.Variables["target_uid"].Set(uint32(*uid)); err != nil {
			log.Fatalf("set target_uid constant: %v", err)
		}
	}

	var objs execsnoopObjects
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		handleLoadError(err)
	}
	defer objs.Close()

	links := attachAll(&objs.execsnoopPrograms)
	defer func() {
		for _, l := range links {
			l.Close()
		}
	}()

	rd, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		log.Fatalf("open ring buffer: %v", err)
	}
	defer rd.Close()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if *duration > 0 {
			select {
			case <-time.After(*duration):
			case <-sigs:
			}
		} else {
			<-sigs
		}
		rd.Close()
	}()

	printHeader(*showTS)

	var e event
	for {
		rec, err := rd.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				break
			}
			log.Printf("ring buffer read: %v", err)
			continue
		}

		if err := binary.Read(bytes.NewReader(rec.RawSample), binary.LittleEndian, &e); err != nil {
			log.Printf("parse event: %v", err)
			continue
		}

		if *failOnly && e.Ret == 0 {
			continue
		}

		container := resolveContainer(e.CgroupId)

		if *containerOnly && container == "-" {
			continue
		}

		comm := nullStr(e.Comm[:])
		args := parseArgs(e.Args[:], e.ArgsCount)
		printEvent(*showTS, e.Pid, e.Ppid, e.Uid, e.Ret, e.Inode, e.Dev, container, comm, args)
	}
}

func printHeader(showTS bool) {
	if showTS {
		fmt.Printf("%-20s ", "TIME")
	}
	fmt.Printf("%-16s %-7s %-7s %-6s %-4s %-11s %-7s %-14s %s\n",
		"PCOMM", "PID", "PPID", "UID", "RET", "INODE", "DEV", "CONTAINER", "ARGS")
}

func printEvent(showTS bool, pid, ppid, uid uint32, ret int32, inode uint64, dev uint32, container, comm, args string) {
	if showTS {
		fmt.Printf("%-20s ", time.Now().Format("15:04:05.000000000"))
	}
	inodeStr := "-"
	if inode != 0 {
		inodeStr = fmt.Sprintf("%d", inode)
	}
	devStr := "-"
	if dev != 0 {
		devStr = fmt.Sprintf("%d", dev)
	}
	fmt.Printf("%-16s %-7d %-7d %-6d %-4d %-11s %-7s %-14s %s\n",
		comm, pid, ppid, uid, ret, inodeStr, devStr, container, args)
}

// attachAll links every execsnoop program to its tracepoint.
// The sched and execve tracepoints are required; execveat is optional
// (present on all kernels since 3.19, but we tolerate absence for robustness).
func attachAll(progs *execsnoopPrograms) []link.Link {
	type tp struct {
		group    string
		name     string
		prog     *ebpf.Program
		required bool
	}
	tps := []tp{
		{"sched", "sched_process_fork", progs.TraceProcessFork, true},
		{"sched", "sched_process_exit", progs.TraceProcessExit, true},
		{"syscalls", "sys_enter_execve", progs.TraceEnterExecve, true},
		{"syscalls", "sys_exit_execve", progs.TraceExitExecve, true},
		{"syscalls", "sys_enter_execveat", progs.TraceEnterExecveat, false},
		{"syscalls", "sys_exit_execveat", progs.TraceExitExecveat, false},
	}

	var links []link.Link
	for _, t := range tps {
		l, err := link.Tracepoint(t.group, t.name, t.prog, nil)
		if err != nil {
			if t.required {
				log.Fatalf("attach tracepoint %s/%s: %v\n"+
					"Hint: kernel must be built with CONFIG_FTRACE_SYSCALLS "+
					"and tracefs must be mounted.", t.group, t.name, err)
			}
			continue
		}
		links = append(links, l)
	}
	return links
}

func handleLoadError(err error) {
	switch {
	case errors.Is(err, syscall.EPERM):
		log.Fatalf("load BPF objects: %v\n"+
			"Hint: run as root or grant CAP_BPF and CAP_PERFMON capabilities.", err)
	case errors.Is(err, syscall.EINVAL):
		log.Fatalf("load BPF objects: %v\n"+
			"Hint: BPF tracepoint programs require Linux 4.7+ built with\n"+
			"CONFIG_BPF_SYSCALL=y and CONFIG_FTRACE_SYSCALLS=y, and tracefs mounted.", err)
	default:
		var ve *ebpf.VerifierError
		if errors.As(err, &ve) && len(ve.Log) > 0 {
			log.Fatalf("BPF verifier rejected program: %+v", ve)
		}
		log.Fatalf("load BPF objects: %v", err)
	}
}
