package main

import "strings"

const tooWideHint = "This diagram is too wide to display here — open the image to view it in full."

// fallback frames the raw source in a titled box for unsupported or oversize
// diagrams. limit<0 means no width limit.
func fallback(src string, maxWidth int, tooWide bool) []span2D {
	header := firstWord(src)
	title := " mermaid: " + header + " "
	limit := -1
	if maxWidth > 0 {
		limit = maxInt(maxWidth-4, 8)
	}

	var body []string
	skipping := true
	for _, l := range strings.Split(src, "\n") {
		l = strings.TrimRight(l, " \t")
		if skipping {
			if l == "" {
				continue
			}
			skipping = false
		}
		body = append(body, chunkLine(l, limit)...)
	}

	contentW := strWidth(title)
	for _, l := range body {
		contentW = maxInt(contentW, strWidth(l))
	}
	inner := contentW + 2

	var out []span2D

	dashes := satSub(inner, strWidth(title))
	out = append(out, span2D{
		{"╭", clsBorder},
		{title, clsEdgeLabel},
		{strings.Repeat("─", dashes) + "╮", clsBorder},
	})

	for _, line := range body {
		padN := satSub(contentW, strWidth(line))
		out = append(out, span2D{
			{"│ ", clsBorder},
			{line, clsText},
			{strings.Repeat(" ", padN) + " │", clsBorder},
		})
	}

	out = append(out, span2D{{"╰" + strings.Repeat("─", inner) + "╯", clsBorder}})

	if tooWide {
		for _, chunk := range wrapWords(tooWideHint, maxWidth) {
			out = append(out, span2D{{chunk, clsBorder}})
		}
	}
	return out
}

func chunkLine(line string, limit int) []string {
	if limit < 0 || strWidth(line) <= limit {
		return []string{line}
	}
	var out []string
	var cur strings.Builder
	curW := 0
	for _, r := range line {
		cw := maxInt(charWidth(r), 1)
		if curW+cw > limit && cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteRune(r)
		curW += cw
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func wrapWords(text string, limit int) []string {
	if limit <= 0 {
		return []string{text}
	}
	var lines []string
	var cur strings.Builder
	for _, word := range strings.Split(text, " ") {
		if word == "" {
			continue
		}
		switch {
		case cur.Len() == 0:
			cur.WriteString(word)
		case strWidth(cur.String())+1+strWidth(word) <= limit:
			cur.WriteByte(' ')
			cur.WriteString(word)
		default:
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(word)
		}
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	var out []string
	for _, l := range lines {
		out = append(out, chunkLine(l, limit)...)
	}
	return out
}

func firstWord(src string) string {
	if fs := strings.Fields(src); len(fs) > 0 {
		return fs[0]
	}
	return "diagram"
}
