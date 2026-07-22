package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteHTMLStructure(t *testing.T) {
	lines := render("flowchart LR\n  A --> B\n", 0)
	if lines == nil {
		t.Fatal("expected rendered lines")
	}
	var buf bytes.Buffer
	if err := writeHTML(&buf, "My Title", [][]span2D{lines}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"<!doctype html>",
		"<title>My Title</title>",
		"<h1>My Title</h1>",
		"prefers-color-scheme: dark",
		`<pre role="img">`,
		`<span class="t">A</span>`,
		"</html>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML output missing %q", want)
		}
	}
}

func TestWriteHTMLEscaping(t *testing.T) {
	// A label with HTML metacharacters must be escaped, not emitted raw.
	lines := render("flowchart LR\n  A[\"x < y & z\"] --> B\n", 0)
	var buf bytes.Buffer
	if err := writeHTML(&buf, "t", [][]span2D{lines}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "x &lt; y &amp; z") {
		t.Errorf("expected escaped label, got:\n%s", out)
	}
	if strings.Contains(out, "x < y & z") {
		t.Errorf("raw metacharacters leaked into HTML:\n%s", out)
	}
}

func TestWriteHTMLDefaultTitle(t *testing.T) {
	var buf bytes.Buffer
	if err := writeHTML(&buf, "", nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "<title>Mermaid diagrams</title>") {
		t.Errorf("expected default title")
	}
}

func TestCSSClass(t *testing.T) {
	cases := map[int]string{
		clsBorder: "b", clsText: "t", clsEdge: "e", clsEdgeLabel: "l", clsEmpty: "",
	}
	for cls, want := range cases {
		if got := cssClass(cls); got != want {
			t.Errorf("cssClass(%d) = %q, want %q", cls, got, want)
		}
	}
}

func TestRunHTML(t *testing.T) {
	code, out, errb := runCLI(t, []string{"-html", "-title", "CLI Page"},
		"flowchart LR\n  A --> B\n")
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb)
	}
	for _, want := range []string{"<!doctype html>", "<title>CLI Page</title>", `<pre role="img">`, "</html>"} {
		if !strings.Contains(out, want) {
			t.Errorf("CLI -html output missing %q", want)
		}
	}
	// HTML uses CSS classes, not ANSI escapes.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("HTML output should not contain ANSI escapes")
	}
}

func TestRunHTMLMultipleBlocks(t *testing.T) {
	doc := "```mermaid\ngraph LR\n  A --> B\n```\n```mermaid\ngraph TD\n  C --> D\n```\n"
	code, out, _ := runCLI(t, []string{"-html"}, doc)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	// Two diagrams -> two <pre> blocks.
	if n := strings.Count(out, "<pre"); n != 2 {
		t.Errorf("expected 2 <pre> blocks, got %d", n)
	}
}
