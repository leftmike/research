package main

import "strings"

// splitStatements splits a source line into statements on ';', respecting quotes
// and stopping at a %% comment. Statements are appended to out.
func splitStatements(line string, out *[]string) {
	var cur strings.Builder
	inQuotes := false
	chars := []rune(line)
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		if inQuotes {
			if c == '"' {
				inQuotes = false
			}
			cur.WriteRune(c)
			continue
		}
		switch {
		case c == '"':
			inQuotes = true
			cur.WriteRune(c)
		case c == '%' && i+1 < len(chars) && chars[i+1] == '%':
			flushStatement(&cur, out)
			return
		case c == ';':
			flushStatement(&cur, out)
		default:
			cur.WriteRune(c)
		}
	}
	flushStatement(&cur, out)
}

func flushStatement(cur *strings.Builder, out *[]string) {
	trimmed := strings.TrimSpace(cur.String())
	if trimmed != "" {
		*out = append(*out, trimmed)
	}
	cur.Reset()
}

func statements(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		splitStatements(line, &out)
	}
	return out
}

func parseGraph(src string) *graph {
	sts := statements(src)
	if len(sts) == 0 {
		return nil
	}
	headerTokens := strings.Fields(sts[0])
	if len(headerTokens) == 0 {
		return nil
	}
	kind := strings.ToLower(headerTokens[0])
	if kind != "graph" && kind != "flowchart" {
		return nil
	}
	dir := dirDown
	if len(headerTokens) > 1 {
		dir = parseDir(headerTokens[1])
	}
	g := newGraph(dir)

	var stack []int
	for _, st := range sts[1:] {
		firstWord := ""
		if fs := strings.Fields(st); len(fs) > 0 {
			firstWord = strings.ToLower(fs[0])
		}
		switch firstWord {
		case "subgraph":
			if len(g.groups) >= maxGroups || len(stack) >= maxGroupDepth {
				return nil
			}
			id, label := parseSubgraphDecl(strings.TrimSpace(st[len("subgraph"):]))
			parent := -1
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			g.groups = append(g.groups, group{id: id, label: label, parent: parent})
			stack = append(stack, len(g.groups)-1)
			g.curGroup = stack[len(stack)-1]
			continue
		case "end":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			g.curGroup = -1
			if len(stack) > 0 {
				g.curGroup = stack[len(stack)-1]
			}
			continue
		case "classdef", "class", "style", "linkstyle", "click", "direction":
			continue
		}
		parseStatement(st, g)
		if g.overCap {
			return nil
		}
	}

	if len(g.nodes) == 0 {
		return nil
	}
	return g
}

func parseDir(tok string) int {
	switch strings.ToUpper(tok) {
	case "LR":
		return dirRight
	case "RL":
		return dirLeft
	case "BT":
		return dirUp
	default:
		return dirDown
	}
}

func parseSubgraphDecl(rest string) (string, string) {
	if q, ok := strings.CutPrefix(rest, "\""); ok {
		if label, _, ok := strings.Cut(q, "\""); ok {
			return label, decodeHTMLEntities(label)
		}
	}
	if open := strings.IndexByte(rest, '['); open >= 0 {
		id := strings.TrimSpace(rest[:open])
		label := cleanLabel(strings.TrimSpace(strings.TrimRight(rest[open+1:], "]")))
		if id != "" && label != "" {
			return id, label
		}
	}
	return rest, rest
}

func parseStatement(st string, g *graph) {
	chars := []rune(st)
	i := 0
	prev, ni, ok := parseNodeGroup(chars, i, g)
	if !ok {
		return
	}
	i = ni
	for {
		i = skipSpaces(chars, i)
		if i >= len(chars) {
			break
		}
		left, right, line, label, hasLabel, ni, ok := parseLink(chars, i)
		if !ok {
			break
		}
		i = skipSpaces(chars, ni)
		next, ni2, ok := parseNodeGroup(chars, i, g)
		if !ok {
			break
		}
		i = ni2
		for _, f := range prev {
			for _, t := range next {
				if len(g.edges) >= maxEdges {
					g.overCap = true
					return
				}
				var from, to, headTo, headFrom int
				if left == headArrow && right != headArrow {
					from, to, headTo, headFrom = t, f, headArrow, right
				} else {
					from, to, headTo, headFrom = f, t, right, left
				}
				g.edges = append(g.edges, edge{
					from: from, to: to, label: label, hasLabel: hasLabel,
					headTo: headTo, headFrom: headFrom, line: line,
				})
			}
		}
		prev = next
	}
}

func parseNodeGroup(chars []rune, start int, g *graph) ([]int, int, bool) {
	first, i, ok := parseNode(chars, start, g)
	if !ok {
		return nil, 0, false
	}
	grp := []int{first}
	for {
		j := skipSpaces(chars, i)
		if j >= len(chars) || chars[j] != '&' {
			break
		}
		next, k, ok := parseNode(chars, j+1, g)
		if !ok {
			return nil, 0, false
		}
		grp = append(grp, next)
		i = k
	}
	return grp, i, true
}

func parseNode(chars []rune, start int, g *graph) (int, int, bool) {
	i := skipSpaces(chars, start)
	idStart := i
	for i < len(chars) && isIDChar(chars[i]) {
		i++
	}
	if i == idStart {
		return 0, 0, false
	}
	id := string(chars[idStart:i])

	shape := shapeRect
	label := ""
	hasLabel := false
	after := i
	get := func(k int) (rune, bool) {
		if k < len(chars) {
			return chars[k], true
		}
		return 0, false
	}
	c, has := get(i)
	if has {
		switch c {
		case '[':
			if n, _ := get(i + 1); n == '[' {
				shape, label, hasLabel, after = readShape(chars, i+2, "]]", shapeRect)
			} else if n == '(' {
				shape, label, hasLabel, after = readShape(chars, i+2, ")]", shapeRound)
			} else {
				shape, label, hasLabel, after = readShape(chars, i+1, "]", shapeRect)
			}
		case '(':
			if n, _ := get(i + 1); n == '(' {
				shape, label, hasLabel, after = readShape(chars, i+2, "))", shapeRound)
			} else if n == '[' {
				shape, label, hasLabel, after = readShape(chars, i+2, "])", shapeRound)
			} else {
				shape, label, hasLabel, after = readShape(chars, i+1, ")", shapeRound)
			}
		case '{':
			if n, _ := get(i + 1); n == '{' {
				shape, label, hasLabel, after = readShape(chars, i+2, "}}", shapeDiamond)
			} else {
				shape, label, hasLabel, after = readShape(chars, i+1, "}", shapeDiamond)
			}
		case '>':
			shape, label, hasLabel, after = readShape(chars, i+1, "]", shapeRect)
		}
	}

	idx, ok := g.nodeIndex(id, label, hasLabel, shape)
	if !ok {
		return 0, 0, false
	}
	return idx, after, true
}

// readShape reads a bracketed node label up to closer and returns the shape,
// cleaned label, and the index after the closer.
func readShape(chars []rune, start int, closerStr string, shape int) (int, string, bool, int) {
	closer := []rune(closerStr)
	i := start
	var text strings.Builder
	quoted := false
	for j := start; j < len(chars); {
		if chars[j] == ' ' || chars[j] == '\t' {
			j++
			continue
		}
		quoted = chars[j] == '"'
		break
	}
	inQuotes := false
	for i < len(chars) {
		c := chars[i]
		if quoted && c == '"' {
			inQuotes = !inQuotes
			text.WriteRune(c)
			i++
			continue
		}
		if !inQuotes && runesHasPrefix(chars, i, closer) {
			return shape, cleanLabel(text.String()), true, i + len(closer)
		}
		text.WriteRune(c)
		i++
	}
	return shape, cleanLabel(text.String()), true, len(chars)
}

// parseLink parses an edge operator with optional label, returning the left and
// right heads, line kind, label, and the index after the operator.
func parseLink(chars []rune, start int) (left, right, line int, label string, hasLabel bool, next int, ok bool) {
	i := skipSpaces(chars, start)
	left = headNone
	if i < len(chars) && (chars[i] == 'o' || chars[i] == 'x') && i+1 < len(chars) {
		if n := chars[i+1]; n == '-' || n == '.' || n == '=' {
			if chars[i] == 'o' {
				left = headCircle
			} else {
				left = headCross
			}
			i++
		}
	}
	opStart := i
	for i < len(chars) && isLinkChar(chars[i]) {
		i++
	}
	if i == opStart {
		return 0, 0, 0, "", false, 0, false
	}
	op1 := string(chars[opStart:i])
	if left == headNone && strings.HasPrefix(op1, "<") {
		left = headArrow
	}
	line = lineKind(op1)
	right = headNone
	if strings.ContainsRune(op1, '>') {
		right = headArrow
	}
	if right == headNone {
		if head, ni, ok := trailingHead(chars, i); ok {
			right = head
			i = ni
		}
	}

	if i < len(chars) && chars[i] == '|' {
		i++
		lStart := i
		for i < len(chars) && chars[i] != '|' {
			i++
		}
		lbl := cleanLabel(string(chars[lStart:i]))
		if i < len(chars) && chars[i] == '|' {
			i++
		}
		l, has := nonEmpty(lbl)
		return left, right, line, l, has, i, true
	}

	if right == headNone {
		textStart := skipSpaces(chars, i)
		j := textStart
		for j < len(chars) && !isLinkChar(chars[j]) {
			j++
		}
		if j < len(chars) && j > textStart && (chars[j] == '-' || chars[j] == '.' || chars[j] == '=' || chars[j] == '>') {
			text := string(chars[textStart:j])
			op2Start := j
			for j < len(chars) && isLinkChar(chars[j]) {
				j++
			}
			op2 := string(chars[op2Start:j])
			switch {
			case strings.ContainsRune(op2, '>'):
				right = headArrow
			default:
				if head, nj, ok := trailingHead(chars, j); ok {
					j = nj
					right = head
				} else {
					right = headNone
				}
			}
			if line == lineSolid {
				line = lineKind(op2)
			}
			l, has := nonEmpty(cleanLabel(text))
			return left, right, line, l, has, j, true
		}
	}

	return left, right, line, "", false, i, true
}

func lineKind(op string) int {
	switch {
	case strings.ContainsRune(op, '='):
		return lineThick
	case strings.ContainsRune(op, '.'):
		return lineDotted
	default:
		return lineSolid
	}
}

func trailingHead(chars []rune, i int) (int, int, bool) {
	if i >= len(chars) {
		return 0, 0, false
	}
	var head int
	switch chars[i] {
	case 'o':
		head = headCircle
	case 'x':
		head = headCross
	default:
		return 0, 0, false
	}
	if i+1 >= len(chars) {
		return head, i + 1, true
	}
	switch chars[i+1] {
	case ' ', '\t', '|', '&', ';':
		return head, i + 1, true
	}
	return 0, 0, false
}

func nonEmpty(s string) (string, bool) {
	if s == "" {
		return "", false
	}
	return s, true
}
