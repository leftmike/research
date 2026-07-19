package main

func noteGeometry(xs []int, anchor noteAnchor, textW int) (int, int) {
	switch anchor.kind {
	case noteOver:
		l, r := anchor.a, anchor.b
		center := (xs[l] + xs[r]) / 2
		w := maxInt(xs[r]-xs[l]+5, textW+2*pad+2)
		return satSub(center, w/2), w
	case noteLeft:
		w := textW + 2*pad + 2
		return satSub(xs[anchor.a], 2+w-1), w
	default: // noteRight
		return xs[anchor.a] + 2, textW + 2*pad + 2
	}
}

func layoutSequence(seq *sequence, maxWidth int) ([]span2D, int) {
	n := len(seq.labels)
	labels := make([]string, n)
	for i, l := range seq.labels {
		labels[i] = fitLabel(l, wrapWidth)
	}
	boxW := make([]int, n)
	for i, l := range labels {
		boxW[i] = maxInt(strWidth(l), 1) + 2*pad + 2
	}
	boxH := 3

	itemTextW := func(it seqItem) int {
		if it.hasText {
			return strWidth(it.text)
		}
		return 0
	}

	gaps := make([]int, satSub(n, 1))
	for i := range gaps {
		gaps[i] = maxInt(seqGap, divCeil(boxW[i], 2)+divCeil(boxW[i+1], 2)+1)
	}

	var reqs []seqReq
	for _, it := range seq.items {
		switch it.kind {
		case seqMessage:
			tw := itemTextW(it)
			if it.from != it.to {
				l, r := minInt(it.from, it.to), maxInt(it.from, it.to)
				reqs = append(reqs, seqReq{l, r, maxInt(tw+2, 4)})
			} else if it.from+1 < n {
				reqs = append(reqs, seqReq{it.from, it.from + 1, 5 + tw + 2})
			}
		case seqNote:
			tw := strWidth(it.noteText)
			switch it.anchor.kind {
			case noteOver:
				if it.anchor.a < it.anchor.b {
					reqs = append(reqs, seqReq{it.anchor.a, it.anchor.b, satSub(tw, 1)})
				} else {
					i := it.anchor.a
					half := divCeil(tw+4, 2) + 2
					if i > 0 {
						reqs = append(reqs, seqReq{i - 1, i, half})
					}
					if i+1 < n {
						reqs = append(reqs, seqReq{i, i + 1, half})
					}
				}
			case noteLeft:
				if it.anchor.a > 0 {
					reqs = append(reqs, seqReq{it.anchor.a - 1, it.anchor.a, tw + 7})
				}
			case noteRight:
				if it.anchor.a+1 < n {
					reqs = append(reqs, seqReq{it.anchor.a, it.anchor.a + 1, tw + 7})
				}
			}
		}
	}
	// stable sort by span width (r-l)
	stableSortReqs(reqs)
	for _, rq := range reqs {
		cur := 0
		for i := rq.l; i < rq.r; i++ {
			cur += gaps[i]
		}
		if cur < rq.need {
			gaps[rq.r-1] += rq.need - cur
		}
	}

	xs := make([]int, n)
	if n > 0 {
		xs[0] = boxW[0] / 2
	}
	for i := 1; i < n; i++ {
		xs[i] = xs[i-1] + gaps[i-1]
	}

	canvasW := 0
	if n > 0 {
		canvasW = xs[n-1] + divCeil(boxW[n-1], 2) + 1
	}
	for _, it := range seq.items {
		switch it.kind {
		case seqMessage:
			if it.from == it.to {
				canvasW = maxInt(canvasW, xs[it.from]+5+itemTextW(it)+1)
			}
		case seqNote:
			x, w := noteGeometry(xs, it.anchor, strWidth(it.noteText))
			canvasW = maxInt(canvasW, x+w+1)
		case seqDivider:
			canvasW = maxInt(canvasW, strWidth(it.noteText)+4)
		}
	}

	rows := make([]int, len(seq.items))
	y := boxH + 1
	for i, it := range seq.items {
		rows[i] = y
		switch it.kind {
		case seqMessage:
			if it.from == it.to {
				y += 4
			} else if it.hasText {
				y += 3
			} else {
				y += 2
			}
		case seqNote:
			y += 4
		case seqDivider:
			y += 2
		}
	}
	bottomTop := y
	canvasH := bottomTop + boxH

	if maxWidth > 0 && canvasW > maxWidth {
		return nil, ovWidth
	}
	if canvasW*canvasH > maxCanvasCell {
		return nil, ovCells
	}

	c := newCanvas(canvasW, canvasH)
	for i := 0; i < n; i++ {
		for _, by := range []int{0, bottomTop} {
			p := placed{
				x:  satSub(xs[i], boxW[i]/2),
				y:  by,
				w:  boxW[i],
				h:  boxH,
				cx: xs[i],
				cy: by + 1,
			}
			drawBox(c, p, []string{labels[i]}, shapeRect)
		}
	}
	for i, it := range seq.items {
		if it.kind == seqNote {
			x, w := noteGeometry(xs, it.anchor, strWidth(it.noteText))
			p := placed{x: x, y: rows[i], w: w, h: 3, cx: x + w/2, cy: rows[i] + 1}
			drawBox(c, p, []string{it.noteText}, shapeRect)
		}
	}
	for _, x := range xs {
		c.junction(x, boxH-1, bD)
		c.segV(x, boxH, bottomTop-1)
		c.junction(x, bottomTop, bU)
	}

	for i, it := range seq.items {
		r := rows[i]
		switch it.kind {
		case seqMessage:
			lineCh := '─'
			if it.dashed {
				lineCh = '╌'
			}
			if it.from == it.to {
				x := xs[it.from]
				c.junction(x, r, bR)
				c.set(x+1, r, lineCh, clsEdge)
				c.set(x+2, r, lineCh, clsEdge)
				c.set(x+3, r, '╮', clsEdge)
				c.set(x+3, r+1, '│', clsEdge)
				headCh := '◄'
				if it.head == seqHeadCross {
					headCh = '×'
				}
				c.set(x+1, r+2, headCh, clsEdge)
				c.set(x+2, r+2, lineCh, clsEdge)
				c.set(x+3, r+2, '╯', clsEdge)
				if it.hasText {
					drawSeqText(c, it.text, x+5, r+1, clsText)
				}
			} else {
				x0, x1 := xs[it.from], xs[it.to]
				rightward := x1 > x0
				arrowRow := r
				if it.hasText {
					arrowRow = r + 1
				}
				lo, hi := minInt(x0, x1), maxInt(x0, x1)
				if rightward {
					c.junction(x0, arrowRow, bR)
				} else {
					c.junction(x0, arrowRow, bL)
				}
				for x := lo + 1; x < hi; x++ {
					c.set(x, arrowRow, lineCh, clsEdge)
				}
				var headCh rune
				switch {
				case it.head == seqHeadCross:
					headCh = '×'
				case rightward:
					headCh = '▶'
				default:
					headCh = '◄'
				}
				headX := x1 + 1
				if rightward {
					headX = x1 - 1
				}
				c.set(headX, arrowRow, headCh, clsEdge)
				if it.hasText {
					span := hi - lo - 1
					t := fitLabel(it.text, maxInt(span, 1))
					tx := lo + 1 + satSub(span, strWidth(t))/2
					drawSeqText(c, t, tx, r, clsText)
				}
			}
		case seqDivider:
			for x := 0; x < canvasW; x++ {
				c.set(x, r, '─', clsEdge)
			}
			t := fitLabel(it.noteText, satSub(canvasW, 4))
			drawSeqText(c, " "+t+" ", 2, r, clsEdgeLabel)
		}
	}

	c.finalizeMask()
	return c.toLines(), ovOK
}

func divCeil(a, b int) int { return (a + b - 1) / b }

type seqReq struct{ l, r, need int }

// stableSortReqs sorts requirement spans by (r-l) ascending, stably.
func stableSortReqs(reqs []seqReq) {
	// insertion sort keeps stability and matches Rust's sort_by_key.
	for i := 1; i < len(reqs); i++ {
		j := i
		for j > 0 && (reqs[j-1].r-reqs[j-1].l) > (reqs[j].r-reqs[j].l) {
			reqs[j-1], reqs[j] = reqs[j], reqs[j-1]
			j--
		}
	}
}
