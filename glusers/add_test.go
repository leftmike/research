package main

import (
	"testing"
	"time"
)

var now = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

func TestParseExpires(t *testing.T) {
	cases := []struct {
		s    string
		r    string
		fail bool
	}{
		{s: "", r: ""},

		{s: "2026-01-02", r: "2026-01-02"},
		{s: "2026/01/02", r: "2026-01-02"},
		{s: "01/02/2026", r: "2026-01-02"},
		{s: "01-02-2026", r: "2026-01-02"},
		{s: "Jan 2, 2026", r: "2026-01-02"},
		{s: "2 Jan 2026", r: "2026-01-02"},
		{s: "2 Jan 26", r: "2026-01-02"},
		{s: "02 Jan 06", r: "2006-01-02"},
		{s: "02 Jan 2006", r: "2006-01-02"},
		{s: "02-Jan-06", r: "2006-01-02"},
		{s: "02-Jan-2006", r: "2006-01-02"},
		{s: "jan 2, 2026", r: "2026-01-02"},
		{s: "JAN 2, 2026", r: "2026-01-02"},

		{s: "3d", r: "2026-01-05"},
		{s: "6w", r: "2026-02-13"},
		{s: "3D", r: "", fail: true},
		{s: "6W", r: "", fail: true},

		{s: "0d", r: "", fail: true},
		{s: "-1d", r: "", fail: true},
		{s: "3x", r: "", fail: true},
		{s: "2026-13-01", r: "", fail: true},
		{s: "13/01/2026", r: "", fail: true},
	}

	for _, c := range cases {
		r, err := parseExpires(c.s, now)
		if c.fail {
			if err == nil {
				t.Errorf("parseExpires(%q) did not fail", c.s)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseExpires(%q) failed with %s", c.s, err)
			continue
		}
		if r != c.r {
			t.Errorf("parseExpires(%q) got %q want %q", c.s, r, c.r)
		}
	}
}
