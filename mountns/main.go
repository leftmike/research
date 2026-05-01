// mountns demonstrates how an unprivileged process can create a mount namespace
// by pairing CLONE_NEWNS with CLONE_NEWUSER. The new user namespace grants the
// process full capabilities within its scope, which is enough to perform mounts.
//
// Usage: just run the binary — no root required.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		runChild()
		return
	}
	runParent()
}

// runParent re-executes the binary inside a new user+mount namespace.
func runParent() {
	fmt.Printf("parent: pid=%d  mnt-ns=%s\n", os.Getpid(), nsID("mnt"))

	cmd := exec.Command("/proc/self/exe", "child")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// CLONE_NEWUSER is required for unprivileged namespace creation.
		// CLONE_NEWNS creates the new mount namespace.
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
	}

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runChild runs inside the new namespaces and proves it can mount.
func runChild() {
	fmt.Printf("child:  pid=%d  mnt-ns=%s  uid=%d\n",
		os.Getpid(), nsID("mnt"), os.Getuid())

	dir, err := os.MkdirTemp("", "mountns-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.Remove(dir)

	if err := syscall.Mount("tmpfs", dir, "tmpfs", 0, ""); err != nil {
		fmt.Fprintln(os.Stderr, "mount:", err)
		os.Exit(1)
	}
	fmt.Printf("child:  mounted tmpfs on %s (invisible to parent)\n", dir)

	syscall.Unmount(dir, 0)
}

// nsID returns the kernel identifier for the named namespace of the current process.
func nsID(ns string) string {
	link, err := os.Readlink("/proc/self/ns/" + ns)
	if err != nil {
		return "?"
	}
	return link
}
