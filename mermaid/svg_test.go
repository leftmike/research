package main

import (
	"strings"
	"testing"
)

func TestDiagramSVGDimensions(t *testing.T) {
	lines := render("flowchart LR\n  A --> B\n", 0)
	group, wpx, hpx := diagramSVG(lines)
	if wpx <= 2*svgPad || hpx <= 2*svgPad {
		t.Errorf("degenerate dimensions: w=%v h=%v", wpx, hpx)
	}
	if !strings.HasPrefix(group, "<g>") || !strings.Contains(group, "<text") {
		t.Errorf("group missing text runs: %s", group)
	}
	// One row per rendered line.
	if want := 2*svgPad + float64(len(lines))*svgCellH; hpx != want {
		t.Errorf("height = %v, want %v", hpx, want)
	}
}

func TestSVGElement(t *testing.T) {
	lines := render("flowchart LR\n  A --> B\n", 0)
	out := svgElement(lines)
	for _, want := range []string{
		"<svg", `xmlns="http://www.w3.org/2000/svg"`, "viewBox=",
		`textLength=`, `lengthAdjust="spacingAndGlyphs"`, `class="t"`, "</svg>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("svgElement missing %q", want)
		}
	}
}

func TestStandaloneSVGStructure(t *testing.T) {
	lines := render("flowchart LR\n  A --> B\n", 0)
	out := standaloneSVG([][]span2D{lines})
	for _, want := range []string{
		"<?xml version=", "<svg", "<style>", "prefers-color-scheme: dark",
		`fill="var(--bg)"`, "text.e { fill: var(--edge); }", "</svg>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("standaloneSVG missing %q", want)
		}
	}
}

func TestStandaloneSVGStacksDiagrams(t *testing.T) {
	a := render("graph LR\n  A --> B\n", 0)
	b := render("graph TD\n  C --> D\n", 0)
	out := standaloneSVG([][]span2D{a, b})
	if n := strings.Count(out, "<g transform="); n != 2 {
		t.Errorf("expected 2 translated groups, got %d", n)
	}
	// Second diagram is offset below the first.
	if strings.Count(out, "translate(0,10)") == 0 {
		t.Errorf("first group should sit at the top padding")
	}
}

func TestSVGEscaping(t *testing.T) {
	lines := render("flowchart LR\n  A[\"p < q & r\"] --> B\n", 0)
	out := svgElement(lines)
	if !strings.Contains(out, "p &lt; q &amp; r") {
		t.Errorf("expected escaped label in SVG:\n%s", out)
	}
	if strings.Contains(out, "p < q & r") {
		t.Errorf("raw metacharacters leaked into SVG")
	}
}

func TestRunSVGStandalone(t *testing.T) {
	code, out, errb := runCLI(t, []string{"-svg"}, "flowchart LR\n  A --> B\n")
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb)
	}
	if !strings.HasPrefix(out, "<?xml version=") || !strings.Contains(out, "<svg") {
		t.Errorf("expected standalone SVG document, got:\n%.120s", out)
	}
	if strings.Contains(out, "<!doctype html>") {
		t.Errorf("standalone -svg should not wrap in HTML")
	}
}

func TestRunHTMLWithSVG(t *testing.T) {
	code, out, _ := runCLI(t, []string{"-html", "-svg"}, "flowchart LR\n  A --> B\n")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "<!doctype html>") || !strings.Contains(out, "<svg") {
		t.Errorf("expected HTML page with inline SVG")
	}
	if strings.Contains(out, `<pre role="img">`) {
		t.Errorf("-html -svg should embed <svg>, not <pre>")
	}
	if !strings.Contains(out, `<div class="card">`) {
		t.Errorf("expected svg wrapped in a card div")
	}
}
