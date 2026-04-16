//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

// Init must be called at the very top of main() in any program that will call
// Run. In the re-exec'd child it installs the seccomp filter, passes the
// resulting notifier fd up to the parent, and execs the target. In the parent
// it is a no-op and returns immediately.
func Init() {
	if os.Getenv(initEnvVar) != "1" {
		return
	}
	childInit() // never returns
}

// Run launches cfg.Program under a seccomp user-notify filter, supervising
// its openat/open syscalls. It returns the child's exit code and any
// supervisor-side error.
func Run(cfg Config) (int, error) {
	if cfg.Program == "" {
		return 0, fmt.Errorf("sandbox: Program is empty")
	}
	if cfg.Authorizer == nil {
		cfg.Authorizer = func(OpenRequest) bool { return true }
	}

	// socketpair: parent keeps pair[0], child inherits pair[1] as fd 3.
	pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return 0, fmt.Errorf("sandbox: socketpair: %w", err)
	}
	parentSock := os.NewFile(uintptr(pair[0]), "sbx-parent")
	childSock := os.NewFile(uintptr(pair[1]), "sbx-child")
	defer parentSock.Close()

	// Re-exec ourselves; childSock lands at fd 3 in the child via ExtraFiles.
	self, err := os.Executable()
	if err != nil {
		childSock.Close()
		return 0, fmt.Errorf("sandbox: os.Executable: %w", err)
	}

	cmd := exec.Command(self)
	cmd.Args = append([]string{cfg.Program}, cfg.Args...)
	cmd.Env = append(os.Environ(),
		initEnvVar+"=1",
		"__SANDBOX_PROGRAM="+cfg.Program,
	)
	cmd.ExtraFiles = []*os.File{childSock}

	if cfg.Stdin != nil {
		cmd.Stdin = cfg.Stdin
	} else {
		cmd.Stdin = os.Stdin
	}
	if cfg.Stdout != nil {
		cmd.Stdout = cfg.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	if cfg.Stderr != nil {
		cmd.Stderr = cfg.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		childSock.Close()
		return 0, fmt.Errorf("sandbox: start: %w", err)
	}
	// Parent no longer needs the child end of the socketpair.
	childSock.Close()

	// Receive the seccomp listener fd from the child via SCM_RIGHTS.
	listenerFd, err := recvListenerFd(parentSock)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return 0, fmt.Errorf("sandbox: recv listener fd: %w", err)
	}

	// Verify the kernel's notify struct sizes match our Go definitions.
	if err := checkNotifSizes(); err != nil {
		unix.Close(listenerFd)
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return 0, err
	}

	// Run supervisor in background; it exits when the child dies and the kernel
	// closes the listener fd (RECV returns ENOENT).
	supErrCh := make(chan error, 1)
	go func() {
		supErrCh <- supervise(listenerFd, cfg)
	}()

	waitErr := cmd.Wait()
	unix.Close(listenerFd) // wake supervisor if still blocked on RECV
	supErr := <-supErrCh

	code := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			return 0, fmt.Errorf("sandbox: wait: %w", waitErr)
		}
	}
	if supErr != nil {
		return code, supErr
	}
	return code, nil
}

// recvListenerFd blocks until the child sends the seccomp listener fd over the
// socketpair via SCM_RIGHTS.
func recvListenerFd(sock *os.File) (int, error) {
	buf := make([]byte, 1)
	oob := make([]byte, unix.CmsgSpace(4))

	rawConn, err := sock.SyscallConn()
	if err != nil {
		return -1, err
	}

	var n, oobn int
	var recvErr error
	err = rawConn.Read(func(fd uintptr) bool {
		n, oobn, _, _, recvErr = unix.Recvmsg(int(fd), buf, oob, 0)
		return recvErr != unix.EAGAIN && recvErr != unix.EWOULDBLOCK
	})
	if err != nil {
		return -1, err
	}
	if recvErr != nil {
		return -1, recvErr
	}
	_ = n

	cmsgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, fmt.Errorf("parse cmsg: %w", err)
	}
	for _, c := range cmsgs {
		if c.Header.Level == unix.SOL_SOCKET && c.Header.Type == unix.SCM_RIGHTS {
			fds, err := unix.ParseUnixRights(&c)
			if err != nil {
				return -1, err
			}
			if len(fds) > 0 {
				return fds[0], nil
			}
		}
	}
	return -1, fmt.Errorf("sandbox: no fd received in SCM_RIGHTS")
}
