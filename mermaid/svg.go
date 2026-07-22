package main

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
)

// SVG grid metrics (pixels). Each terminal column/row maps to a fixed cell; a
// run's textLength is pinned to its exact column span so box-drawing glyphs stay
// aligned no matter what monospace font the viewer has.
const (
	svgCellW    = 8.4
	svgCellH    = 17.0
	svgFontSize = 14.0
	svgBaseline = 13.0 // baseline offset from the top of a row
	svgPad      = 10.0
	svgGap      = 17.0 // vertical gap between stacked diagrams
)

// fnum formats a coordinate rounded to 2 decimals, trimming trailing zeros.
func fnum(x float64) string {
	return strconv.FormatFloat(math.Round(x*100)/100, 'f', -1, 64)
}

// diagramSVG builds a <g> group of positioned <text> runs for one diagram and
// returns it with the diagram's pixel width and height. Coordinates are local
// to the diagram (origin at its top-left), so callers can translate it.
func diagramSVG(lines []span2D) (group string, wpx, hpx float64) {
	var b strings.Builder
	b.WriteString("<g>")
	maxCols := 0
	for row, line := range lines {
		col := 0
		for _, sp := range line {
			cols := strWidth(sp.text)
			if c := cssClass(sp.cls); c != "" && strings.TrimSpace(sp.text) != "" {
				x := svgPad + float64(col)*svgCellW
				y := svgPad + float64(row)*svgCellH + svgBaseline
				fmt.Fprintf(&b,
					`<text x="%s" y="%s" textLength="%s" lengthAdjust="spacingAndGlyphs" class="%s" xml:space="preserve">%s</text>`,
					fnum(x), fnum(y), fnum(float64(cols)*svgCellW), c, html.EscapeString(sp.text))
			}
			col += cols
		}
		if col > maxCols {
			maxCols = col
		}
	}
	b.WriteString("</g>")
	wpx = 2*svgPad + float64(maxCols)*svgCellW
	hpx = 2*svgPad + float64(len(lines))*svgCellH
	return b.String(), wpx, hpx
}

// svgElement wraps one diagram as a complete <svg> element for inline embedding
// in an HTML page. Fills come from the page stylesheet (svg text.b/.t/.e/.l).
func svgElement(lines []span2D) string {
	group, wpx, hpx := diagramSVG(lines)
	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s" font-family="%s" font-size="%s" role="img">%s</svg>`,
		fnum(wpx), fnum(hpx), fnum(wpx), fnum(hpx), svgFontFamily, fnum(svgFontSize), group)
}

const svgFontFamily = `ui-monospace, &quot;Cascadia Code&quot;, &quot;JetBrains Mono&quot;, &quot;SFMono-Regular&quot;, Menlo, Consolas, monospace`

const svgStyle = `<style>
:root {
  --bg: #ffffff; --border: #6b7280; --text: #111827; --edge: #0891b2; --label: #b45309;
}
@media (prefers-color-scheme: dark) {
  :root { --bg: #0d1117; --border: #8b949e; --text: #f0f6fc; --edge: #39c5cf; --label: #e3b341; }
}
text { font-variant-ligatures: none; white-space: pre; }
text.b { fill: var(--border); }
text.t { fill: var(--text); font-weight: 600; }
text.e { fill: var(--edge); }
text.l { fill: var(--label); }
</style>`

// standaloneSVG renders one or more diagrams stacked vertically into a single
// self-contained SVG document (its own style, theme-aware, no external assets).
func standaloneSVG(diagrams [][]span2D) string {
	var groups strings.Builder
	maxW := 0.0
	yOff := svgPad
	for i, lines := range diagrams {
		group, wpx, hpx := diagramSVG(lines)
		if wpx > maxW {
			maxW = wpx
		}
		if i > 0 {
			yOff += svgGap
		}
		fmt.Fprintf(&groups, `<g transform="translate(0,%s)">%s</g>`, fnum(yOff), group)
		yOff += hpx
	}
	if maxW == 0 {
		maxW = 2 * svgPad
	}
	totalH := yOff + svgPad
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s" font-family="%s" font-size="%s">`,
		fnum(maxW), fnum(totalH), fnum(maxW), fnum(totalH), svgFontFamily, fnum(svgFontSize))
	b.WriteString("\n")
	b.WriteString(svgStyle)
	b.WriteString("\n")
	fmt.Fprintf(&b, `<rect width="%s" height="%s" fill="var(--bg)"/>`, fnum(maxW), fnum(totalH))
	b.WriteString("\n")
	b.WriteString(groups.String())
	b.WriteString("\n</svg>\n")
	return b.String()
}
