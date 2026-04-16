// Command sbx runs a target program under a seccomp sandbox that audits and
// authorizes its file-open syscalls. Authorization can be:
//
//   - -allow REGEX   allow opens whose path matches; deny otherwise
//   - -deny  REGEX   deny opens whose path matches; allow otherwise
//   - -interactive   prompt y/n on /dev/tty for each open (default if no policy)
//
// An audit log (JSON lines) is written to stderr by default, or to a file via
// -audit FILE.
//
// Examples:
//
//	sbx -allow '^/etc/' cat /etc/hostname
//	sbx -deny  '.*'     cat /etc/hostname      # EACCES on every open
//	sbx -interactive    ls /tmp
//	sbx -audit out.log  cat /etc/hostname      # allow-all with file audit log
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/leftmike/research/sandbox"
)

func main() {
	// Must be the very first call so the re-exec'd child can install the seccomp
	// filter before the target program starts.
	sandbox.Init()

	allowPat := flag.String("allow", "", "allow opens matching this regex; deny all others")
	denyPat := flag.String("deny", "", "deny opens matching this regex; allow all others")
	interactive := flag.Bool("interactive", false, "prompt y/n on /dev/tty for each open")
	auditFile := flag.String("audit", "", "write JSON-line audit log to this file (default: stderr)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: sbx [flags] PROGRAM [ARGS...]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	// Exactly one policy mode.
	modes := 0
	for _, v := range []bool{*allowPat != "", *denyPat != "", *interactive} {
		if v {
			modes++
		}
	}
	if modes > 1 {
		fmt.Fprintln(os.Stderr, "sbx: at most one of -allow, -deny, -interactive may be used")
		os.Exit(2)
	}

	// Resolve the program early so we get a clear error message.
	program, err := exec.LookPath(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbx: %v\n", err)
		os.Exit(127)
	}

	auditW, closeAudit, err := openAuditWriter(*auditFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbx: %v\n", err)
		os.Exit(1)
	}
	defer closeAudit()

	auth, err := buildAuthorizer(*allowPat, *denyPat, *interactive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbx: %v\n", err)
		os.Exit(1)
	}

	code, err := sandbox.Run(sandbox.Config{
		Program:    program,
		Args:       flag.Args(),
		Authorizer: auth,
		AuditLog:   auditW,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "sbx: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func openAuditWriter(path string) (io.Writer, func(), error) {
	if path == "" {
		return os.Stderr, func() {}, nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open audit log %q: %w", path, err)
	}
	return f, func() { f.Close() }, nil
}

func buildAuthorizer(allowPat, denyPat string, interactive bool) (sandbox.Authorizer, error) {
	switch {
	case allowPat != "":
		re, err := regexp.Compile(allowPat)
		if err != nil {
			return nil, fmt.Errorf("-allow: %w", err)
		}
		return func(r sandbox.OpenRequest) bool { return re.MatchString(r.Path) }, nil

	case denyPat != "":
		re, err := regexp.Compile(denyPat)
		if err != nil {
			return nil, fmt.Errorf("-deny: %w", err)
		}
		return func(r sandbox.OpenRequest) bool { return !re.MatchString(r.Path) }, nil

	case interactive:
		return newPrompter(), nil

	default:
		// No explicit policy: allow everything (audit only).
		return func(sandbox.OpenRequest) bool { return true }, nil
	}
}

// newPrompter returns an Authorizer that prompts on /dev/tty for each open.
// Calls are serialized by a mutex since the supervisor runs one goroutine.
func newPrompter() sandbox.Authorizer {
	var mu sync.Mutex
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		// Fall back to stderr+stdin (may interleave with target output).
		reader := bufio.NewReader(os.Stdin)
		return func(r sandbox.OpenRequest) bool {
			mu.Lock()
			defer mu.Unlock()
			return promptOnce(os.Stderr, reader, r)
		}
	}
	reader := bufio.NewReader(tty)
	return func(r sandbox.OpenRequest) bool {
		mu.Lock()
		defer mu.Unlock()
		return promptOnce(tty, reader, r)
	}
}

func promptOnce(out io.Writer, in *bufio.Reader, r sandbox.OpenRequest) bool {
	fmt.Fprintf(out,
		"[sbx] pid=%d %s path=%q flags=0x%x — allow? [y/N] ",
		r.PID, syscallLabel(r.Nr), r.Path, r.Flags,
	)
	line, _ := in.ReadString('\n')
	ans := strings.TrimSpace(line)
	return strings.EqualFold(ans, "y") || strings.EqualFold(ans, "yes")
}

func syscallLabel(nr int) string {
	// Keep cmd/ independent of the internal constants.
	switch nr {
	case 257: // SYS_openat on amd64
		return "openat"
	case 2: // SYS_open on amd64
		return "open"
	default:
		return fmt.Sprintf("syscall_%d", nr)
	}
}
