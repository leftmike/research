package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestDecodeHTMLEntities(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a &amp; b", "a & b"},
		{"&lt;tag&gt;", "<tag>"},
		{"&#65;&#66;&#67;", "ABC"},
		{"&#x41;&#X42;", "AB"},
		{"&quot;q&quot; &apos;a&apos;", "\"q\" 'a'"},
		// Single left-to-right pass: emitted text is never re-scanned, so a
		// double-encoded entity decodes only one level.
		{"&amp;lt;", "&lt;"},
		// Unknown / malformed stay literal.
		{"AT&T", "AT&T"},
		{"&nope;", "&nope;"},
		{"no entities here", "no entities here"},
		// Control chars are rejected and left literal.
		{"&#0;", "&#0;"},
		{"&#10;", "&#10;"},
	}
	for _, c := range cases {
		if got := decodeHTMLEntities(c.in); got != c.want {
			t.Errorf("decodeHTMLEntities(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestStripMarkdown(t *testing.T) {
	cases := []struct{ in, want string }{
		{"**bold**", "bold"},
		{"__strong__", "strong"},
		{"*em* text", "em text"},
		{"snake_case_name", "snake_case_name"}, // interior underscores survive
		{"`code`", "code"},
	}
	for _, c := range cases {
		if got := stripMarkdown(c.in); got != c.want {
			t.Errorf("stripMarkdown(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCleanLabel(t *testing.T) {
	cases := []struct{ in, want string }{
		{"\"quoted\"", "quoted"},
		{"<b>bold</b>", "bold"},
		{"line<br/>break", "line break"},
		{"`**md**`", "md"},
		{"&lt;keep&gt;", "<keep>"},
	}
	for _, c := range cases {
		if got := cleanLabel(c.in); got != c.want {
			t.Errorf("cleanLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWrapLabel(t *testing.T) {
	// Fits on one line.
	if got := wrapLabel("hello world", 20, 4); len(got) != 1 || got[0] != "hello world" {
		t.Errorf("short wrap = %#v", got)
	}
	// Wraps onto multiple lines at word boundaries.
	got := wrapLabel("one two three four five", 9, 4)
	if len(got) < 2 {
		t.Errorf("expected multi-line wrap, got %#v", got)
	}
	for _, l := range got {
		if strWidth(l) > 9 {
			t.Errorf("line %q exceeds width 9", l)
		}
	}
	// Long unbroken token breaks at an identifier boundary.
	seg := wrapLabel("alpha_beta_gamma_delta", 8, 4)
	if len(seg) < 2 {
		t.Errorf("expected long token to break, got %#v", seg)
	}
	// Overflow beyond max lines is truncated with an ellipsis.
	trunc := wrapLabel("a b c d e f g h i j k l", 3, 2)
	if len(trunc) != 2 || !strings.HasSuffix(trunc[len(trunc)-1], "…") {
		t.Errorf("expected 2 lines ending in ellipsis, got %#v", trunc)
	}
}

func TestFitLabel(t *testing.T) {
	if got := fitLabel("short", 10); got != "short" {
		t.Errorf("fitLabel no-op = %q", got)
	}
	got := fitLabel("this is far too long", 8)
	if strWidth(got) > 8 || !strings.HasSuffix(got, "…") {
		t.Errorf("fitLabel truncation = %q (width %d)", got, strWidth(got))
	}
}

func TestCharWidth(t *testing.T) {
	cases := []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{'世', 2},    // CJK wide
		{'́', 0},    // combining acute accent
		{'\x00', 0}, // NUL / cont sentinel
		{'…', 1},
	}
	for _, c := range cases {
		if got := charWidth(c.r); got != c.want {
			t.Errorf("charWidth(%q) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestAnsiFor(t *testing.T) {
	for _, cls := range []int{clsBorder, clsText, clsEdge, clsEdgeLabel} {
		if ansiFor(cls) == "" {
			t.Errorf("ansiFor(%d) should be non-empty", cls)
		}
	}
	if ansiFor(clsEmpty) != "" {
		t.Errorf("ansiFor(clsEmpty) should be empty")
	}
}

func TestWriteLinesColor(t *testing.T) {
	lines := render("flowchart LR\n  A --> B\n", 100)
	if lines == nil {
		t.Fatal("expected rendered lines")
	}

	var colored bytes.Buffer
	w := bufio.NewWriter(&colored)
	writeLines(w, lines, true)
	w.Flush()
	if !strings.Contains(colored.String(), "\x1b[") {
		t.Errorf("colored output should contain ANSI escapes:\n%q", colored.String())
	}

	var plainBuf bytes.Buffer
	w2 := bufio.NewWriter(&plainBuf)
	writeLines(w2, lines, false)
	w2.Flush()
	if strings.Contains(plainBuf.String(), "\x1b[") {
		t.Errorf("plain output should not contain ANSI escapes:\n%q", plainBuf.String())
	}
	// Stripping the color codes should reproduce the plain rendering.
	if stripANSI(colored.String()) != plainBuf.String() {
		t.Errorf("colored output (stripped) != plain output")
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip 'm'
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func TestColumnsEnv(t *testing.T) {
	t.Setenv("COLUMNS", "137")
	if got := columnsEnv(); got != 137 {
		t.Errorf("columnsEnv() = %d, want 137", got)
	}
	t.Setenv("COLUMNS", "not-a-number")
	if got := columnsEnv(); got != 0 {
		t.Errorf("columnsEnv() with junk = %d, want 0", got)
	}
}

func TestExtractBlocksEdgeCases(t *testing.T) {
	// No fences at all.
	if b := extractMermaidBlocks("just prose\nno code\n"); len(b) != 0 {
		t.Errorf("expected no blocks, got %#v", b)
	}
	// Non-mermaid fenced block is ignored.
	if b := extractMermaidBlocks("```go\nfmt.Println()\n```\n"); len(b) != 0 {
		t.Errorf("non-mermaid block should be ignored, got %#v", b)
	}
	// Unclosed mermaid block still yields its contents.
	b := extractMermaidBlocks("```mermaid\ngraph TD\n  A --> B\n")
	if len(b) != 1 || !strings.Contains(b[0], "A --> B") {
		t.Errorf("unclosed block = %#v", b)
	}
}
