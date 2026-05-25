// bind-mount demonstrates a bind mount without root by combining a user
// namespace (CLONE_NEWUSER) with a mount namespace (CLONE_NEWNS).  Inside the
// user namespace the process appears as uid 0, which grants the CAP_SYS_ADMIN
// needed for mount(2).
//
// The program re-executes itself via /proc/self/exe.  The first run is the
// parent; the re-exec is the child (detected by the --child flag).
//
// Usage: bind-mount <src> <dst>
//
// Example:
//
//	bind-mount /tmp/src /tmp/dst
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const childFlag = "--child"

func main() {
	if len(os.Args) == 4 && os.Args[1] == childFlag {
		runChild(os.Args[2], os.Args[3])
		return
	}
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <src> <dst>\n", os.Args[0])
		os.Exit(1)
	}
	runParent(os.Args[1], os.Args[2])
}

// runParent re-executes this binary inside a new user+mount namespace.
// UID/GID mappings make the process appear as root (uid 0) inside the
// namespace, satisfying the CAP_SYS_ADMIN check in mount(2).
func runParent(src, dst string) {
	fmt.Printf("parent pid=%d  uid=%d  mntns=%s\n", os.Getpid(), os.Getuid(), nsID("mnt"))

	uid := os.Getuid()
	gid := os.Getgid()

	cmd := exec.Command("/proc/self/exe", childFlag, src, dst)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: uid, Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: gid, Size: 1},
		},
	}
	if err := cmd.Run(); err != nil {
		fatalf("parent: %v", err)
	}
}

// runChild is invoked in the new namespace.  It bind-mounts src onto dst and
// lists the contents of dst to prove the mount succeeded.
func runChild(src, dst string) {
	fmt.Printf("child  pid=%d  uid=%d  mntns=%s\n", os.Getpid(), os.Getuid(), nsID("mnt"))

	// MS_BIND|MS_REC does a recursive bind mount.  MS_REC is required when the
	// source contains locked submounts (created in a more privileged namespace):
	// a plain MS_BIND returns EINVAL in that case.
	if err := syscall.Mount(src, dst, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		fatalf("mount %s -> %s: %v", src, dst, err)
	}
	fmt.Printf("bind-mounted %s -> %s\n", src, dst)

	cmd := exec.Command("ls", "-la", dst)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatalf("ls: %v", err)
	}
}

// nsID returns the inode of a namespace file as a compact identifier.
func nsID(ns string) string {
	info, err := os.Stat(fmt.Sprintf("/proc/self/ns/%s", ns))
	if err != nil {
		return "?"
	}
	return fmt.Sprintf("[%d]", info.Sys().(*syscall.Stat_t).Ino)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

