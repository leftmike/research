package main

import (
	"encoding/binary"
	"net"
	"sync"
	"time"
)

const udpHdrLen = 8

// udpIdleTimeout is how long a UDP flow may sit idle before it is torn down.
const udpIdleTimeout = 30 * time.Second

// udpFlowKey identifies a guest UDP flow by its ports and remote address. The
// guest address is always the configured guest IP, so it is not part of the key.
type udpFlowKey struct {
	srcPort uint16
	dstIP   [4]byte
	dstPort uint16
}

type udpFlow struct {
	key      udpFlowKey
	conn     *net.UDPConn
	srcIP    net.IP // remote (host-side) address, echoed back as source to guest
	dstIP    net.IP // guest address
	lastSeen time.Time
	timer    *time.Timer
}

type udpStack struct {
	s     *stack
	mu    sync.Mutex
	flows map[udpFlowKey]*udpFlow
}

func newUDPStack(s *stack) *udpStack {
	return &udpStack{s: s, flows: make(map[udpFlowKey]*udpFlow)}
}

func (u *udpStack) handlePacket(pkt ipv4Packet) {
	p := pkt.payload
	if len(p) < udpHdrLen {
		return
	}
	srcPort := binary.BigEndian.Uint16(p[0:2])
	dstPort := binary.BigEndian.Uint16(p[2:4])
	length := int(binary.BigEndian.Uint16(p[4:6]))
	if length < udpHdrLen || length > len(p) {
		length = len(p)
	}
	data := p[udpHdrLen:length]

	var key udpFlowKey
	key.srcPort = srcPort
	copy(key.dstIP[:], pkt.dst.To4())
	key.dstPort = dstPort

	u.mu.Lock()
	flow := u.flows[key]
	if flow == nil {
		conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: pkt.dst, Port: int(dstPort)})
		if err != nil {
			u.mu.Unlock()
			return
		}
		flow = &udpFlow{
			key:   key,
			conn:  conn,
			srcIP: append(net.IP(nil), pkt.dst.To4()...),
			dstIP: append(net.IP(nil), pkt.src.To4()...),
		}
		u.flows[key] = flow
		go u.readReplies(flow)
	}
	flow.touch(u)
	u.mu.Unlock()

	flow.conn.Write(data)
}

// touch resets the idle timer for a flow. Caller holds u.mu.
func (f *udpFlow) touch(u *udpStack) {
	f.lastSeen = time.Now()
	if f.timer == nil {
		f.timer = time.AfterFunc(udpIdleTimeout, func() { u.expire(f) })
	} else {
		f.timer.Reset(udpIdleTimeout)
	}
}

func (u *udpStack) expire(flow *udpFlow) {
	u.mu.Lock()
	if u.flows[flow.key] == flow {
		delete(u.flows, flow.key)
	}
	u.mu.Unlock()
	flow.conn.Close()
}

// readReplies forwards datagrams received on the host socket back to the guest.
func (u *udpStack) readReplies(flow *udpFlow) {
	buf := make([]byte, 65536)
	for {
		n, err := flow.conn.Read(buf)
		if err != nil {
			return
		}
		datagram := buildUDP(flow.srcIP, flow.dstIP, flow.key.dstPort, flow.key.srcPort, buf[:n])
		u.s.sendIPv4(flow.srcIP, flow.dstIP, protoUDP, datagram)

		u.mu.Lock()
		if u.flows[flow.key] == flow {
			flow.touch(u)
		}
		u.mu.Unlock()
	}
}

// buildUDP assembles a UDP datagram (header + payload) with a valid checksum.
func buildUDP(src, dst net.IP, srcPort, dstPort uint16, payload []byte) []byte {
	dg := make([]byte, udpHdrLen+len(payload))
	binary.BigEndian.PutUint16(dg[0:2], srcPort)
	binary.BigEndian.PutUint16(dg[2:4], dstPort)
	binary.BigEndian.PutUint16(dg[4:6], uint16(len(dg)))
	copy(dg[udpHdrLen:], payload)

	sum := pseudoHeaderSum(src, dst, protoUDP, len(dg))
	csum := checksum(dg, sum)
	if csum == 0 {
		csum = 0xffff // 0 means "no checksum"; use the equivalent all-ones.
	}
	binary.BigEndian.PutUint16(dg[6:8], csum)
	return dg
}
