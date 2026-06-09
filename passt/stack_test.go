package main

import (
	"encoding/binary"
	"net"
	"testing"
)

// TestIPChecksum checks the one's-complement checksum against a well-known
// IPv4 header whose correct checksum is 0xb861.
func TestIPChecksum(t *testing.T) {
	hdr := []byte{
		0x45, 0x00, 0x00, 0x73, 0x00, 0x00, 0x40, 0x00,
		0x40, 0x11, 0x00, 0x00, // checksum field zeroed
		0xc0, 0xa8, 0x00, 0x01,
		0xc0, 0xa8, 0x00, 0xc7,
	}
	got := checksum(hdr, 0)
	if got != 0xb861 {
		t.Fatalf("checksum = %#04x, want 0xb861", got)
	}
	// With the correct checksum in place, the header must verify to zero.
	binary.BigEndian.PutUint16(hdr[10:12], got)
	if v := checksum(hdr, 0); v != 0 {
		t.Fatalf("verify checksum = %#04x, want 0", v)
	}
}

// TestUDPChecksum builds a datagram and confirms its checksum verifies to zero
// when recomputed over the pseudo-header plus the datagram.
func TestUDPChecksum(t *testing.T) {
	src := net.IPv4(10, 0, 0, 1)
	dst := net.IPv4(10, 0, 0, 2)
	dg := buildUDP(src, dst, 53, 4000, []byte("the quick brown fox"))

	sum := pseudoHeaderSum(src, dst, protoUDP, len(dg))
	if v := checksum(dg, sum); v != 0 {
		t.Fatalf("UDP checksum verify = %#04x, want 0", v)
	}
	if got := binary.BigEndian.Uint16(dg[4:6]); int(got) != len(dg) {
		t.Fatalf("UDP length field = %d, want %d", got, len(dg))
	}
}

func TestSeqCompare(t *testing.T) {
	cases := []struct {
		a, b   uint32
		gt, ge bool
	}{
		{5, 3, true, true},
		{3, 5, false, false},
		{5, 5, false, true},
		{0, 0xffffffff, true, true},   // wraparound: 0 is "after" 2^32-1
		{0xffffffff, 0, false, false}, // and 2^32-1 is "before" 0
	}
	for _, c := range cases {
		if got := seqGT(c.a, c.b); got != c.gt {
			t.Errorf("seqGT(%d,%d)=%v want %v", c.a, c.b, got, c.gt)
		}
		if got := seqGEQ(c.a, c.b); got != c.ge {
			t.Errorf("seqGEQ(%d,%d)=%v want %v", c.a, c.b, got, c.ge)
		}
	}
}

// TestRTAttrPadding verifies route attributes are padded to a 4-byte boundary.
func TestRTAttrPadding(t *testing.T) {
	a := rtAttr(7, []byte{1, 2, 3}) // 4-byte header + 3-byte value = 7, pad to 8
	if len(a) != 8 {
		t.Fatalf("len = %d, want 8", len(a))
	}
	if l := binary.LittleEndian.Uint16(a[0:2]); l != 7 {
		t.Fatalf("rta_len = %d, want 7", l)
	}
	if typ := binary.LittleEndian.Uint16(a[2:4]); typ != 7 {
		t.Fatalf("rta_type = %d, want 7", typ)
	}
}
