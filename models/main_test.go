package main

import "testing"

func TestParseSources(t *testing.T) {
	cases := []struct {
		in             string
		md, ll, wantOK bool
	}{
		{"all", true, true, true},
		{"", true, true, true},
		{"models.dev", true, false, true},
		{"MD", true, false, true},
		{"litellm", false, true, true},
		{"ll", false, true, true},
		{"bogus", false, false, false},
	}
	for _, c := range cases {
		md, ll, err := parseSources(c.in)
		if (err == nil) != c.wantOK {
			t.Errorf("parseSources(%q) ok = %v, want %v (err=%v)", c.in, err == nil, c.wantOK, err)
			continue
		}
		if err == nil && (md != c.md || ll != c.ll) {
			t.Errorf("parseSources(%q) = (%v,%v), want (%v,%v)", c.in, md, ll, c.md, c.ll)
		}
	}
}
