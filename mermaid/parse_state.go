package main

import "strings"

func parseState(src string) *graph {
	sts := statements(src)
	if len(sts) == 0 {
		return nil
	}
	firstTok := strings.Fields(sts[0])
	if len(firstTok) == 0 || !strings.HasPrefix(strings.ToLower(firstTok[0]), "statediagram") {
		return nil
	}
	g := newGraph(dirDown)
	inNote := false
	for _, st := range sts[1:] {
		if inNote {
			if strings.EqualFold(st, "end note") {
				inNote = false
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
		case "note":
			if !strings.ContainsRune(st, ':') {
				inNote = true
			}
		case "state":
			if !parseStateDecl(st, g) {
				return nil
			}
		case "classdef", "class", "hide", "scale", "}", "--":
			// ignore
		default:
			if strings.Contains(st, "-->") {
				if !parseTransition(st, g) {
					return nil
				}
			} else {
				if !parseStateDesc(st, g) {
					return nil
				}
			}
		}
		if g.overCap {
			return nil
		}
	}
	if len(g.nodes) == 0 {
		return nil
	}
	return g
}

func parseStateDecl(st string, g *graph) bool {
	rest := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(st[len("state"):]), "{"))
	if rest == "" {
		return true
	}
	if q, ok := strings.CutPrefix(rest, "\""); ok {
		label, after, ok := strings.Cut(q, "\"")
		if !ok {
			return false
		}
		id := label
		if a := strings.TrimSpace(after); strings.HasPrefix(a, "as") {
			id = strings.TrimSpace(a[len("as"):])
		}
		_, ok = g.nodeLabel(id, decodeHTMLEntities(label))
		return ok
	}
	shape := shapeRound
	id := rest
	stereotyped := false
	if pos := strings.Index(rest, "<<"); pos >= 0 {
		stereo := strings.TrimSpace(strings.TrimRight(rest[pos+2:], ">>"))
		if stereo == "choice" {
			shape = shapeDiamond
		}
		id = strings.TrimSpace(rest[:pos])
		stereotyped = true
	}
	if id == "" || containsWhitespace(id) {
		return false
	}
	if stereotyped {
		_, ok := g.nodeIndex(id, id, true, shape)
		return ok
	}
	_, ok := g.nodeIndex(id, "", false, shape)
	return ok
}

func parseTransition(st string, g *graph) bool {
	rest := st
	prev := -1
	for {
		lhs, rhs, ok := strings.Cut(rest, "-->")
		if !ok {
			break
		}
		fromID := strings.TrimSpace(strings.TrimRight(strings.TrimRight(lhs, " \t"), "-"))
		var from int
		if prev >= 0 {
			if fromID != "" {
				return false
			}
			from = prev
		} else {
			if fromID == "" {
				return false
			}
			f, ok := stateEndpoint(g, fromID, true)
			if !ok {
				return false
			}
			from = f
		}
		toPart := rhs
		tail := ""
		if t, _, ok := strings.Cut(rhs, "-->"); ok {
			toPart = t
			tail = rhs[len(t):]
		}
		label := ""
		hasLabel := false
		if t, l, ok := strings.Cut(toPart, ":"); ok {
			toPart = t
			label, hasLabel = nonEmpty(decodeHTMLEntities(strings.TrimSpace(l)))
		}
		toID := strings.TrimSpace(strings.TrimRight(strings.TrimRight(strings.TrimLeft(strings.TrimSpace(toPart), ">"), " \t"), "-"))
		if toID == "" {
			return false
		}
		to, ok := stateEndpoint(g, toID, false)
		if !ok {
			return false
		}
		if len(g.edges) >= maxEdges {
			g.overCap = true
			return true
		}
		g.edges = append(g.edges, edge{
			from: from, to: to, label: label, hasLabel: hasLabel,
			headTo: headArrow, headFrom: headNone, line: lineSolid,
		})
		prev = to
		rest = tail
	}
	return true
}

func stateEndpoint(g *graph, id string, isSource bool) (int, bool) {
	if id == "[*]" {
		key := "[*]end"
		if isSource {
			key = "[*]start"
		}
		return g.nodeIndex(key, "●", true, shapeRound)
	}
	return g.nodeIndex(id, "", false, shapeRound)
}

func parseStateDesc(st string, g *graph) bool {
	if id, desc, ok := strings.Cut(st, ":"); ok {
		id = strings.TrimSpace(id)
		desc = strings.TrimSpace(desc)
		if id == "" || containsWhitespace(id) || desc == "" {
			return false
		}
		_, ok := g.nodeLabel(id, decodeHTMLEntities(desc))
		return ok
	}
	if !containsWhitespace(st) {
		_, ok := g.nodeIndex(st, "", false, shapeRound)
		return ok
	}
	return false
}
