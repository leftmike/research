package main

import "strings"

// render renders a mermaid source block to styled lines, or nil for blank input.
// maxWidth<=0 means no width limit.
func render(src string, maxWidth int) []span2D {
	if strings.TrimSpace(src) == "" {
		return nil
	}

	lines, outcome, ok := tryRenderers(src, maxWidth)
	if ok {
		if outcome == ovOK {
			return lines
		}
	}
	tooWide := outcome == ovWidth
	return fallback(src, maxWidth, tooWide)
}

// tryRenderers walks the parser chain in the same order as the Rust renderer,
// returning the first diagram type that parses. ok reports whether any parser
// matched; outcome carries the layout result (ovOK / ovWidth / ovCells).
func tryRenderers(src string, maxWidth int) (lines []span2D, outcome int, ok bool) {
	if g := parseGraph(src); g != nil {
		if len(g.groups) == 0 {
			l, ov := layoutFlowchart(g, maxWidth)
			return l, ov, true
		}
		l, ov := renderGrouped(g, maxWidth)
		return l, ov, true
	}
	if g := parseState(src); g != nil {
		l, ov := layoutFlowchart(g, maxWidth)
		return l, ov, true
	}
	if g, infos, okc := parseClass(src); okc {
		l, ov := renderClass(g, infos, maxWidth)
		return l, ov, true
	}
	if g, infos, oke := parseER(src); oke {
		l, ov := renderClass(g, infos, maxWidth)
		return l, ov, true
	}
	if seq := parseSequence(src); seq != nil {
		l, ov := layoutSequence(seq, maxWidth)
		return l, ov, true
	}
	return nil, ovCells, false
}

func renderClass(g *graph, infos []classInfo, maxWidth int) ([]span2D, int) {
	extras := make([]nodeExtra, len(g.nodes))
	for i := range g.nodes {
		var title []string
		if infos[i].hasAnnotation {
			title = append(title, "«"+infos[i].annotation+"»")
		}
		title = append(title, displayGenerics(g.nodes[i].label))
		extras[i] = nodeExtra{
			kind:    extraCompart,
			compart: [][]string{title, infos[i].attrs, infos[i].methods},
		}
	}
	c, ov := layoutCanvas(g, extras, maxWidth)
	if ov != ovOK {
		return nil, ov
	}
	switch g.dir {
	case dirUp:
		c.flipVertical()
	case dirLeft:
		c.flipHorizontal()
	}
	return c.toLines(), ovOK
}
