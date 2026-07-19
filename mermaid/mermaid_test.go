package main

import "strings"

import "testing"

// plain renders src and joins the styled runs back into plain text lines.
func plain(src string) string {
	lines := render(src, 120)
	var b strings.Builder
	for _, line := range lines {
		for _, sp := range line {
			b.WriteString(sp.text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestBlankInput(t *testing.T) {
	if got := render("   \n\t\n", 120); got != nil {
		t.Fatalf("blank input should render nil, got %d lines", len(got))
	}
}

func TestFlowchart(t *testing.T) {
	out := plain("flowchart TD\n  A[Start] --> B[End]\n")
	for _, want := range []string{"Start", "End", "▼"} {
		if !strings.Contains(out, want) {
			t.Errorf("flowchart output missing %q:\n%s", want, out)
		}
	}
}

func TestFlowchartLabelAndDiamond(t *testing.T) {
	out := plain("flowchart TD\n  A --> B{Choice}\n  B -->|yes| C\n")
	if !strings.Contains(out, "Choice") || !strings.Contains(out, "yes") {
		t.Errorf("missing diamond label or edge label:\n%s", out)
	}
	// Diamond nodes use rounded corners.
	if !strings.Contains(out, "╭") {
		t.Errorf("diamond should use rounded corner glyphs:\n%s", out)
	}
}

func TestStateDiagram(t *testing.T) {
	out := plain("stateDiagram-v2\n  [*] --> Idle\n  Idle --> [*]\n")
	if !strings.Contains(out, "Idle") || !strings.Contains(out, "●") {
		t.Errorf("state diagram missing Idle or start/end marker:\n%s", out)
	}
}

func TestClassDiagram(t *testing.T) {
	out := plain("classDiagram\n  Animal <|-- Dog\n  Animal : +int age\n")
	for _, want := range []string{"Animal", "Dog", "+int age", "△", "├"} {
		if !strings.Contains(out, want) {
			t.Errorf("class diagram missing %q:\n%s", want, out)
		}
	}
}

func TestERDiagram(t *testing.T) {
	out := plain("erDiagram\n  CUSTOMER ||--o{ ORDER : places\n")
	for _, want := range []string{"CUSTOMER", "ORDER", "places", "0..*"} {
		if !strings.Contains(out, want) {
			t.Errorf("ER diagram missing %q:\n%s", want, out)
		}
	}
}

func TestSequenceDiagram(t *testing.T) {
	out := plain("sequenceDiagram\n  participant A as Alice\n  A->>B: Hello\n")
	for _, want := range []string{"Alice", "Hello", "▶"} {
		if !strings.Contains(out, want) {
			t.Errorf("sequence diagram missing %q:\n%s", want, out)
		}
	}
}

func TestSubgraphFrame(t *testing.T) {
	out := plain("flowchart TB\n  subgraph web[Web Tier]\n    A --> B\n  end\n")
	if !strings.Contains(out, "Web Tier") {
		t.Errorf("subgraph title missing:\n%s", out)
	}
}

func TestFallback(t *testing.T) {
	out := plain("pie title Pets\n  \"Dogs\" : 50\n")
	if !strings.Contains(out, "mermaid: pie") {
		t.Errorf("fallback frame title missing:\n%s", out)
	}
}

func TestTooWideFallback(t *testing.T) {
	// A very wide flowchart with a narrow max width should fall back with a hint.
	src := "flowchart LR\n  A --> B --> C --> D --> E --> F --> G --> H\n"
	lines := render(src, 12)
	var b strings.Builder
	for _, line := range lines {
		for _, sp := range line {
			b.WriteString(sp.text)
		}
		b.WriteByte('\n')
	}
	if !strings.Contains(b.String(), "too wide") {
		t.Errorf("expected too-wide hint:\n%s", b.String())
	}
}

func TestHTMLEntitiesAndBreak(t *testing.T) {
	out := plain("flowchart TD\n  A[\"a &amp; b<br/>c\"] --> B\n")
	if !strings.Contains(out, "a & b") || !strings.Contains(out, "c") {
		t.Errorf("entity/br handling wrong:\n%s", out)
	}
}

func TestDirLR(t *testing.T) {
	out := plain("graph LR\n  A --> B\n")
	if !strings.Contains(out, "▶") {
		t.Errorf("LR flow should use ▶ arrow:\n%s", out)
	}
}

func TestExtractMermaidBlocks(t *testing.T) {
	doc := "# Title\n\n```mermaid\ngraph LR\n  A --> B\n```\n\ntext\n\n~~~mermaid\ngraph TD\n  C --> D\n~~~\n"
	blocks := extractMermaidBlocks(doc)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d: %#v", len(blocks), blocks)
	}
	if !strings.Contains(blocks[0], "A --> B") || !strings.Contains(blocks[1], "C --> D") {
		t.Errorf("block contents wrong: %#v", blocks)
	}
}

func TestNodeCap(t *testing.T) {
	// Exceeding the node cap makes parse_graph bail, so it renders as fallback.
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for i := 0; i < maxNodes+10; i++ {
		b.WriteString("  n")
		b.WriteString(itoa(i))
		b.WriteString(" --> n")
		b.WriteString(itoa(i + 1))
		b.WriteByte('\n')
	}
	out := plain(b.String())
	if !strings.Contains(out, "mermaid: flowchart") {
		t.Errorf("over-cap graph should fall back:\n%s", out[:min(len(out), 200)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
