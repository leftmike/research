package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// This file talks directly to the kernel over netlink(7) and ioctl(2) to
// configure the namespace's network stack -- the equivalent of running
//
//	ip link set lo up
//	ip route add default dev lo scope link src 127.0.0.1
//
// without depending on the "ip" tool (which, notably, this minimal namespace
// can't reach anyway, and may not even be installed on the host).

// ifreqFlags mirrors enough of Linux's struct ifreq for SIOC{G,S}IFFLAGS:
// a 16-byte interface name unioned with assorted fields, of which we only
// care about the flags (a 16-bit field at the start of the union). The
// struct is 40 bytes on amd64; the trailing bytes are padding we never read.
type ifreqFlags struct {
	Name  [syscall.IFNAMSIZ]byte
	Flags uint16
	_     [40 - syscall.IFNAMSIZ - 2]byte
}

func ioctl(fd int, req uintptr, ifr *ifreqFlags) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(unsafe.Pointer(ifr))); errno != 0 {
		return errno
	}
	return nil
}

// Routing attribute types and route parameters used below, from
// linux/rtnetlink.h. RTA_PREFSRC picks the source address the kernel uses
// for connections that match the route; without it, connections routed
// through a gateway-less ("onlink") route pick INADDR_ANY as their source,
// which breaks the loopback 3-way handshake.
const (
	rtaOif     = 4 // route output interface index
	rtaPrefSrc = 7 // route preferred source address

	rtTableMain = 254 // RT_TABLE_MAIN
	rtProtBoot  = 3   // RTPROT_BOOT: route installed during startup
	rtScopeLink = 253 // RT_SCOPE_LINK: reachable directly, no gateway
	rtnUnicast  = 1   // RTN_UNICAST
)

type nlMsgHdr struct {
	Len   uint32
	Type  uint16
	Flags uint16
	Seq   uint32
	Pid   uint32
}

type rtMsg struct {
	Family   uint8
	DstLen   uint8
	SrcLen   uint8
	Tos      uint8
	Table    uint8
	Protocol uint8
	Scope    uint8
	Type     uint8
	Flags    uint32
}

func align4(n int) int { return (n + 3) &^ 3 }

// appendAttr appends a netlink attribute (rtattr header + value, padded to a
// 4-byte boundary) to buf.
func appendAttr(buf []byte, atype uint16, value []byte) []byte {
	var hdr [4]byte
	binary.LittleEndian.PutUint16(hdr[0:2], uint16(4+len(value)))
	binary.LittleEndian.PutUint16(hdr[2:4], atype)
	buf = append(buf, hdr[:]...)
	buf = append(buf, value...)
	for len(buf)%4 != 0 {
		buf = append(buf, 0)
	}
	return buf
}

// addOnlinkDefaultRoute adds a "default route via dev, no gateway, with the
// given preferred source address" -- equivalent to
//
//	ip route add default dev <dev> scope link src <src>
//
// This is what makes every destination address look reachable from inside
// the namespace: the kernel is willing to hand connect()-ed packets to "dev"
// (loopback, in our case) for any destination, which is the prerequisite for
// netfilter's REDIRECT target ever getting a chance to run. Without *some*
// matching route, connect() fails immediately with ENETUNREACH and the
// packet never reaches netfilter at all.
func addOnlinkDefaultRoute(dev string, src net.IP) error {
	iface, err := net.InterfaceByName(dev)
	if err != nil {
		return err
	}
	src4 := src.To4()
	if src4 == nil {
		return fmt.Errorf("preferred source %s is not an IPv4 address", src)
	}

	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_ROUTE)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	rt := rtMsg{
		Family:   syscall.AF_INET,
		Table:    rtTableMain,
		Protocol: rtProtBoot,
		Scope:    rtScopeLink,
		Type:     rtnUnicast,
	}
	body := (*[unsafe.Sizeof(rt)]byte)(unsafe.Pointer(&rt))[:]

	var attrs []byte
	var oif [4]byte
	binary.LittleEndian.PutUint32(oif[:], uint32(iface.Index))
	attrs = appendAttr(attrs, rtaOif, oif[:])
	attrs = appendAttr(attrs, rtaPrefSrc, src4)

	msgBody := append(append([]byte{}, body...), attrs...)

	hdr := nlMsgHdr{
		Type:  syscall.RTM_NEWROUTE,
		Flags: syscall.NLM_F_REQUEST | syscall.NLM_F_CREATE | syscall.NLM_F_EXCL | syscall.NLM_F_ACK,
		Seq:   1,
	}
	hdr.Len = uint32(align4(int(unsafe.Sizeof(hdr))) + len(msgBody))
	hdrBytes := (*[unsafe.Sizeof(hdr)]byte)(unsafe.Pointer(&hdr))[:]

	msg := append(append([]byte{}, hdrBytes...), msgBody...)
	if err := syscall.Sendto(fd, msg, 0, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}); err != nil {
		return err
	}
	return recvNetlinkAck(fd)
}

// recvNetlinkAck reads a single netlink response and turns an NLMSG_ERROR
// with a non-zero error code into a Go error. The kernel acks successful
// NLM_F_ACK requests with an NLMSG_ERROR carrying error code 0.
func recvNetlinkAck(fd int) error {
	buf := make([]byte, 4096)
	n, _, err := syscall.Recvfrom(fd, buf, 0)
	if err != nil {
		return err
	}
	buf = buf[:n]

	for len(buf) >= int(unsafe.Sizeof(nlMsgHdr{})) {
		var h nlMsgHdr
		h.Len = binary.LittleEndian.Uint32(buf[0:4])
		h.Type = binary.LittleEndian.Uint16(buf[4:6])
		if h.Len < uint32(unsafe.Sizeof(h)) || int(h.Len) > len(buf) {
			break
		}
		if h.Type == syscall.NLMSG_ERROR {
			errno := int32(binary.LittleEndian.Uint32(buf[16:20]))
			if errno == 0 {
				return nil
			}
			return syscall.Errno(-errno)
		}
		buf = buf[align4(int(h.Len)):]
	}
	return nil
}
