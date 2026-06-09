// Command passt is a tiny, self-contained reimplementation of the core idea
// behind passt/pasta (https://passt.top): give an isolated network namespace
// connectivity to the outside world without any privileged host configuration
// (no bridges, no veth, no NAT rules).
//
// It works like pasta: it creates a user+network namespace, makes a TAP device
// inside it, configures an address and a default route, then runs a command in
// the namespace. The parent process keeps the other end of the TAP and runs a
// small userspace TCP/IP stack that translates the Layer-2 Ethernet frames
// coming out of the namespace into ordinary Layer-4 sockets on the host:
//
//   - ARP requests for the gateway are answered locally.
//   - ICMP echo to the gateway is answered locally (so `ping <gw>` works).
//   - Outbound TCP connections are proxied to real host sockets.
//   - Outbound UDP datagrams are forwarded via real host sockets (so DNS works).
//
// Because the namespace reaches everything through the gateway, the host needs
// no special privileges beyond what an unprivileged user namespace already
// grants over its own netns.
//
// Usage:
//
//	passt [flags] [command [args...]]
//
// With no command it runs $SHELL (or /bin/sh). Example:
//
//	passt curl -sS https://example.com
//	passt            # interactive shell with proxied networking
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// childEnv names an environment variable that, when set, tells a re-executed
// copy of this binary that it is the namespace child. Its value is the file
// descriptor number of the Unix socket used to hand the TAP fd back to the
// parent.
const childEnv = "_PASTA_CHILD_FD"

type config struct {
	tapName string
	guestIP net.IP // address assigned inside the namespace
	gwIP    net.IP // the gateway we impersonate
	prefix  int    // subnet prefix length
	mtu     int
	gwMAC   net.HardwareAddr
	cmdline []string
}

func main() {
	if fdStr := os.Getenv(childEnv); fdStr != "" {
		fd, err := strconv.Atoi(fdStr)
		if err != nil {
			fatal(fmt.Errorf("bad %s: %w", childEnv, err))
		}
		runChild(fd)
		return
	}
	runParent()
}

func defaultConfig() config {
	return config{
		tapName: "pasta0",
		guestIP: net.IPv4(10, 0, 0, 2),
		gwIP:    net.IPv4(10, 0, 0, 1),
		prefix:  24,
		mtu:     1500,
		// Locally administered, unicast MAC for the gateway we impersonate.
		gwMAC: net.HardwareAddr{0x52, 0x54, 0x00, 0x00, 0x00, 0x01},
	}
}

// runParent sets up the namespace child, receives the TAP fd from it, and runs
// the userspace network stack until the child exits.
func runParent() {
	cfg := defaultConfig()

	var guestStr, gwStr string
	var prefix, mtu int
	flag.StringVar(&cfg.tapName, "ifname", cfg.tapName, "TAP interface name inside the namespace")
	flag.StringVar(&guestStr, "address", cfg.guestIP.String(), "address assigned inside the namespace")
	flag.StringVar(&gwStr, "gateway", cfg.gwIP.String(), "gateway address (impersonated by the stack)")
	flag.IntVar(&prefix, "prefix", cfg.prefix, "subnet prefix length")
	flag.IntVar(&mtu, "mtu", cfg.mtu, "interface MTU")
	flag.Parse()

	cfg.guestIP = mustIP(guestStr)
	cfg.gwIP = mustIP(gwStr)
	cfg.prefix = prefix
	cfg.mtu = mtu
	cfg.cmdline = flag.Args()
	if len(cfg.cmdline) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cfg.cmdline = []string{shell}
	}

	// A Unix socket pair: the child sends the TAP fd back to us over it.
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		fatal(fmt.Errorf("socketpair: %w", err))
	}
	parentSock := os.NewFile(uintptr(fds[0]), "parent-sock")
	childSock := os.NewFile(uintptr(fds[1]), "child-sock")

	cmd := exec.Command("/proc/self/exe", childArgs(cfg)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// childSock becomes fd 3 in the child.
	cmd.ExtraFiles = []*os.File{childSock}
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=3", childEnv))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		fatal(fmt.Errorf("start child: %w", err))
	}
	childSock.Close()

	tapFD, err := recvFD(parentSock)
	if err != nil {
		fatal(fmt.Errorf("receive TAP fd: %w", err))
	}
	parentSock.Close()

	stack, err := newStack(tapFD, cfg)
	if err != nil {
		fatal(fmt.Errorf("create stack: %w", err))
	}
	go stack.run()

	err = cmd.Wait()
	stack.close()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		fatal(err)
	}
}

// childArgs encodes the configuration the child needs as command-line args.
func childArgs(cfg config) []string {
	args := []string{
		"--ifname=" + cfg.tapName,
		"--address=" + cfg.guestIP.String(),
		"--gateway=" + cfg.gwIP.String(),
		"--prefix=" + strconv.Itoa(cfg.prefix),
		"--mtu=" + strconv.Itoa(cfg.mtu),
	}
	return append(append(args, "--"), cfg.cmdline...)
}

// runChild runs inside the new user+network namespace. It creates and
// configures the TAP device, hands its fd back to the parent, and execs the
// requested command.
func runChild(sockFD int) {
	// Reparse the same flags the parent set; cmdline follows a "--".
	cfg := defaultConfig()
	var guestStr, gwStr string
	fs := flag.NewFlagSet("child", flag.ExitOnError)
	fs.StringVar(&cfg.tapName, "ifname", cfg.tapName, "")
	fs.StringVar(&guestStr, "address", cfg.guestIP.String(), "")
	fs.StringVar(&gwStr, "gateway", cfg.gwIP.String(), "")
	fs.IntVar(&cfg.prefix, "prefix", cfg.prefix, "")
	fs.IntVar(&cfg.mtu, "mtu", cfg.mtu, "")
	fs.Parse(os.Args[1:])
	cfg.guestIP = mustIP(guestStr)
	cfg.gwIP = mustIP(gwStr)
	cfg.cmdline = fs.Args()

	tapFD, err := createTAP(cfg.tapName)
	if err != nil {
		fatal(fmt.Errorf("create TAP: %w", err))
	}
	if err := configureNamespace(cfg); err != nil {
		fatal(fmt.Errorf("configure namespace: %w", err))
	}

	if err := sendFD(sockFD, tapFD); err != nil {
		fatal(fmt.Errorf("send TAP fd: %w", err))
	}
	unix.Close(tapFD)
	unix.Close(sockFD)

	path, err := exec.LookPath(cfg.cmdline[0])
	if err != nil {
		fatal(fmt.Errorf("lookup %q: %w", cfg.cmdline[0], err))
	}
	if err := syscall.Exec(path, cfg.cmdline, os.Environ()); err != nil {
		fatal(fmt.Errorf("exec %q: %w", path, err))
	}
}

// recvFD receives a single file descriptor sent over the Unix socket.
func recvFD(sock *os.File) (int, error) {
	buf := make([]byte, 1)
	oob := make([]byte, unix.CmsgSpace(4))
	n, oobn, _, _, err := unix.Recvmsg(int(sock.Fd()), buf, oob, 0)
	if err != nil {
		return -1, err
	}
	if n == 0 {
		return -1, fmt.Errorf("child closed socket without sending fd")
	}
	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, err
	}
	if len(msgs) == 0 {
		return -1, fmt.Errorf("no control message")
	}
	fds, err := unix.ParseUnixRights(&msgs[0])
	if err != nil {
		return -1, err
	}
	if len(fds) == 0 {
		return -1, fmt.Errorf("no fd in control message")
	}
	return fds[0], nil
}

// sendFD sends a single file descriptor over the Unix socket fd.
func sendFD(sockFD, fd int) error {
	rights := unix.UnixRights(fd)
	return unix.Sendmsg(sockFD, []byte{0}, rights, nil, 0)
}

func mustIP(s string) net.IP {
	ip := net.ParseIP(s).To4()
	if ip == nil {
		fatal(fmt.Errorf("invalid IPv4 address: %q", s))
	}
	return ip
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "passt:", err)
	os.Exit(1)
}
