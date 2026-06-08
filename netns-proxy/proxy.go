package main

import (
	"fmt"
	"io"
	"net"
	"net/netip"
	"sort"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// proxy is a deliberately simple TCP proxy: it accepts the connections that
// the namespace's REDIRECT rule funnels to it, recovers each connection's
// real (pre-redirect) destination, records it, and sends the client a short
// banner naming the destination it "reached" before closing. A real proxy
// would dial out to that destination instead -- there's nowhere to dial out
// *to* from this namespace, which is exactly the point of the example: every
// connection the command makes is observed here, and nowhere else.
type proxy struct {
	mu      sync.Mutex
	records []connRecord
}

type connRecord struct {
	when time.Time
	src  netip.AddrPort // the command's local address:port for this connection
	dst  netip.AddrPort // where the command was actually trying to connect
}

func newProxy() *proxy {
	return &proxy{}
}

func (p *proxy) serve(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go p.handle(c.(*net.TCPConn))
	}
}

func (p *proxy) handle(conn *net.TCPConn) {
	defer conn.Close()

	dst, err := originalDestination(conn)
	if err != nil {
		fmt.Printf("[proxy] could not recover original destination: %v\n", err)
		return
	}
	src, _ := netip.ParseAddrPort(conn.RemoteAddr().String())

	p.mu.Lock()
	p.records = append(p.records, connRecord{when: time.Now(), src: src, dst: dst})
	p.mu.Unlock()

	fmt.Printf("[proxy] intercepted %s -> %s\n", src, dst)
	fmt.Fprintf(conn, "netns-proxy: intercepted your connection to %s\n", dst)
	io.Copy(io.Discard, conn) // drain anything the client sends, then let defer close it
}

// printSummary lists every distinct address the command tried to reach, and
// every local port it used to do so -- i.e. every IP address and port that
// were involved in the command's network activity, all observed at the
// proxy because nothing could get past it.
func (p *proxy) printSummary(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fmt.Fprintf(w, "\nnetns-proxy: %d connection(s) intercepted\n", len(p.records))
	if len(p.records) == 0 {
		return
	}

	destinations := map[netip.AddrPort]int{}
	destAddrs := map[netip.Addr]bool{}
	localPorts := map[uint16]bool{}

	fmt.Fprintln(w, "\n  when                           local (command)        -> remote (destination it tried to reach)")
	for _, r := range p.records {
		fmt.Fprintf(w, "  %-30s %-22s -> %s\n", r.when.Format(time.RFC3339Nano), r.src, r.dst)
		destinations[r.dst]++
		destAddrs[r.dst.Addr()] = true
		localPorts[r.src.Port()] = true
	}

	fmt.Fprintln(w, "\n  remote IP addresses the command tried to reach:")
	for _, a := range sortedAddrs(destAddrs) {
		fmt.Fprintf(w, "    %s\n", a)
	}

	fmt.Fprintln(w, "\n  remote address:port pairs the command tried to reach:")
	for _, ap := range sortedAddrPorts(destinations) {
		fmt.Fprintf(w, "    %-22s (%d connection(s))\n", ap, destinations[ap])
	}

	fmt.Fprintf(w, "\n  local ports the command connected from (all on %s, the namespace's loopback):\n", p.records[0].src.Addr())
	for _, port := range sortedPorts(localPorts) {
		fmt.Fprintf(w, "    %d\n", port)
	}
}

func sortedAddrs(m map[netip.Addr]bool) []netip.Addr {
	out := make([]netip.Addr, 0, len(m))
	for a := range m {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func sortedAddrPorts(m map[netip.AddrPort]int) []netip.AddrPort {
	out := make([]netip.AddrPort, 0, len(m))
	for ap := range m {
		out = append(out, ap)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func sortedPorts(m map[uint16]bool) []uint16 {
	out := make([]uint16, 0, len(m))
	for p := range m {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// soOriginalDst is SO_ORIGINAL_DST (linux/netfilter_ipv4.h): the getsockopt
// that recovers a redirected connection's destination as it was *before*
// netfilter rewrote it. It's how transparent proxies discover what their
// clients actually wanted.
const soOriginalDst = 80

// sockaddrIn mirrors struct sockaddr_in, which is what SO_ORIGINAL_DST fills
// in for an AF_INET socket: 2-byte family, 2-byte port (network byte order),
// 4-byte address, 8 bytes of padding to match struct sockaddr's size.
type sockaddrIn struct {
	Family uint16
	Port   uint16
	Addr   [4]byte
	_      [8]byte
}

func originalDestination(conn *net.TCPConn) (netip.AddrPort, error) {
	sc, err := conn.SyscallConn()
	if err != nil {
		return netip.AddrPort{}, err
	}

	var addr sockaddrIn
	size := uint32(unsafe.Sizeof(addr))
	var sockErr error
	ctlErr := sc.Control(func(fd uintptr) {
		_, _, errno := syscall.Syscall6(syscall.SYS_GETSOCKOPT, fd,
			uintptr(syscall.SOL_IP), uintptr(soOriginalDst),
			uintptr(unsafe.Pointer(&addr)), uintptr(unsafe.Pointer(&size)), 0)
		if errno != 0 {
			sockErr = errno
		}
	})
	if ctlErr != nil {
		return netip.AddrPort{}, ctlErr
	}
	if sockErr != nil {
		return netip.AddrPort{}, sockErr
	}

	port := (addr.Port >> 8) | (addr.Port << 8) // network -> host byte order
	return netip.AddrPortFrom(netip.AddrFrom4(addr.Addr), port), nil
}
