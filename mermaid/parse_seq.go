package main

import "strings"

const (
	seqHeadArrow = iota
	seqHeadCross
)

type seqOp struct {
	op     string
	dashed bool
	head   int
}

var seqOps = []seqOp{
	{"-->>", true, seqHeadArrow},
	{"->>", false, seqHeadArrow},
	{"--x", true, seqHeadCross},
	{"-x", false, seqHeadCross},
	{"--)", true, seqHeadArrow},
	{"-)", false, seqHeadArrow},
	{"-->", true, seqHeadArrow},
	{"->", false, seqHeadArrow},
}

const (
	noteOver = iota
	noteLeft
	noteRight
)

type noteAnchor struct {
	kind int
	a, b int // participant indices; b used only for Over
}

const (
	seqMessage = iota
	seqNote
	seqDivider
)

type seqItem struct {
	kind int
	// message
	from, to int
	text     string
	hasText  bool
	dashed   bool
	head     int
	// note
	anchor noteAnchor
	// note/divider text stored in noteText
	noteText string
}

type sequence struct {
	labels []string
	index  map[string]int
	items  []seqItem
}

func (s *sequence) participant(id string, label string, hasLabel bool) (int, bool) {
	if i, ok := s.index[id]; ok {
		if hasLabel {
			s.labels[i] = label
		}
		return i, true
	}
	if len(s.labels) >= maxNodes {
		return 0, false
	}
	s.index[id] = len(s.labels)
	if !hasLabel {
		label = id
	}
	s.labels = append(s.labels, label)
	return len(s.labels) - 1, true
}

func parseSequence(src string) *sequence {
	sts := statements(src)
	if len(sts) == 0 {
		return nil
	}
	firstTok := strings.Fields(sts[0])
	if len(firstTok) == 0 || !strings.EqualFold(firstTok[0], "sequencediagram") {
		return nil
	}
	seq := &sequence{index: map[string]int{}}
	autonumber := false
	msgCount := 0
	var blocks []bool

	for _, st := range sts[1:] {
		first := ""
		if fs := strings.Fields(st); len(fs) > 0 {
			first = strings.ToLower(fs[0])
		}
		switch first {
		case "participant", "actor":
			rest := strings.TrimSpace(st[len(first):])
			if rest == "" {
				return nil
			}
			id := rest
			label := ""
			hasLabel := false
			if i, l, ok := strings.Cut(rest, " as "); ok {
				id = strings.TrimSpace(i)
				label = cleanLabel(l)
				hasLabel = true
			}
			if _, ok := seq.participant(id, label, hasLabel); !ok {
				return nil
			}
		case "autonumber":
			autonumber = true
		case "activate", "deactivate", "create", "destroy", "title", "acctitle",
			"accdescr", "links", "link", "properties":
			// ignore
		case "note":
			rest := strings.TrimSpace(st[len(first):])
			textPart, anchor, ok := parseNoteAnchor(rest, seq)
			if !ok {
				return nil
			}
			if len(seq.items) >= maxEdges {
				return nil
			}
			seq.items = append(seq.items, seqItem{kind: seqNote, anchor: anchor, noteText: textPart})
		case "loop", "alt", "opt", "par", "critical", "break", "else", "and", "option":
			if first == "else" || first == "and" || first == "option" {
				if len(blocks) == 0 || !blocks[len(blocks)-1] {
					continue
				}
			} else {
				blocks = append(blocks, true)
			}
			if len(seq.items) >= maxEdges {
				return nil
			}
			seq.items = append(seq.items, seqItem{kind: seqDivider, noteText: decodeHTMLEntities(st)})
		case "rect", "box":
			blocks = append(blocks, false)
		case "end":
			if len(blocks) > 0 {
				top := blocks[len(blocks)-1]
				blocks = blocks[:len(blocks)-1]
				if top {
					if len(seq.items) >= maxEdges {
						return nil
					}
					seq.items = append(seq.items, seqItem{kind: seqDivider, noteText: "end"})
				}
			}
		default:
			from, to, text, hasText, dashed, head, ok := parseSeqMessage(st, seq)
			if !ok {
				return nil
			}
			if autonumber {
				msgCount++
				if hasText {
					text = itoa(msgCount) + ". " + text
				} else {
					text = itoa(msgCount) + "."
				}
				hasText = true
			}
			if len(seq.items) >= maxEdges {
				return nil
			}
			seq.items = append(seq.items, seqItem{
				kind: seqMessage, from: from, to: to, text: text, hasText: hasText,
				dashed: dashed, head: head,
			})
		}
	}

	if len(seq.labels) == 0 {
		return nil
	}
	return seq
}

func parseNoteAnchor(rest string, seq *sequence) (string, noteAnchor, bool) {
	lower := strings.ToLower(rest)
	var idsAndText string
	kind := 0
	switch {
	case strings.HasPrefix(lower, "over "):
		idsAndText = rest[len("over "):]
		kind = noteOver
	case strings.HasPrefix(lower, "left of "):
		idsAndText = rest[len("left of "):]
		kind = noteLeft
	case strings.HasPrefix(lower, "right of "):
		idsAndText = rest[len("right of "):]
		kind = noteRight
	default:
		return "", noteAnchor{}, false
	}
	ids, text, ok := strings.Cut(idsAndText, ":")
	if !ok {
		return "", noteAnchor{}, false
	}
	text = decodeHTMLEntities(strings.TrimSpace(text))
	var parts []string
	for _, p := range strings.Split(ids, ",") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "", noteAnchor{}, false
	}
	a, ok := seq.participant(parts[0], "", false)
	if !ok {
		return "", noteAnchor{}, false
	}
	switch kind {
	case noteOver:
		b := a
		if len(parts) > 1 {
			bb, ok := seq.participant(parts[1], "", false)
			if !ok {
				return "", noteAnchor{}, false
			}
			b = bb
		}
		return text, noteAnchor{kind: noteOver, a: minInt(a, b), b: maxInt(a, b)}, true
	case noteLeft:
		return text, noteAnchor{kind: noteLeft, a: a}, true
	default:
		return text, noteAnchor{kind: noteRight, a: a}, true
	}
}

func parseSeqMessage(st string, seq *sequence) (from, to int, text string, hasText, dashed bool, head int, ok bool) {
	pos := -1
	var opStr string
	runes := []rune(st)
	for p := 0; p < len(runes) && pos < 0; p++ {
		for _, so := range seqOps {
			if runesHasPrefix(runes, p, []rune(so.op)) {
				pos = p
				opStr = so.op
				dashed = so.dashed
				head = so.head
				break
			}
		}
	}
	if pos < 0 {
		return 0, 0, "", false, false, 0, false
	}
	fromID := strings.TrimSpace(string(runes[:pos]))
	if fromID == "" {
		return 0, 0, "", false, false, 0, false
	}
	rest := strings.TrimLeft(string(runes[pos+len([]rune(opStr)):]), " \t")
	rest = strings.TrimLeft(rest, "+-")
	toID := strings.TrimSpace(rest)
	if t, txt, cut := strings.Cut(rest, ":"); cut {
		toID = strings.TrimSpace(t)
		text, hasText = nonEmpty(decodeHTMLEntities(strings.TrimSpace(txt)))
	}
	if toID == "" {
		return 0, 0, "", false, false, 0, false
	}
	f, ok := seq.participant(fromID, "", false)
	if !ok {
		return 0, 0, "", false, false, 0, false
	}
	t, ok := seq.participant(toID, "", false)
	if !ok {
		return 0, 0, "", false, false, 0, false
	}
	return f, t, text, hasText, dashed, head, true
}
