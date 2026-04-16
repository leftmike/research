// Package sandbox runs a target program on Linux under a seccomp user-notify
// filter so that its file-open syscalls can be audited and dynamically
// authorized by a supervisor in the parent process.
//
// v1 intercepts openat and open only. Allowed syscalls proceed via
// SECCOMP_USER_NOTIF_FLAG_CONTINUE; denied syscalls return EACCES. An optional
// JSON-lines audit log records every decision.
//
// Usage in a consumer program:
//
//	func main() {
//	    sandbox.Init()   // must be first
//	    // ... normal program logic ...
//	    code, err := sandbox.Run(sandbox.Config{
//	        Program:    "/bin/cat",
//	        Args:       []string{"cat", "/etc/hostname"},
//	        Authorizer: func(r sandbox.OpenRequest) bool { return true },
//	        AuditLog:   os.Stderr,
//	    })
//	    os.Exit(code)
//	}
package sandbox

import "io"

// initEnvVar is the sentinel the parent sets so that Init, in the re-exec'd
// child, knows it should install the seccomp filter and exec the target.
const initEnvVar = "__SANDBOX_INIT"

// OpenRequest describes an intercepted file-open syscall.
type OpenRequest struct {
	PID   int    // sandboxed process id
	Nr    int    // syscall number (SYS_openat or SYS_open)
	Dirfd int    // dirfd argument (openat only; -1 for open or AT_FDCWD)
	Path  string // NUL-terminated path read from the target's address space
	Flags int    // open(2) flags
	Mode  uint32 // open(2) mode (only meaningful with O_CREAT)
}

// Authorizer decides whether a single intercepted open syscall is allowed.
// Returning true allows the syscall; false causes the kernel to return EACCES.
// It runs synchronously in the supervisor goroutine.
type Authorizer func(OpenRequest) bool

// Config configures a sandboxed execution.
type Config struct {
	Program    string     // absolute path of the target binary
	Args       []string   // argv, conventionally Args[0] == basename of Program
	Authorizer Authorizer // nil ⇒ allow-all
	AuditLog   io.Writer  // nil ⇒ discard; otherwise one JSON object per line

	// Target's stdio; nil inherits the parent's file (os.Stdin/Stdout/Stderr).
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}
