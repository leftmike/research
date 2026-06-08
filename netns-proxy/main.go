// netns-proxy runs a command inside a fresh, unprivileged network namespace
// that has no connectivity of its own: the only thing it can reach is a
// loopback-bound proxy that this program also runs. A transparent-redirect
// firewall rule forces every outbound TCP connection the command makes,
// regardless of destination, through that proxy. The proxy records the
// original destination of each connection (recovered via SO_ORIGINAL_DST),
// so at the end we can print every IP address and port the command tried to
// use.
//
// No root is required: pairing CLONE_NEWUSER with CLONE_NEWNET lets an
// unprivileged user create a network namespace in which it appears as uid 0
// and therefore holds CAP_NET_ADMIN/CAP_NET_RAW over that namespace's network
// stack -- enough to bring up loopback, add routes and install netfilter
// rules, none of which can reach outside the namespace.
//
// Usage:
//
//	netns-proxy <command> [args...]
//
// Example:
//
//	netns-proxy curl http://203.0.113.5/ http://198.51.100.9:8080/
//
// (203.0.113.0/24 and 198.51.100.0/24 are the documentation/example ranges
// from RFC 5737 -- safe stand-ins for "the internet" since nothing routes to
// them; the proxy intercepts the connections before they ever leave the box.)
package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const (
	childFlag = "--child"

	// proxyPort is where the in-namespace proxy listens on loopback. Every
	// outbound TCP connection the command makes is transparently redirected
	// here, whatever destination it asked for.
	proxyPort = 1080
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == childFlag {
		runChild(os.Args[2:])
		return
	}
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command> [args...]\n", os.Args[0])
		os.Exit(2)
	}
	runParent(os.Args[1:])
}

// runParent re-executes this binary inside a new user+network namespace.
// The uid/gid mappings make the re-exec'd process appear as root inside the
// namespace, which is what grants it the capabilities needed to configure
// the (otherwise empty) network stack it now owns.
func runParent(cmdArgs []string) {
	uid, gid := os.Getuid(), os.Getgid()

	args := append([]string{childFlag}, cmdArgs...)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: uid, Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: gid, Size: 1},
		},
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fatal(err)
	}
}

// runChild sets up the namespace's network stack, starts the proxy, runs the
// requested command against it, and reports what the command talked to.
//
// The namespace starts out completely empty: no interfaces but a down
// loopback, no routes, no firewall rules. We:
//
//  1. bring loopback up,
//  2. add a default route through loopback so that *any* destination address
//     looks routable (without this, connect() to a non-loopback address fails
//     immediately with "network is unreachable" and never reaches netfilter),
//  3. install an iptables REDIRECT rule that rewrites every outbound TCP
//     connection's destination to our proxy's loopback address,
//  4. start the proxy, which recovers each connection's pre-redirect
//     destination via SO_ORIGINAL_DST and records it.
func runChild(cmdArgs []string) {
	if len(cmdArgs) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s <command> [args...]\n", os.Args[0])
		os.Exit(2)
	}

	fmt.Fprintf(os.Stderr, "netns-proxy: pid=%d uid=%d net-ns=%s\n",
		os.Getpid(), os.Getuid(), nsID("net"))

	if err := bringLoopbackUp(); err != nil {
		fatal(fmt.Errorf("bring up loopback: %w", err))
	}
	if err := addOnlinkDefaultRoute("lo", net.IPv4(127, 0, 0, 1)); err != nil {
		fatal(fmt.Errorf("add default route: %w", err))
	}
	if err := redirectOutboundTCP(proxyPort); err != nil {
		fatal(fmt.Errorf("install redirect rule: %w", err))
	}

	p := newProxy()
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(proxyPort))
	if err != nil {
		fatal(fmt.Errorf("start proxy: %w", err))
	}
	go p.serve(ln)

	fmt.Fprintf(os.Stderr, "netns-proxy: all outbound TCP is forced through 127.0.0.1:%d\n", proxyPort)
	fmt.Fprintf(os.Stderr, "netns-proxy: running %v\n\n", cmdArgs)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runErr := cmd.Run()

	ln.Close()
	p.printSummary(os.Stderr)

	if exitErr, ok := runErr.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	} else if runErr != nil {
		fatal(runErr)
	}
}

// bringLoopbackUp sets IFF_UP|IFF_RUNNING on "lo" via the classic
// SIOCGIFFLAGS/SIOCSIFFLAGS ioctls -- the same calls "ip link set lo up"
// makes, done directly so the example needs no external tools.
func bringLoopbackUp() error {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	var ifr ifreqFlags
	copy(ifr.Name[:], "lo")

	if err := ioctl(fd, syscall.SIOCGIFFLAGS, &ifr); err != nil {
		return err
	}
	ifr.Flags |= syscall.IFF_UP | syscall.IFF_RUNNING
	return ioctl(fd, syscall.SIOCSIFFLAGS, &ifr)
}

// redirectOutboundTCP installs an iptables nat/OUTPUT rule that rewrites the
// destination of every locally-generated TCP segment to 127.0.0.1:port --
// the standard transparent-proxy trick (the same one tools like redsocks
// rely on). The "! --dport port" exclusion keeps the proxy's own connections
// (should it ever make any) from being looped back into itself.
func redirectOutboundTCP(port int) error {
	p := strconv.Itoa(port)
	cmd := exec.Command("iptables", "-t", "nat", "-A", "OUTPUT",
		"-p", "tcp", "!", "--dport", p,
		"-j", "REDIRECT", "--to-port", p)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

// nsID returns the kernel identifier of the named namespace this process is
// currently in, e.g. "net:[4026532341]".
func nsID(ns string) string {
	link, err := os.Readlink("/proc/self/ns/" + ns)
	if err != nil {
		return "?"
	}
	return link
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "netns-proxy:", err)
	os.Exit(1)
}
