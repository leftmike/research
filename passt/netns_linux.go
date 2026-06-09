package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"unsafe"

	"golang.org/x/sys/unix"
)

// createTAP opens /dev/net/tun and creates a non-persistent TAP device with the
// given name. The returned fd reads and writes raw Ethernet frames. The device
// lives in whatever network namespace is current when this is called, and stays
// alive as long as the fd is open.
func createTAP(name string) (int, error) {
	fd, err := unix.Open("/dev/net/tun", unix.O_RDWR, 0)
	if err != nil {
		return -1, fmt.Errorf("open /dev/net/tun: %w", err)
	}

	var req ifreq
	copy(req.name[:], name)
	// IFF_TAP: Ethernet frames. IFF_NO_PI: no 4-byte packet-info prefix.
	binary.LittleEndian.PutUint16(req.data[:2], unix.IFF_TAP|unix.IFF_NO_PI)
	if err := ioctl(fd, unix.TUNSETIFF, unsafe.Pointer(&req)); err != nil {
		unix.Close(fd)
		return -1, fmt.Errorf("TUNSETIFF: %w", err)
	}
	return fd, nil
}

// configureNamespace assigns the address, MTU and default route inside the
// current (child) network namespace, and brings loopback up.
func configureNamespace(cfg config) error {
	if err := setLinkUp("lo"); err != nil {
		return fmt.Errorf("lo up: %w", err)
	}
	if err := setMTU(cfg.tapName, cfg.mtu); err != nil {
		return fmt.Errorf("set mtu: %w", err)
	}
	if err := setAddr(cfg.tapName, cfg.guestIP, cfg.prefix); err != nil {
		return fmt.Errorf("set address: %w", err)
	}
	if err := setLinkUp(cfg.tapName); err != nil {
		return fmt.Errorf("link up: %w", err)
	}
	idx, err := ifIndex(cfg.tapName)
	if err != nil {
		return fmt.Errorf("ifindex: %w", err)
	}
	if err := addDefaultRoute(cfg.gwIP, idx); err != nil {
		return fmt.Errorf("default route: %w", err)
	}
	return nil
}

// ifreq mirrors the kernel struct ifreq: a 16-byte interface name followed by a
// union we access as raw bytes.
type ifreq struct {
	name [unix.IFNAMSIZ]byte
	data [24]byte
}

func ioctl(fd int, request uintptr, arg unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), request, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// withSocket runs fn with a temporary AF_INET socket usable for interface
// ioctls.
func withSocket(fn func(fd int) error) error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	return fn(fd)
}

func setLinkUp(name string) error {
	return withSocket(func(fd int) error {
		var req ifreq
		copy(req.name[:], name)
		if err := ioctl(fd, unix.SIOCGIFFLAGS, unsafe.Pointer(&req)); err != nil {
			return err
		}
		flags := binary.LittleEndian.Uint16(req.data[:2])
		flags |= unix.IFF_UP | unix.IFF_RUNNING
		binary.LittleEndian.PutUint16(req.data[:2], flags)
		return ioctl(fd, unix.SIOCSIFFLAGS, unsafe.Pointer(&req))
	})
}

func setMTU(name string, mtu int) error {
	return withSocket(func(fd int) error {
		var req ifreq
		copy(req.name[:], name)
		binary.LittleEndian.PutUint32(req.data[:4], uint32(mtu))
		return ioctl(fd, unix.SIOCSIFMTU, unsafe.Pointer(&req))
	})
}

func ifIndex(name string) (int, error) {
	var idx int
	err := withSocket(func(fd int) error {
		var req ifreq
		copy(req.name[:], name)
		if err := ioctl(fd, unix.SIOCGIFINDEX, unsafe.Pointer(&req)); err != nil {
			return err
		}
		idx = int(int32(binary.LittleEndian.Uint32(req.data[:4])))
		return nil
	})
	return idx, err
}

func setAddr(name string, ip net.IP, prefix int) error {
	return withSocket(func(fd int) error {
		// SIOCSIFADDR with a sockaddr_in in the ifreq union.
		var req ifreq
		copy(req.name[:], name)
		putSockaddrIn(req.data[:], ip)
		if err := ioctl(fd, unix.SIOCSIFADDR, unsafe.Pointer(&req)); err != nil {
			return err
		}
		// SIOCSIFNETMASK with the mask derived from the prefix length.
		var maskReq ifreq
		copy(maskReq.name[:], name)
		mask := net.CIDRMask(prefix, 32)
		putSockaddrIn(maskReq.data[:], net.IP(mask))
		return ioctl(fd, unix.SIOCSIFNETMASK, unsafe.Pointer(&maskReq))
	})
}

// putSockaddrIn writes a struct sockaddr_in (AF_INET, port 0, addr) into b.
func putSockaddrIn(b []byte, ip net.IP) {
	ip4 := ip.To4()
	// sa_family is a host-byte-order uint16 in struct sockaddr.
	binary.LittleEndian.PutUint16(b[0:2], unix.AF_INET)
	b[2], b[3] = 0, 0 // port
	copy(b[4:8], ip4)
}

// addDefaultRoute installs a default route (0.0.0.0/0) via gw out of interface
// index idx, using a minimal rtnetlink request.
func addDefaultRoute(gw net.IP, idx int) error {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, unix.NETLINK_ROUTE)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	if err := unix.Bind(fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return err
	}

	// struct rtmsg followed by RTA_GATEWAY and RTA_OIF attributes.
	rtmsg := make([]byte, unix.SizeofRtMsg)
	rtmsg[0] = unix.AF_INET           // rtm_family
	rtmsg[4] = unix.RT_TABLE_MAIN     // rtm_table
	rtmsg[5] = unix.RTPROT_BOOT       // rtm_protocol
	rtmsg[6] = unix.RT_SCOPE_UNIVERSE // rtm_scope
	rtmsg[7] = unix.RTN_UNICAST       // rtm_type

	payload := append([]byte{}, rtmsg...)
	payload = append(payload, rtAttr(unix.RTA_GATEWAY, gw.To4())...)
	oif := make([]byte, 4)
	binary.LittleEndian.PutUint32(oif, uint32(idx))
	payload = append(payload, rtAttr(unix.RTA_OIF, oif)...)

	msg := make([]byte, unix.SizeofNlMsghdr+len(payload))
	binary.LittleEndian.PutUint32(msg[0:4], uint32(len(msg)))  // nlmsg_len
	binary.LittleEndian.PutUint16(msg[4:6], unix.RTM_NEWROUTE) // nlmsg_type
	binary.LittleEndian.PutUint16(msg[6:8], unix.NLM_F_REQUEST|unix.NLM_F_CREATE|unix.NLM_F_ACK)
	binary.LittleEndian.PutUint32(msg[8:12], 1) // nlmsg_seq
	copy(msg[unix.SizeofNlMsghdr:], payload)

	if err := unix.Sendto(fd, msg, 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return err
	}
	return readNetlinkAck(fd)
}

// rtAttr builds a single netlink route attribute (rtattr) with the given type
// and value, padded to a 4-byte boundary.
func rtAttr(typ uint16, value []byte) []byte {
	const hdr = 4 // sizeof(struct rtattr)
	l := hdr + len(value)
	b := make([]byte, (l+3)&^3)
	binary.LittleEndian.PutUint16(b[0:2], uint16(l))
	binary.LittleEndian.PutUint16(b[2:4], typ)
	copy(b[hdr:], value)
	return b
}

// readNetlinkAck reads the ACK for a request and returns any error it reports.
// It parses the nlmsghdr stream by hand to avoid depending on helpers that vary
// between library versions.
func readNetlinkAck(fd int) error {
	buf := make([]byte, 4096)
	n, err := unix.Read(fd, buf)
	if err != nil {
		return err
	}
	b := buf[:n]
	for len(b) >= unix.SizeofNlMsghdr {
		msgLen := int(binary.LittleEndian.Uint32(b[0:4]))
		msgType := binary.LittleEndian.Uint16(b[4:6])
		if msgLen < unix.SizeofNlMsghdr || msgLen > len(b) {
			break
		}
		if msgType == unix.NLMSG_ERROR {
			// The error code is the first int32 after the header; 0 == ACK.
			errno := int32(binary.LittleEndian.Uint32(b[unix.SizeofNlMsghdr : unix.SizeofNlMsghdr+4]))
			if errno != 0 {
				return fmt.Errorf("netlink: %w", unix.Errno(-errno))
			}
			return nil
		}
		b = b[(msgLen+3)&^3:]
	}
	return nil
}
