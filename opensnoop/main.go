//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang opensnoop ./opensnoop.bpf.c -- -I/usr/include/x86_64-linux-gnu -O2 -g -Wall -Werror

// opensnoop traces open(2) / openat(2) / openat2(2) system calls and prints
// the filename, process, result, and optionally the flags for every file-open
// attempt, modelled after the iovisor/bcc opensnoop tool.
//
// Requirements:
//   - Linux 5.8+ (BPF ring buffer support)
//   - CAP_BPF and CAP_PERFMON, or CAP_SYS_ADMIN
//   - Kernel built with CONFIG_BPF_SYSCALL, CONFIG_FTRACE_SYSCALLS
//
// Usage:
//
//	sudo opensnoop [-p pid] [-n name] [-d duration] [-x] [-f] [-T]
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

// event mirrors struct event in opensnoop.bpf.c exactly.
// Layout: pid(4) + ret(4) + flags(4) + comm(16) + fname(256) = 284 bytes; no padding.
type event struct {
	Pid   uint32
	Ret   int32
	Flags int32
	Comm  [16]byte
	Fname [256]byte
}

func nullStr(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

// flagStr decodes open(2) O_* flags into a human-readable string.
func flagStr(f int32) string {
	var parts []string
	switch f & 0x3 {
	case 0:
		parts = append(parts, "O_RDONLY")
	case 1:
		parts = append(parts, "O_WRONLY")
	case 2:
		parts = append(parts, "O_RDWR")
	}
	for bit, name := range map[int32]string{
		0o100:     "O_CREAT",
		0o200:     "O_EXCL",
		0o400:     "O_NOCTTY",
		0o1000:    "O_TRUNC",
		0o2000:    "O_APPEND",
		0o4000:    "O_NONBLOCK",
		0o2000000: "O_CLOEXEC",
	} {
		if f&bit != 0 {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, "|")
}

func main() {
	pid := flag.Int("p", 0, "trace only `PID`")
	name := flag.String("n", "", "trace only processes whose name contains `NAME`")
	duration := flag.Duration("d", 0, "stop after `duration` (e.g. 10s)")
	failOnly := flag.Bool("x", false, "only print failed opens")
	showFlags := flag.Bool("f", false, "show open flags column")
	showTS := flag.Bool("T", false, "include absolute HH:MM:SS.nsec timestamp")
	flag.Parse()

	// Allow the process to lock memory for eBPF maps.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("remove memlock limit: %v", err)
	}

	// Load the embedded BPF object into a CollectionSpec so we can set
	// the target_pid constant before the programs are committed to the kernel.
	spec, err := loadOpensnoop()
	if err != nil {
		log.Fatalf("load BPF spec: %v", err)
	}

	if *pid != 0 {
		if err := spec.Variables["target_pid"].Set(uint32(*pid)); err != nil {
			log.Fatalf("set target_pid constant: %v", err)
		}
	}

	var objs opensnoopObjects
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		handleLoadError(err)
	}
	defer objs.Close()

	// Attach tracepoints.  openat is required; open and openat2 are optional.
	// The open(2) syscall has no distinct tracepoint on x86-64 (glibc calls
	// openat directly); openat2(2) landed in Linux 5.6.
	links := attachAll(&objs.opensnoopPrograms)
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

	// Close the ring buffer on SIGINT / SIGTERM or after -d duration.
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

	start := time.Now()
	printHeader(*showTS, *showFlags)

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

		comm := nullStr(e.Comm[:])
		if *name != "" && !strings.Contains(comm, *name) {
			continue
		}

		fd, errno := int(e.Ret), 0
		if e.Ret < 0 {
			fd = -1
			errno = int(-e.Ret)
		}
		if *failOnly && errno == 0 {
			continue
		}

		ts := time.Since(start).Seconds()
		fname := nullStr(e.Fname[:])

		printEvent(*showTS, *showFlags, ts, e.Pid, comm, fd, errno, e.Flags, fname)
	}
}

func printHeader(showTS, showFlags bool) {
	if showTS {
		fmt.Printf("%-20s ", "TIME")
	}
	if showFlags {
		fmt.Printf("%-10s %-7s %-16s %4s %3s %-22s %s\n",
			"TIME(s)", "PID", "COMM", "FD", "ERR", "FLAGS", "PATH")
	} else {
		fmt.Printf("%-10s %-7s %-16s %4s %3s %s\n",
			"TIME(s)", "PID", "COMM", "FD", "ERR", "PATH")
	}
}

func printEvent(showTS, showFlags bool, ts float64, pid uint32, comm string, fd, errno int, flags int32, fname string) {
	if showTS {
		fmt.Printf("%-20s ", time.Now().Format("15:04:05.000000000"))
	}
	if showFlags {
		fmt.Printf("%-10.6f %-7d %-16s %4d %3d %-22s %s\n",
			ts, pid, comm, fd, errno, flagStr(flags), fname)
	} else {
		fmt.Printf("%-10.6f %-7d %-16s %4d %3d %s\n",
			ts, pid, comm, fd, errno, fname)
	}
}

// attachAll links every opensnoop program to its tracepoint and returns all
// attached links.  Required tracepoints (openat) cause a fatal log on failure;
// optional ones (open, openat2) are silently skipped on missing tracepoints.
func attachAll(progs *opensnoopPrograms) []link.Link {
	type tp struct {
		group    string
		name     string
		prog     *ebpf.Program
		required bool
	}
	tps := []tp{
		// openat is the canonical file-open syscall on modern Linux.
		{"syscalls", "sys_enter_openat", progs.TraceEnterOpenat, true},
		{"syscalls", "sys_exit_openat", progs.TraceExitOpenat, true},
		// open still appears on 32-bit ABIs and some non-x86 architectures.
		{"syscalls", "sys_enter_open", progs.TraceEnterOpen, false},
		{"syscalls", "sys_exit_open", progs.TraceExitOpen, false},
		// openat2 landed in Linux 5.6.
		{"syscalls", "sys_enter_openat2", progs.TraceEnterOpenat2, false},
		{"syscalls", "sys_exit_openat2", progs.TraceExitOpenat2, false},
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

// handleLoadError prints a descriptive message for common BPF program load failures.
func handleLoadError(err error) {
	switch {
	case errors.Is(err, syscall.EPERM):
		log.Fatalf("load BPF objects: %v\n"+
			"Hint: run as root or grant CAP_BPF and CAP_PERFMON capabilities.", err)
	case errors.Is(err, syscall.EINVAL):
		// EINVAL typically means BPF_PROG_TYPE_TRACEPOINT is unsupported; the
		// verifier log will be empty in that case.
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
