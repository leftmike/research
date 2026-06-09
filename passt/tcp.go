package main

import (
	"encoding/binary"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

// TCP flag bits.
const (
	flagFIN = 0x01
	flagSYN = 0x02
	flagRST = 0x04
	flagPSH = 0x08
	flagACK = 0x10
)

const (
	tcpHdrLen     = 20
	recvWindow    = 65535            // window we advertise to the guest
	maxSendBuffer = 256 * 1024       // cap on bytes in flight toward the guest
	dialTimeout   = 10 * time.Second // host connect timeout
	rto           = 200 * time.Millisecond
	maxRetries    = 50
)

// tcpKey identifies a guest TCP connection. The guest address is constant, so
// only its port and the remote endpoint are needed.
type tcpKey struct {
	guestPort  uint16
	remoteIP   [4]byte
	remotePort uint16
}

// tcpSegment is a parsed TCP segment handed from the demux to a connection.
type tcpSegment struct {
	seq, ack uint32
	flags    byte
	window   uint16
	payload  []byte
}

type tcpStack struct {
	s     *stack
	mu    sync.Mutex
	conns map[tcpKey]*tcpConn
}

func newTCPStack(s *stack) *tcpStack {
	return &tcpStack{s: s, conns: make(map[tcpKey]*tcpConn)}
}

func (ts *tcpStack) remove(key tcpKey) {
	ts.mu.Lock()
	delete(ts.conns, key)
	ts.mu.Unlock()
}

func (ts *tcpStack) handlePacket(pkt ipv4Packet) {
	p := pkt.payload
	if len(p) < tcpHdrLen {
		return
	}
	dataOff := int(p[12]>>4) * 4
	if dataOff < tcpHdrLen || dataOff > len(p) {
		return
	}
	seg := &tcpSegment{
		seq:     binary.BigEndian.Uint32(p[4:8]),
		ack:     binary.BigEndian.Uint32(p[8:12]),
		flags:   p[13],
		window:  binary.BigEndian.Uint16(p[14:16]),
		payload: append([]byte(nil), p[dataOff:]...),
	}
	guestPort := binary.BigEndian.Uint16(p[0:2])
	remotePort := binary.BigEndian.Uint16(p[2:4])

	var key tcpKey
	key.guestPort = guestPort
	copy(key.remoteIP[:], pkt.dst.To4())
	key.remotePort = remotePort

	ts.mu.Lock()
	c := ts.conns[key]
	if c == nil {
		if seg.flags&flagSYN == 0 || seg.flags&flagACK != 0 {
			ts.mu.Unlock()
			return // not a fresh connection attempt; ignore
		}
		c = &tcpConn{
			s:          ts.s,
			ts:         ts,
			key:        key,
			guestIP:    ts.s.cfg.guestIP,
			remoteIP:   append(net.IP(nil), pkt.dst.To4()...),
			guestPort:  guestPort,
			remotePort: remotePort,
			mss:        ts.s.cfg.mtu - ipv4HdrLen - tcpHdrLen,
			in:         make(chan *tcpSegment, 256),
			done:       make(chan struct{}),
		}
		ts.conns[key] = c
		go c.run(seg)
		ts.mu.Unlock()
		return
	}
	ts.mu.Unlock()

	// Deliver to the connection goroutine; drop if it is backed up (the guest
	// will retransmit over the lossless TAP link).
	select {
	case c.in <- seg:
	default:
	}
}

// tcpConn proxies a single guest TCP connection to a host socket. All mutable
// fields are owned by the run goroutine.
type tcpConn struct {
	s  *stack
	ts *tcpStack

	key                   tcpKey
	guestIP, remoteIP     net.IP
	guestPort, remotePort uint16
	mss                   int

	host net.Conn

	in       chan *tcpSegment
	fromHost chan []byte // data read from the host socket; nil signals EOF
	done     chan struct{}

	// Receive side (guest -> host).
	rcvNxt uint32

	// Send side (host -> guest).
	iss     uint32
	sndUna  uint32
	sndNxt  uint32
	sndWnd  uint32
	unacked []byte // bytes sent but not yet acked, starting at sndUna

	established bool
	hostEOF     bool
	guestFIN    bool
	ourFIN      bool
	ourFINSeq   uint32
	finAcked    bool

	retries   int
	terminate bool
}

func (c *tcpConn) run(syn *tcpSegment) {
	if !c.dial(syn) {
		return
	}
	go c.readHost()

	ticker := time.NewTicker(rto)
	defer ticker.Stop()

	for !c.terminate {
		var hostCh chan []byte
		if c.established && len(c.unacked) < c.effWindow() {
			hostCh = c.fromHost
		}
		select {
		case seg := <-c.in:
			c.onSegment(seg)
		case data := <-hostCh:
			c.onHostData(data)
		case <-ticker.C:
			c.onTick()
		case <-c.s.closed:
			c.terminate = true
		}
	}
	c.cleanup()
}

// dial connects to the remote host and replies with a SYN-ACK. It returns false
// (after sending a RST and removing the connection) if the connect fails.
func (c *tcpConn) dial(syn *tcpSegment) bool {
	c.rcvNxt = syn.seq + 1
	c.sndWnd = uint32(syn.window)

	addr := net.JoinHostPort(c.remoteIP.String(), strconv.Itoa(int(c.remotePort)))
	host, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		c.send(flagRST|flagACK, nil, 0)
		c.ts.remove(c.key)
		return false
	}
	c.host = host
	c.iss = rand.Uint32()
	c.sndUna = c.iss
	c.sndNxt = c.iss
	c.fromHost = make(chan []byte, 4)

	c.send(flagSYN|flagACK, nil, c.iss)
	c.sndNxt = c.iss + 1
	return true
}

func (c *tcpConn) cleanup() {
	if c.host != nil {
		c.host.Close()
	}
	c.ts.remove(c.key)
	close(c.done) // stop the host reader
}

func (c *tcpConn) effWindow() int {
	w := int(c.sndWnd)
	if w > maxSendBuffer {
		w = maxSendBuffer
	}
	if w < c.mss {
		w = c.mss // always allow at least one segment
	}
	return w
}

// readHost forwards bytes from the host socket to the run goroutine, signaling
// EOF with a nil chunk.
func (c *tcpConn) readHost() {
	buf := make([]byte, c.mss)
	for {
		n, err := c.host.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			select {
			case c.fromHost <- chunk:
			case <-c.done:
				return
			}
		}
		if err != nil {
			select {
			case c.fromHost <- nil:
			case <-c.done:
			}
			return
		}
	}
}

func (c *tcpConn) onSegment(seg *tcpSegment) {
	if seg.flags&flagRST != 0 {
		c.terminate = true
		return
	}
	if seg.flags&flagACK != 0 {
		c.sndWnd = uint32(seg.window)
		c.processAck(seg.ack)
	}
	if seg.flags&flagSYN != 0 && !c.established {
		// Retransmitted SYN before the handshake completed: resend SYN-ACK.
		c.send(flagSYN|flagACK, nil, c.iss)
		return
	}
	if len(seg.payload) > 0 {
		c.onGuestData(seg.seq, seg.payload)
	}
	if seg.flags&flagFIN != 0 {
		finSeq := seg.seq + uint32(len(seg.payload))
		if finSeq == c.rcvNxt && !c.guestFIN {
			c.rcvNxt++
			c.guestFIN = true
			c.send(flagACK, nil, c.sndNxt)
			if tcp, ok := c.host.(*net.TCPConn); ok {
				tcp.CloseWrite() // let the remote see EOF
			}
		} else {
			c.send(flagACK, nil, c.sndNxt) // duplicate/out-of-order FIN
		}
	}
	c.maybeFIN()
	c.checkClose()
}

func (c *tcpConn) processAck(ack uint32) {
	if !c.established {
		if seqGT(ack, c.iss) {
			c.established = true
			c.sndUna = c.iss + 1
		} else {
			return
		}
	}
	if seqGT(ack, c.sndUna) {
		n := int(ack - c.sndUna)
		if n > len(c.unacked) {
			n = len(c.unacked)
		}
		c.unacked = c.unacked[n:]
		c.sndUna += uint32(n)
		c.retries = 0
	}
	if c.ourFIN && seqGEQ(ack, c.ourFINSeq+1) {
		c.finAcked = true
	}
}

func (c *tcpConn) onGuestData(seq uint32, payload []byte) {
	if seq == c.rcvNxt {
		if _, err := c.host.Write(payload); err != nil {
			c.terminate = true
			return
		}
		c.rcvNxt += uint32(len(payload))
	}
	// In all cases (in-order, duplicate, or gap) acknowledge what we have so
	// the guest knows where to resume.
	c.send(flagACK, nil, c.sndNxt)
}

func (c *tcpConn) onHostData(data []byte) {
	if data == nil {
		c.hostEOF = true
		c.maybeFIN()
		c.checkClose()
		return
	}
	c.send(flagPSH|flagACK, data, c.sndNxt)
	c.unacked = append(c.unacked, data...)
	c.sndNxt += uint32(len(data))
}

// maybeFIN sends our FIN once the host has hit EOF and all data is acknowledged.
func (c *tcpConn) maybeFIN() {
	if c.established && c.hostEOF && !c.ourFIN && len(c.unacked) == 0 {
		c.ourFINSeq = c.sndNxt
		c.send(flagFIN|flagACK, nil, c.sndNxt)
		c.sndNxt++
		c.ourFIN = true
	}
}

func (c *tcpConn) checkClose() {
	if c.guestFIN && c.ourFIN && c.finAcked {
		c.terminate = true
	}
}

// onTick drives retransmission of the oldest unacknowledged data, a pending
// FIN, or the SYN-ACK during the handshake.
func (c *tcpConn) onTick() {
	switch {
	case len(c.unacked) > 0:
		n := len(c.unacked)
		if n > c.mss {
			n = c.mss
		}
		if c.bumpRetries() {
			c.send(flagPSH|flagACK, c.unacked[:n], c.sndUna)
		}
	case c.ourFIN && !c.finAcked:
		if c.bumpRetries() {
			c.send(flagFIN|flagACK, nil, c.ourFINSeq)
		}
	case !c.established:
		if c.bumpRetries() {
			c.send(flagSYN|flagACK, nil, c.iss)
		}
	default:
		c.retries = 0
	}
}

// bumpRetries increments the retry counter, giving up (terminating the
// connection) once the limit is reached. It returns true if a retransmit should
// proceed.
func (c *tcpConn) bumpRetries() bool {
	c.retries++
	if c.retries > maxRetries {
		c.terminate = true
		return false
	}
	return true
}

// send builds and emits a TCP segment toward the guest. We impersonate the
// remote endpoint, so the segment's source is the remote address.
func (c *tcpConn) send(flags byte, payload []byte, seq uint32) {
	seg := make([]byte, tcpHdrLen+len(payload))
	binary.BigEndian.PutUint16(seg[0:2], c.remotePort) // source: remote
	binary.BigEndian.PutUint16(seg[2:4], c.guestPort)  // dest: guest
	binary.BigEndian.PutUint32(seg[4:8], seq)
	binary.BigEndian.PutUint32(seg[8:12], c.rcvNxt) // ack
	seg[12] = byte(tcpHdrLen/4) << 4
	seg[13] = flags
	binary.BigEndian.PutUint16(seg[14:16], recvWindow)
	copy(seg[tcpHdrLen:], payload)

	sum := pseudoHeaderSum(c.remoteIP, c.guestIP, protoTCP, len(seg))
	csum := checksum(seg, sum)
	binary.BigEndian.PutUint16(seg[16:18], csum)

	c.s.sendIPv4(c.remoteIP, c.guestIP, protoTCP, seg)
}

// Sequence-number comparisons, valid across 32-bit wraparound.
func seqGT(a, b uint32) bool  { return int32(a-b) > 0 }
func seqGEQ(a, b uint32) bool { return int32(a-b) >= 0 }
