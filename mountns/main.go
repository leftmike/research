// mountns demonstrates how an unprivileged process can create a mount namespace
// by pairing CLONE_NEWNS with CLONE_NEWUSER. The new user namespace grants the
// process full capabilities within its scope, which is enough to perform mounts.
//
// Usage: mountns <cmdline> [dir...]
//
//	cmdline  command (and optional arguments) to run inside the namespace
//	dir...   absolute directories to bind-mount into the namespace root
//
// Example:
//
//	mountns "/bin/sh -i" /bin /lib /lib64 /usr
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const childFlag = "--child"

func main() {
	if len(os.Args) >= 2 && os.Args[1] == childFlag {
		runChildFromArgs(os.Args[2:])
		return
	}
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <cmdline> [dir...]\n", os.Args[0])
		os.Exit(1)
	}
	runParent(strings.Fields(os.Args[1]), os.Args[2:])
}

// runParent creates a temp directory for the new root, then re-executes the
// binary inside a new user+mount namespace.
func runParent(cmdArgs, dirs []string) {
	fmt.Fprintf(os.Stderr, "parent: pid=%d  mnt-ns=%s\n", os.Getpid(), nsID("mnt"))

	// The child mounts a tmpfs here; from the parent's view it stays empty,
	// so we can rmdir it cleanly once the child exits.
	newRoot, err := os.MkdirTemp("", "mountns-*")
	if err != nil {
		fatal(err)
	}
	defer os.Remove(newRoot)

	// Pass newRoot and dirs to the child process via argv, separated by "--".
	args := []string{childFlag, newRoot}
	args = append(args, cmdArgs...)
	args = append(args, "--")
	args = append(args, dirs...)

	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// CLONE_NEWUSER is required for unprivileged namespace creation.
		// CLONE_NEWNS creates the new mount namespace.
		// CLONE_NEWPID is needed so we can mount a fresh proc inside the namespace.
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
	}

	if err := cmd.Run(); err != nil {
		fatal(err)
	}
}

// runChildFromArgs parses the argv passed by runParent and calls runChild.
// Expected format: newRoot cmd [cmdargs...] -- [dirs...]
func runChildFromArgs(args []string) {
	if len(args) < 2 {
		fatal(fmt.Errorf("internal: malformed child args"))
	}
	newRoot := args[0]
	rest := args[1:]

	sep := -1
	for i, a := range rest {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep < 0 {
		fatal(fmt.Errorf("internal: missing -- separator"))
	}
	runChild(newRoot, rest[:sep], rest[sep+1:])
}

// runChild sets up the mount namespace: a fresh tmpfs root with the requested
// directories bind-mounted in, then execs the command.
func runChild(newRoot string, cmdArgs, dirs []string) {
	fmt.Fprintf(os.Stderr, "child:  pid=%d  mnt-ns=%s  uid=%d\n",
		os.Getpid(), nsID("mnt"), os.Getuid())

	if err := syscall.Mount("tmpfs", newRoot, "tmpfs", 0, ""); err != nil {
		fatal(fmt.Errorf("mount tmpfs: %w", err))
	}

	for _, dir := range dirs {
		dest := filepath.Join(newRoot, dir)
		if err := os.MkdirAll(dest, 0755); err != nil {
			fatal(err)
		}
		if err := syscall.Mount(dir, dest, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
			fatal(fmt.Errorf("bind %s: %w", dir, err))
		}
	}

	// Always provide /proc so the command can inspect its own process.
	procDir := filepath.Join(newRoot, "proc")
	if err := os.MkdirAll(procDir, 0555); err != nil {
		fatal(err)
	}
	if err := syscall.Mount("proc", procDir, "proc", 0, ""); err != nil {
		fatal(fmt.Errorf("mount proc: %w", err))
	}

	if err := syscall.Chroot(newRoot); err != nil {
		fatal(fmt.Errorf("chroot: %w", err))
	}
	if err := os.Chdir("/"); err != nil {
		fatal(err)
	}

	if err := syscall.Exec(cmdArgs[0], cmdArgs, os.Environ()); err != nil {
		fatal(fmt.Errorf("exec %s: %w", cmdArgs[0], err))
	}
}

// nsID returns the kernel identifier for the named namespace of the current process.
func nsID(ns string) string {
	link, err := os.Readlink("/proc/self/ns/" + ns)
	if err != nil {
		return "?"
	}
	return link
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
