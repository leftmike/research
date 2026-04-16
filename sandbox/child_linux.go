//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

// childInit runs in the re-exec'd child before execve. It installs the seccomp
// filter, sends the resulting notifier fd back to the parent over fd 3, then
// execs the target program. It never returns; on error it exits with status 127.
func childInit() {
	// Stay on one OS thread: PR_SET_NO_NEW_PRIVS and the seccomp(2) install
	// must happen on the same thread that calls execve.
	runtime.LockOSThread()

	program := os.Getenv("__SANDBOX_PROGRAM")
	// os.Args was set by the parent to [program, arg1, arg2, ...].
	args := os.Args
	if program == "" || len(args) == 0 {
		childFatal(fmt.Errorf("missing __SANDBOX_PROGRAM or args"))
	}

	// fd 3 is the child end of the socketpair, passed via cmd.ExtraFiles[0].
	const sockFd = 3

	// 1. PR_SET_NO_NEW_PRIVS allows loading a seccomp filter without CAP_SYS_ADMIN.
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		childFatal(fmt.Errorf("prctl(PR_SET_NO_NEW_PRIVS): %w", err))
	}

	// 2. Install the seccomp filter with FLAG_NEW_LISTENER; the listener fd
	//    is returned from the seccomp(2) syscall.
	listenerFd, err := installListener()
	if err != nil {
		childFatal(fmt.Errorf("install seccomp: %w", err))
	}

	// 3. Send the listener fd to the parent via SCM_RIGHTS.
	if err := sendFd(sockFd, listenerFd); err != nil {
		childFatal(fmt.Errorf("send listener fd: %w", err))
	}
	unix.Close(listenerFd)
	unix.Close(sockFd)

	// Clear the sentinel env vars so the target program (even if it is this
	// same binary) does not re-enter childInit.
	os.Unsetenv(initEnvVar)
	os.Unsetenv("__SANDBOX_PROGRAM")

	// 4. Exec the target; the seccomp filter is now active on this thread.
	if err := unix.Exec(program, args, os.Environ()); err != nil {
		childFatal(fmt.Errorf("exec %s: %w", program, err))
	}
}

// sendFd sends fd over the Unix socketpair end at sockFd via SCM_RIGHTS.
func sendFd(sockFd, fd int) error {
	oob := unix.UnixRights(fd)
	// Body must be non-empty on some kernels for SOCK_SEQPACKET.
	return unix.Sendmsg(sockFd, []byte{0}, oob, nil, 0)
}

func childFatal(err error) {
	fmt.Fprintf(os.Stderr, "sandbox child: %v\n", err)
	os.Exit(127)
}
