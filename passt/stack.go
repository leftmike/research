package main

import (
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

// Ethernet / IP constants.
const (
	ethHdrLen = 14

	ethTypeIPv4 = 0x0800
	ethTypeARP  = 0x0806

	ipv4HdrLen = 20

	protoICMP = 1
	protoTCP  = 6
	protoUDP  = 17
)

// stack is the userspace network stack driving one TAP device. It demultiplexes
// incoming Ethernet frames and proxies Layer-4 traffic onto host sockets.
type stack struct {
	fd  int
	cfg config

	writeMu sync.Mutex // serializes writes to the TAP fd

	macMu    sync.RWMutex
	guestMAC net.HardwareAddr // learned from inbound frames

	ipID uint32 // counter for IPv4 identification field

	tcp *tcpStack
	udp *udpStack

	closeOnce sync.Once
	closed    chan struct{}
}

func newStack(fd int, cfg config) (*stack, error) {
	s := &stack{
		fd:     fd,
		cfg:    cfg,
		closed: make(chan struct{}),
	}
	s.tcp = newTCPStack(s)
	s.udp = newUDPStack(s)
	return s, nil
}

func (s *stack) close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		unix.Close(s.fd)
	})
}

// run reads frames from the TAP until the device is closed.
func (s *stack) run() {
	buf := make([]byte, 65536)
	for {
		n, err := unix.Read(s.fd, buf)
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			if err == unix.EINTR {
				continue
			}
			return
		}
		if n < ethHdrLen {
			continue
		}
		s.handleFrame(buf[:n])
	}
}

func (s *stack) setGuestMAC(mac net.HardwareAddr) {
	s.macMu.Lock()
	if s.guestMAC == nil {
		s.guestMAC = append(net.HardwareAddr(nil), mac...)
	}
	s.macMu.Unlock()
}

func (s *stack) getGuestMAC() net.HardwareAddr {
	s.macMu.RLock()
	defer s.macMu.RUnlock()
	return s.guestMAC
}

func (s *stack) handleFrame(frame []byte) {
	srcMAC := net.HardwareAddr(frame[6:12])
	ethType := binary.BigEndian.Uint16(frame[12:14])
	payload := frame[ethHdrLen:]

	s.setGuestMAC(srcMAC)

	switch ethType {
	case ethTypeARP:
		s.handleARP(payload)
	case ethTypeIPv4:
		s.handleIPv4(payload)
	}
}

// handleARP answers requests for the gateway address.
func (s *stack) handleARP(p []byte) {
	if len(p) < 28 {
		return
	}
	oper := binary.BigEndian.Uint16(p[6:8])
	if oper != 1 { // only requests
		return
	}
	senderMAC := net.HardwareAddr(p[8:14])
	senderIP := net.IP(p[14:18]).To4()
	targetIP := net.IP(p[24:28]).To4()
	if !targetIP.Equal(s.cfg.gwIP) {
		return
	}

	reply := make([]byte, 28)
	binary.BigEndian.PutUint16(reply[0:2], 1)           // htype: Ethernet
	binary.BigEndian.PutUint16(reply[2:4], ethTypeIPv4) // ptype: IPv4
	reply[4] = 6                                        // hlen
	reply[5] = 4                                        // plen
	binary.BigEndian.PutUint16(reply[6:8], 2)           // oper: reply
	copy(reply[8:14], s.cfg.gwMAC)
	copy(reply[14:18], s.cfg.gwIP.To4())
	copy(reply[18:24], senderMAC)
	copy(reply[24:28], senderIP)

	s.writeEthernet(senderMAC, ethTypeARP, reply)
}

// ipv4Packet is a parsed IPv4 datagram.
type ipv4Packet struct {
	src, dst net.IP
	proto    byte
	payload  []byte
}

func (s *stack) handleIPv4(p []byte) {
	if len(p) < ipv4HdrLen {
		return
	}
	ihl := int(p[0]&0x0f) * 4
	if ihl < ipv4HdrLen || len(p) < ihl {
		return
	}
	total := int(binary.BigEndian.Uint16(p[2:4]))
	if total < ihl || total > len(p) {
		total = len(p)
	}
	pkt := ipv4Packet{
		src:     net.IP(p[12:16]).To4(),
		dst:     net.IP(p[16:20]).To4(),
		proto:   p[9],
		payload: p[ihl:total],
	}

	switch pkt.proto {
	case protoICMP:
		s.handleICMP(pkt)
	case protoTCP:
		s.tcp.handlePacket(pkt)
	case protoUDP:
		s.udp.handlePacket(pkt)
	}
}

// handleICMP answers echo requests addressed to the gateway so that pinging the
// gateway from inside the namespace works.
func (s *stack) handleICMP(pkt ipv4Packet) {
	if !pkt.dst.Equal(s.cfg.gwIP) {
		return
	}
	p := pkt.payload
	if len(p) < 8 || p[0] != 8 { // type 8 == echo request
		return
	}
	reply := append([]byte(nil), p...)
	reply[0] = 0 // echo reply
	reply[2], reply[3] = 0, 0
	csum := checksum(reply, 0)
	binary.BigEndian.PutUint16(reply[2:4], csum)
	// Reply from the gateway back to the sender.
	s.sendIPv4(pkt.dst, pkt.src, protoICMP, reply)
}

// sendIPv4 builds an IPv4 datagram with the given payload and writes it to the
// guest as an Ethernet frame.
func (s *stack) sendIPv4(src, dst net.IP, proto byte, payload []byte) {
	guestMAC := s.getGuestMAC()
	if guestMAC == nil {
		return
	}
	total := ipv4HdrLen + len(payload)
	buf := make([]byte, total)
	buf[0] = 0x45 // version 4, IHL 5
	binary.BigEndian.PutUint16(buf[2:4], uint16(total))
	id := uint16(atomic.AddUint32(&s.ipID, 1))
	binary.BigEndian.PutUint16(buf[4:6], id)
	buf[6] = 0x40 // Don't Fragment
	buf[8] = 64   // TTL
	buf[9] = proto
	copy(buf[12:16], src.To4())
	copy(buf[16:20], dst.To4())
	csum := checksum(buf[:ipv4HdrLen], 0)
	binary.BigEndian.PutUint16(buf[10:12], csum)
	copy(buf[ipv4HdrLen:], payload)

	s.writeEthernet(guestMAC, ethTypeIPv4, buf)
}

// writeEthernet wraps payload in an Ethernet header (to dstMAC, from the
// gateway MAC) and writes the frame to the TAP device.
func (s *stack) writeEthernet(dstMAC net.HardwareAddr, ethType uint16, payload []byte) {
	frame := make([]byte, ethHdrLen+len(payload))
	copy(frame[0:6], dstMAC)
	copy(frame[6:12], s.cfg.gwMAC)
	binary.BigEndian.PutUint16(frame[12:14], ethType)
	copy(frame[ethHdrLen:], payload)

	s.writeMu.Lock()
	unix.Write(s.fd, frame)
	s.writeMu.Unlock()
}

// checksum computes the 16-bit one's-complement checksum of b, folded with the
// supplied initial accumulator (used to carry a pseudo-header sum).
func checksum(b []byte, initial uint32) uint16 {
	sum := initial
	for len(b) >= 2 {
		sum += uint32(b[0])<<8 | uint32(b[1])
		b = b[2:]
	}
	if len(b) == 1 {
		sum += uint32(b[0]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

// pseudoHeaderSum returns the partial checksum of the TCP/UDP pseudo-header.
func pseudoHeaderSum(src, dst net.IP, proto byte, l4len int) uint32 {
	var sum uint32
	s4, d4 := src.To4(), dst.To4()
	sum += uint32(s4[0])<<8 | uint32(s4[1])
	sum += uint32(s4[2])<<8 | uint32(s4[3])
	sum += uint32(d4[0])<<8 | uint32(d4[1])
	sum += uint32(d4[2])<<8 | uint32(d4[3])
	sum += uint32(proto)
	sum += uint32(l4len)
	return sum
}
