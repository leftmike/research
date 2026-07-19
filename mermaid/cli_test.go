package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunStdinRaw(t *testing.T) {
	code, out, errb := runCLI(t, []string{"-w", "100", "-color", "never", "-raw"},
		"flowchart LR\n  A --> B\n")
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb)
	}
	if !strings.Contains(out, "A") || !strings.Contains(out, "▶") {
		t.Errorf("unexpected output:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("color=never should emit no escapes:\n%q", out)
	}
}

func TestRunColorAlways(t *testing.T) {
	code, out, _ := runCLI(t, []string{"-w", "100", "-color", "always", "-raw"},
		"flowchart LR\n  A --> B\n")
	if code != 0 || !strings.Contains(out, "\x1b[") {
		t.Errorf("color=always should emit escapes; code=%d out=%q", code, out)
	}
}

func TestRunFenceExtraction(t *testing.T) {
	doc := "# Title\n\ntext\n\n```mermaid\ngraph TD\n  A --> B\n```\n\n```go\nx := 1\n```\n"
	code, out, _ := runCLI(t, []string{"-w", "100", "-color", "never"}, doc)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "┌") {
		t.Errorf("expected a rendered box from the mermaid fence:\n%s", out)
	}
	if strings.Contains(out, "x := 1") {
		t.Errorf("go fence should not be rendered:\n%s", out)
	}
}

func TestRunMultipleBlocksSeparated(t *testing.T) {
	doc := "```mermaid\ngraph LR\n  A --> B\n```\n```mermaid\ngraph LR\n  C --> D\n```\n"
	code, out, _ := runCLI(t, []string{"-w", "100", "-color", "never"}, doc)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "A") || !strings.Contains(out, "C") {
		t.Errorf("both blocks should render:\n%s", out)
	}
	// Two diagrams should be separated by a blank line.
	if !strings.Contains(out, "\n\n") {
		t.Errorf("expected blank-line separator between diagrams:\n%s", out)
	}
}

func TestRunFileArg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diagram.mmd")
	if err := os.WriteFile(path, []byte("flowchart TD\n  A --> B\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runCLI(t, []string{"-w", "100", "-color", "never", path}, "")
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb)
	}
	if !strings.Contains(out, "▼") {
		t.Errorf("expected rendered flowchart from file:\n%s", out)
	}
}

func TestRunMissingFile(t *testing.T) {
	code, _, errb := runCLI(t, []string{"/no/such/file.mmd"}, "")
	if code != 1 {
		t.Errorf("missing file should exit 1, got %d", code)
	}
	if !strings.Contains(errb, "mermaid:") {
		t.Errorf("expected error message, got %q", errb)
	}
}

func TestRunBadFlag(t *testing.T) {
	code, _, _ := runCLI(t, []string{"-nonsense"}, "")
	if code != 2 {
		t.Errorf("bad flag should exit 2, got %d", code)
	}
}

func TestRunBlankProducesNothing(t *testing.T) {
	code, out, _ := runCLI(t, []string{"-color", "never"}, "   \n\n")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("blank input should produce no output, got %q", out)
	}
}
