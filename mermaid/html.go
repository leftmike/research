package main

import (
	"fmt"
	"html"
	"io"
	"strings"
)

// cssClass maps a cell style class to its CSS class name (empty = unstyled).
func cssClass(cls int) string {
	switch cls {
	case clsBorder:
		return "b"
	case clsText:
		return "t"
	case clsEdge:
		return "e"
	case clsEdgeLabel:
		return "l"
	}
	return ""
}

const htmlHead = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
:root {
  --bg: #ffffff; --fg: #1f2328; --border: #6b7280;
  --text: #111827; --edge: #0891b2; --label: #b45309; --card: #f6f8fa;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0d1117; --fg: #e6edf3; --border: #8b949e;
    --text: #f0f6fc; --edge: #39c5cf; --label: #e3b341; --card: #161b22;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0; padding: 2rem 1rem; background: var(--bg); color: var(--fg);
  font-family: system-ui, -apple-system, Segoe UI, Roboto, sans-serif;
}
h1 { font-size: 1.1rem; font-weight: 600; margin: 0 auto 1.5rem; max-width: 60rem; }
main { max-width: 60rem; margin: 0 auto; display: flex; flex-direction: column; gap: 1.5rem; }
pre {
  margin: 0; padding: 1rem 1.25rem; overflow-x: auto;
  background: var(--card); border: 1px solid var(--border); border-radius: 8px;
  font-family: "Cascadia Code", "JetBrains Mono", "SFMono-Regular", Menlo, Consolas, monospace;
  font-size: 13px; line-height: 1.35; white-space: pre; tab-size: 4;
}
.b { color: var(--border); }
.t { color: var(--text); font-weight: 600; }
.e { color: var(--edge); }
.l { color: var(--label); }
</style>
</head>
<body>
`

// writeHTML renders diagrams as a single self-contained HTML page. Each diagram
// becomes a <pre> block whose spans are colored by CSS class.
func writeHTML(w io.Writer, title string, diagrams [][]span2D) error {
	if title == "" {
		title = "Mermaid diagrams"
	}
	var b strings.Builder
	fmt.Fprintf(&b, htmlHead, html.EscapeString(title))
	fmt.Fprintf(&b, "<h1>%s</h1>\n<main>\n", html.EscapeString(title))
	for _, lines := range diagrams {
		b.WriteString(`<pre role="img">`)
		b.WriteByte('\n')
		for _, line := range lines {
			for _, sp := range line {
				esc := html.EscapeString(sp.text)
				if c := cssClass(sp.cls); c != "" {
					fmt.Fprintf(&b, `<span class="%s">%s</span>`, c, esc)
				} else {
					b.WriteString(esc)
				}
			}
			b.WriteByte('\n')
		}
		b.WriteString("</pre>\n")
	}
	b.WriteString("</main>\n</body>\n</html>\n")
	_, err := io.WriteString(w, b.String())
	return err
}
