package main

import "strings"

type classInfo struct {
	annotation    string
	hasAnnotation bool
	attrs         []string
	methods       []string
}

type classOp struct {
	op       string
	headFrom int
	headTo   int
	line     int
}

var classOps = []classOp{
	{"<|--", headTriangle, headNone, lineSolid},
	{"--|>", headNone, headTriangle, lineSolid},
	{"<|..", headTriangle, headNone, lineDotted},
	{"..|>", headNone, headTriangle, lineDotted},
	{"*--", headDiamondFill, headNone, lineSolid},
	{"--*", headNone, headDiamondFill, lineSolid},
	{"o--", headDiamondOpen, headNone, lineSolid},
	{"--o", headNone, headDiamondOpen, lineSolid},
	{"<--", headArrow, headNone, lineSolid},
	{"-->", headNone, headArrow, lineSolid},
	{"<..", headArrow, headNone, lineDotted},
	{"..>", headNone, headArrow, lineDotted},
	{"--", headNone, headNone, lineSolid},
	{"..", headNone, headNone, lineDotted},
}

func syncInfos(g *graph, infos *[]classInfo) {
	for len(*infos) < len(g.nodes) {
		*infos = append(*infos, classInfo{})
	}
}

func parseClass(src string) (*graph, []classInfo, bool) {
	sts := statements(src)
	if len(sts) == 0 {
		return nil, nil, false
	}
	firstTok := strings.Fields(sts[0])
	if len(firstTok) == 0 || !strings.HasPrefix(strings.ToLower(firstTok[0]), "classdiagram") {
		return nil, nil, false
	}
	g := newGraph(dirDown)
	var infos []classInfo
	curClass := -1

	for _, st := range sts[1:] {
		if curClass >= 0 {
			if st == "}" {
				curClass = -1
			} else {
				pushMember(&infos[curClass], st)
			}
			continue
		}
		first := ""
		if fs := strings.Fields(st); len(fs) > 0 {
			first = strings.ToLower(fs[0])
		}
		switch first {
		case "direction":
			g.dir = dirDown
			if fs := strings.Fields(st); len(fs) > 1 {
				g.dir = parseDir(fs[1])
			}
			continue
		case "note", "callback", "click", "link", "style", "cssclass", "classdef", "namespace", "}":
			continue
		case "class":
			rest := strings.TrimSpace(st[len("class"):])
			name := rest
			open := false
			if n, ok := strings.CutSuffix(rest, "{"); ok {
				name = strings.TrimSpace(n)
				open = true
			}
			if name == "" || containsWhitespace(name) {
				return nil, nil, false
			}
			idx, ok := g.nodeIndex(name, "", false, shapeRect)
			if !ok {
				return nil, nil, false
			}
			syncInfos(g, &infos)
			if open {
				curClass = idx
			}
			continue
		}
		if ann, ok := strings.CutPrefix(st, "<<"); ok {
			a, rest, ok := strings.Cut(ann, ">>")
			if !ok {
				return nil, nil, false
			}
			name := strings.TrimSpace(rest)
			if name == "" || containsWhitespace(name) {
				return nil, nil, false
			}
			idx, ok := g.nodeIndex(name, "", false, shapeRect)
			if !ok {
				return nil, nil, false
			}
			syncInfos(g, &infos)
			infos[idx].annotation = strings.TrimSpace(a)
			infos[idx].hasAnnotation = true
			continue
		}
		if from, to, headFrom, headTo, line, label, hasLabel, ok := parseClassRelation(st); ok {
			f, ok := g.nodeIndex(from, "", false, shapeRect)
			if !ok {
				return nil, nil, false
			}
			syncInfos(g, &infos)
			t, ok := g.nodeIndex(to, "", false, shapeRect)
			if !ok {
				return nil, nil, false
			}
			syncInfos(g, &infos)
			if len(g.edges) >= maxEdges {
				return nil, nil, false
			}
			g.edges = append(g.edges, edge{
				from: f, to: t, label: label, hasLabel: hasLabel,
				headTo: headTo, headFrom: headFrom, line: line,
			})
			continue
		}
		if id, member, ok := strings.Cut(st, ":"); ok {
			id = strings.TrimSpace(id)
			member = strings.TrimSpace(member)
			if id == "" || containsWhitespace(id) || member == "" {
				return nil, nil, false
			}
			idx, ok := g.nodeIndex(id, "", false, shapeRect)
			if !ok {
				return nil, nil, false
			}
			syncInfos(g, &infos)
			pushMember(&infos[idx], member)
			continue
		}
		return nil, nil, false
	}

	if len(g.nodes) == 0 {
		return nil, nil, false
	}
	syncInfos(g, &infos)
	return g, infos, true
}

func pushMember(info *classInfo, raw string) {
	if ann, ok := strings.CutPrefix(raw, "<<"); ok {
		if a, _, ok := strings.Cut(ann, ">>"); ok {
			info.annotation = strings.TrimSpace(a)
			info.hasAnnotation = true
		}
		return
	}
	member := decodeHTMLEntities(displayGenerics(strings.TrimSpace(raw)))
	list := &info.attrs
	if strings.ContainsRune(member, '(') {
		list = &info.methods
	}
	if len(*list) < maxMembers {
		*list = append(*list, member)
	} else if len(*list) == maxMembers {
		*list = append(*list, "…")
	}
}

func parseClassRelation(st string) (from, to string, headFrom, headTo, line int, label string, hasLabel, ok bool) {
	chars := []rune(st)
	foundPos := -1
	var found classOp
	for pos := 0; pos < len(chars) && foundPos < 0; pos++ {
		for _, co := range classOps {
			opRunes := []rune(co.op)
			if !runesHasPrefix(chars, pos, opRunes) {
				continue
			}
			if strings.HasPrefix(co.op, "o") && pos > 0 && isIDChar(chars[pos-1]) {
				continue
			}
			if strings.HasSuffix(co.op, "o") && pos+len(opRunes) < len(chars) && isIDChar(chars[pos+len(opRunes)]) {
				continue
			}
			foundPos = pos
			found = co
			break
		}
	}
	if foundPos < 0 {
		return "", "", 0, 0, 0, "", false, false
	}
	opRunes := []rune(found.op)
	lhs := strings.TrimSpace(string(chars[:foundPos]))
	rhs := strings.TrimSpace(string(chars[foundPos+len(opRunes):]))

	lhs, cardFrom := stripCardinalitySuffix(lhs)
	rhs, cardTo := stripCardinalityPrefix(rhs)
	toID := strings.TrimSpace(rhs)
	relLabel := ""
	if t, l, ok := strings.Cut(rhs, ":"); ok {
		toID = strings.TrimSpace(t)
		relLabel, _ = nonEmpty(decodeHTMLEntities(strings.TrimSpace(l)))
	}
	lhs = strings.TrimSpace(lhs)
	if lhs == "" || toID == "" || containsWhitespace(lhs) || containsWhitespace(toID) {
		return "", "", 0, 0, 0, "", false, false
	}
	var parts []string
	for _, s := range []string{cardFrom, relLabel, cardTo} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	lbl, hasLbl := nonEmpty(strings.Join(parts, " "))
	return lhs, toID, found.headFrom, found.headTo, found.line, lbl, hasLbl, true
}

func stripCardinalitySuffix(s string) (string, string) {
	t := strings.TrimRight(s, " \t")
	if rest, ok := strings.CutSuffix(t, "\""); ok {
		if q := strings.LastIndexByte(rest, '"'); q >= 0 {
			return strings.TrimRight(rest[:q], " \t"), rest[q+1:]
		}
	}
	return t, ""
}

func stripCardinalityPrefix(s string) (string, string) {
	t := strings.TrimLeft(s, " \t")
	if rest, ok := strings.CutPrefix(t, "\""); ok {
		if q := strings.IndexByte(rest, '"'); q >= 0 {
			return strings.TrimLeft(rest[q+1:], " \t"), rest[:q]
		}
	}
	return t, ""
}

func displayGenerics(s string) string {
	var out strings.Builder
	open := false
	for _, c := range s {
		if c == '~' {
			if open {
				out.WriteRune('>')
			} else {
				out.WriteRune('<')
			}
			open = !open
		} else {
			out.WriteRune(c)
		}
	}
	return out.String()
}

// --- ER diagrams ---

func parseER(src string) (*graph, []classInfo, bool) {
	sts := statements(src)
	if len(sts) == 0 {
		return nil, nil, false
	}
	firstTok := strings.Fields(sts[0])
	if len(firstTok) == 0 || !strings.EqualFold(firstTok[0], "erdiagram") {
		return nil, nil, false
	}
	g := newGraph(dirDown)
	var infos []classInfo
	curEntity := -1

	for _, st := range sts[1:] {
		if curEntity >= 0 {
			if st == "}" {
				curEntity = -1
			} else {
				pushERAttribute(&infos[curEntity], st)
			}
			continue
		}
		if rel, labelPart, hasLabelPart, ok := splitERRelationship(st); ok {
			tokens := strings.Fields(rel)
			if len(tokens) != 3 {
				return nil, nil, false
			}
			cardL, cardR, line, ok := parseEROp(tokens[1])
			if !ok {
				return nil, nil, false
			}
			f, ok := erEntity(g, &infos, tokens[0])
			if !ok {
				return nil, nil, false
			}
			t, ok := erEntity(g, &infos, tokens[2])
			if !ok {
				return nil, nil, false
			}
			if len(g.edges) >= maxEdges {
				return nil, nil, false
			}
			relLabel := ""
			if hasLabelPart {
				relLabel = cleanLabel(labelPart)
			}
			var parts []string
			for _, s := range []string{cardL, relLabel, cardR} {
				if s != "" {
					parts = append(parts, s)
				}
			}
			label, hasLabel := nonEmpty(strings.Join(parts, " "))
			g.edges = append(g.edges, edge{
				from: f, to: t, label: label, hasLabel: hasLabel,
				headTo: headNone, headFrom: headNone, line: line,
			})
			continue
		}
		decl := st
		open := false
		if d, ok := strings.CutSuffix(st, "{"); ok {
			decl = strings.TrimSpace(d)
			open = true
		}
		if decl == "" || len(strings.Fields(decl)) != 1 {
			return nil, nil, false
		}
		idx, ok := erEntity(g, &infos, decl)
		if !ok {
			return nil, nil, false
		}
		if open {
			curEntity = idx
		}
	}

	if len(g.nodes) == 0 {
		return nil, nil, false
	}
	syncInfos(g, &infos)
	return g, infos, true
}

func erEntity(g *graph, infos *[]classInfo, token string) (int, bool) {
	var idx int
	var ok bool
	if open := strings.IndexByte(token, '['); open >= 0 {
		id := token[:open]
		label := cleanLabel(strings.TrimRight(token[open+1:], "]"))
		if id == "" || label == "" {
			return 0, false
		}
		idx, ok = g.nodeLabel(id, label)
	} else {
		idx, ok = g.nodeIndex(token, "", false, shapeRect)
	}
	if !ok {
		return 0, false
	}
	syncInfos(g, infos)
	return idx, true
}

func splitERRelationship(st string) (rel, label string, hasLabel, ok bool) {
	rel = st
	if r, l, cut := strings.Cut(st, ":"); cut {
		rel = r
		label = strings.TrimSpace(l)
		hasLabel = true
	}
	for _, tok := range strings.Fields(rel) {
		if _, _, _, ok := parseEROp(tok); ok {
			return rel, label, hasLabel, true
		}
	}
	return "", "", false, false
}

func parseEROp(tok string) (string, string, int, bool) {
	if len(tok) != 6 {
		return "", "", 0, false
	}
	for i := 0; i < len(tok); i++ {
		if tok[i] > 127 {
			return "", "", 0, false
		}
	}
	var line int
	switch tok[2:4] {
	case "--":
		line = lineSolid
	case "..":
		line = lineDotted
	default:
		return "", "", 0, false
	}
	l, ok := erCard(tok[:2])
	if !ok {
		return "", "", 0, false
	}
	r, ok := erCard(tok[4:6])
	if !ok {
		return "", "", 0, false
	}
	return l, r, line, true
}

func erCard(tok string) (string, bool) {
	switch tok {
	case "|o", "o|":
		return "0..1", true
	case "||":
		return "1", true
	case "}o", "o{":
		return "0..*", true
	case "}|", "|{":
		return "1..*", true
	}
	return "", false
}

func pushERAttribute(info *classInfo, raw string) {
	var parts []string
	for _, tok := range strings.Fields(raw) {
		if strings.HasPrefix(tok, "\"") {
			break
		}
		parts = append(parts, tok)
	}
	if len(parts) == 0 {
		return
	}
	line := decodeHTMLEntities(strings.Join(parts, " "))
	if len(info.attrs) < maxMembers {
		info.attrs = append(info.attrs, line)
	} else if len(info.attrs) == maxMembers {
		info.attrs = append(info.attrs, "…")
	}
}
