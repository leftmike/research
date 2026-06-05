//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang execsnoop ./execsnoop.bpf.c -- -I${GOPATH}/pkg/mod/github.com/cilium/ebpf@v0.21.0/examples/headers -I/usr/include/x86_64-linux-gnu -O2 -g -Wall -Werror

// execsnoop traces execve(2) and execveat(2) system calls and prints the
// command line, process name, PID, PPID, UID, inode, and return value for
// every exec attempt, modelled after the iovisor/bcc execsnoop tool.
//
// The inode is resolved in userspace by stat-ing the filename carried in the
// ring-buffer event.  Because the event arrives while the execed process is
// still alive (or the file still exists for failed execs), the stat races only
// in the uncommon case where the executable is deleted between exec and event
// delivery.
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
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

const (
	argSize = 128
	maxArgs = 20
)

// event mirrors struct event in execsnoop.bpf.c exactly.
// Layout: pid(4)+ppid(4)+uid(4)+ret(4)+args_count(4)+comm(16)+filename(256)+args(2560) = 2852 bytes.
type event struct {
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

// exeInode returns the inode number of the file at path, or 0 on error.
// It is called after the ring-buffer event is received, so the file is
// generally still present (the process is alive or the path still exists).
func exeInode(path string) uint64 {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return 0
	}
	return st.Ino
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

func main() {
	pid := flag.Int("p", 0, "trace only `PID`")
	uid := flag.Uint("u", 0xFFFFFFFF, "trace only `UID` (default: all users)")
	duration := flag.Duration("d", 0, "stop after `duration` (e.g. 10s)")
	failOnly := flag.Bool("x", false, "only print failed execs")
	showTS := flag.Bool("T", false, "include absolute HH:MM:SS.nsec timestamp")
	flag.Parse()

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("remove memlock limit: %v", err)
	}

	spec, err := loadExecsnoop()
	if err != nil {
		log.Fatalf("load BPF spec: %v", err)
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

		filename := nullStr(e.Filename[:])
		inode := exeInode(filename)
		comm := nullStr(e.Comm[:])
		args := parseArgs(e.Args[:], e.ArgsCount)
		printEvent(*showTS, e.Pid, e.Ppid, e.Uid, e.Ret, inode, comm, args)
	}
}

func printHeader(showTS bool) {
	if showTS {
		fmt.Printf("%-20s ", "TIME")
	}
	fmt.Printf("%-16s %-7s %-7s %-6s %-4s %-11s %s\n",
		"PCOMM", "PID", "PPID", "UID", "RET", "INODE", "ARGS")
}

func printEvent(showTS bool, pid, ppid, uid uint32, ret int32, inode uint64, comm, args string) {
	if showTS {
		fmt.Printf("%-20s ", time.Now().Format("15:04:05.000000000"))
	}
	inodeStr := "-"
	if inode != 0 {
		inodeStr = fmt.Sprintf("%d", inode)
	}
	fmt.Printf("%-16s %-7d %-7d %-6d %-4d %-11s %s\n",
		comm, pid, ppid, uid, ret, inodeStr, args)
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
