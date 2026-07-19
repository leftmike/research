package main

import "strings"

// span is a run of same-styled text on a rendered line.
type span struct {
	text string
	cls  int
}

// canvas is a character grid with per-cell edge-bit masks that are resolved into
// box-drawing glyphs by finalizeMask.
type canvas struct {
	w, h     int
	ch       []rune
	cls      []int
	mask     []uint8
	style    []uint8
	occupied []bool
	curStyle uint8
}

func newCanvas(w, h int) *canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	n := w * h
	c := &canvas{
		w: w, h: h,
		ch:       make([]rune, n),
		cls:      make([]int, n),
		mask:     make([]uint8, n),
		style:    make([]uint8, n),
		occupied: make([]bool, n),
		curStyle: stySolid,
	}
	for i := range c.ch {
		c.ch[i] = ' '
	}
	return c
}

func (c *canvas) idx(x, y int) int { return y*c.w + x }

func (c *canvas) set(x, y int, r rune, cls int) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	i := c.idx(x, y)
	c.ch[i] = r
	c.cls[i] = cls
}

func (c *canvas) addBits(x, y int, bits uint8) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	i := c.idx(x, y)
	if c.occupied[i] {
		return
	}
	c.mask[i] |= bits
	c.style[i] |= c.curStyle
	if c.cls[i] != clsBorder {
		c.cls[i] = clsEdge
	}
}

func (c *canvas) blit(sub *canvas, ox, oy int) {
	for sy := 0; sy < sub.h; sy++ {
		for sx := 0; sx < sub.w; sx++ {
			x, y := ox+sx, oy+sy
			if x < 0 || y < 0 || x >= c.w || y >= c.h {
				continue
			}
			si := sub.idx(sx, sy)
			di := c.idx(x, y)
			c.ch[di] = sub.ch[si]
			c.cls[di] = sub.cls[si]
			c.style[di] = sub.style[si]
			c.occupied[di] = true
		}
	}
}

func (c *canvas) junction(x, y int, bits uint8) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	i := c.idx(x, y)
	c.mask[i] |= bits
	if c.cls[i] != clsBorder {
		c.cls[i] = clsEdge
	}
}

func (c *canvas) segV(x, y0, y1 int) {
	a, b := minInt(y0, y1), maxInt(y0, y1)
	for y := a; y <= b; y++ {
		var bits uint8
		if y > a {
			bits |= bU
		}
		if y < b {
			bits |= bD
		}
		c.addBits(x, y, bits)
	}
}

func (c *canvas) segH(y, x0, x1 int) {
	a, b := minInt(x0, x1), maxInt(x0, x1)
	for x := a; x <= b; x++ {
		var bits uint8
		if x > a {
			bits |= bL
		}
		if x < b {
			bits |= bR
		}
		c.addBits(x, y, bits)
	}
}

func (c *canvas) finalizeMask() {
	for i := range c.ch {
		if c.mask[i] != 0 && c.ch[i] == ' ' {
			r := maskChar(c.mask[i])
			switch c.style[i] {
			case styDot:
				r = dottedChar(r)
			case styThick:
				r = thickChar(r)
			}
			c.ch[i] = r
		}
	}
}

// flipVertical mirrors top-to-bottom for BT; box-drawing glyphs flip too.
func (c *canvas) flipVertical() {
	for y := 0; y < c.h/2; y++ {
		y2 := c.h - 1 - y
		for x := 0; x < c.w; x++ {
			i, j := c.idx(x, y), c.idx(x, y2)
			c.ch[i], c.ch[j] = c.ch[j], c.ch[i]
			c.cls[i], c.cls[j] = c.cls[j], c.cls[i]
		}
	}
	for i := range c.ch {
		c.ch[i] = flipGlyphV(c.ch[i])
	}
}

// flipHorizontal mirrors left-to-right for RL, then reverses each text/label run
// back to reading order.
func (c *canvas) flipHorizontal() {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w/2; x++ {
			x2 := c.w - 1 - x
			i, j := c.idx(x, y), c.idx(x2, y)
			c.ch[i], c.ch[j] = c.ch[j], c.ch[i]
			c.cls[i], c.cls[j] = c.cls[j], c.cls[i]
		}
	}
	for i := range c.ch {
		c.ch[i] = flipGlyphH(c.ch[i])
	}
	for y := 0; y < c.h; y++ {
		x := 0
		for x < c.w {
			cls := c.cls[c.idx(x, y)]
			if cls == clsText || cls == clsEdgeLabel {
				start := x
				for x < c.w && c.cls[c.idx(x, y)] == cls {
					x++
				}
				// reverse ch[start..x] on this row
				lo, hi := c.idx(start, y), c.idx(x-1, y)
				for lo < hi {
					c.ch[lo], c.ch[hi] = c.ch[hi], c.ch[lo]
					lo++
					hi--
				}
			} else {
				x++
			}
		}
	}
}

// toLines resolves the canvas into styled lines (runs of same-class text).
func (c *canvas) toLines() []span2D {
	out := make([]span2D, 0, c.h)
	for y := 0; y < c.h; y++ {
		last := c.w
		for x := c.w - 1; x >= 0; x-- {
			r := c.ch[c.idx(x, y)]
			if r != ' ' && r != cont {
				last = x + 1
				break
			}
			if x == 0 {
				last = 0
			}
		}
		var spans []span
		var run strings.Builder
		runCls := clsEmpty
		for x := 0; x < last; x++ {
			i := c.idx(x, y)
			r := c.ch[i]
			if r == cont {
				continue
			}
			cls := c.cls[i]
			if cls != runCls && run.Len() > 0 {
				spans = append(spans, span{run.String(), runCls})
				run.Reset()
			}
			runCls = cls
			run.WriteRune(r)
		}
		if run.Len() > 0 {
			spans = append(spans, span{run.String(), runCls})
		}
		out = append(out, spans)
	}
	return out
}

type span2D = []span

func maskChar(mask uint8) rune {
	switch mask {
	case 0:
		return ' '
	case bU, bD, bU | bD:
		return '│'
	case bL, bR, bL | bR:
		return '─'
	case bD | bR:
		return '┌'
	case bD | bL:
		return '┐'
	case bU | bR:
		return '└'
	case bU | bL:
		return '┘'
	case bU | bD | bR:
		return '├'
	case bU | bD | bL:
		return '┤'
	case bD | bL | bR:
		return '┬'
	case bU | bL | bR:
		return '┴'
	default:
		return '┼'
	}
}

func dottedChar(r rune) rune {
	switch r {
	case '─':
		return '╌'
	case '│':
		return '╎'
	}
	return r
}

func thickChar(r rune) rune {
	switch r {
	case '─':
		return '━'
	case '│':
		return '┃'
	case '┌':
		return '┏'
	case '┐':
		return '┓'
	case '└':
		return '┗'
	case '┘':
		return '┛'
	case '├':
		return '┣'
	case '┤':
		return '┫'
	case '┬':
		return '┳'
	case '┴':
		return '┻'
	case '┼':
		return '╋'
	}
	return r
}

func flipGlyphV(r rune) rune {
	switch r {
	case '┌':
		return '└'
	case '└':
		return '┌'
	case '┐':
		return '┘'
	case '┘':
		return '┐'
	case '┏':
		return '┗'
	case '┗':
		return '┏'
	case '┓':
		return '┛'
	case '┛':
		return '┓'
	case '╭':
		return '╰'
	case '╰':
		return '╭'
	case '╮':
		return '╯'
	case '╯':
		return '╮'
	case '┬':
		return '┴'
	case '┴':
		return '┬'
	case '┳':
		return '┻'
	case '┻':
		return '┳'
	case '▼':
		return '▲'
	case '▲':
		return '▼'
	case '▽':
		return '△'
	case '△':
		return '▽'
	}
	return r
}

func flipGlyphH(r rune) rune {
	switch r {
	case '┌':
		return '┐'
	case '┐':
		return '┌'
	case '└':
		return '┘'
	case '┘':
		return '└'
	case '┏':
		return '┓'
	case '┓':
		return '┏'
	case '┗':
		return '┛'
	case '┛':
		return '┗'
	case '╭':
		return '╮'
	case '╮':
		return '╭'
	case '╰':
		return '╯'
	case '╯':
		return '╰'
	case '├':
		return '┤'
	case '┤':
		return '├'
	case '┣':
		return '┫'
	case '┫':
		return '┣'
	case '▶':
		return '◄'
	case '◄':
		return '▶'
	case '▷':
		return '◁'
	case '◁':
		return '▷'
	}
	return r
}

// placed is a laid-out box: top-left (x,y), size (w,h), center (cx,cy), rank.
type placed struct {
	x, y, w, h, cx, cy, rank int
}

func headGlyph(head int, arrow rune) rune {
	switch head {
	case headCircle:
		return 'o'
	case headCross:
		return '×'
	case headDiamondFill:
		return '◆'
	case headDiamondOpen:
		return '◇'
	case headTriangle:
		switch arrow {
		case '▼':
			return '▽'
		case '▲':
			return '△'
		case '◄':
			return '◁'
		case '▶':
			return '▷'
		}
		return arrow
	}
	return arrow
}

func drawBox(c *canvas, p placed, lines []string, shape int) {
	x, y, w, h := p.x, p.y, p.w, p.h
	right := x + w - 1
	bottom := y + h - 1

	var tl, tr, bl, br rune
	switch shape {
	case shapeRound, shapeDiamond:
		tl, tr, bl, br = '╭', '╮', '╰', '╯'
	default:
		tl, tr, bl, br = '┌', '┐', '└', '┘'
	}
	c.set(x, y, tl, clsBorder)
	c.set(right, y, tr, clsBorder)
	c.set(x, bottom, bl, clsBorder)
	c.set(right, bottom, br, clsBorder)

	for cx := x + 1; cx < right; cx++ {
		c.addBits(cx, y, bL|bR)
		c.addBits(cx, bottom, bL|bR)
	}
	for cy := y + 1; cy < bottom; cy++ {
		c.addBits(x, cy, bU|bD)
		c.addBits(right, cy, bU|bD)
	}

	for cy := y; cy <= bottom; cy++ {
		for cx := x; cx <= right; cx++ {
			if cx >= 0 && cy >= 0 && cx < c.w && cy < c.h {
				c.occupied[c.idx(cx, cy)] = true
			}
		}
	}

	inner := maxInt(w-(2*pad+2), 1)
	for li, line := range lines {
		row := y + 1 + li
		text := fitLabel(line, inner)
		tw := strWidth(text)
		textX := x + 1 + pad + (inner-tw)/2
		if inner-tw < 0 {
			textX = x + 1 + pad
		}
		cur := textX
		for _, r := range text {
			cw := maxInt(charWidth(r), 1)
			c.set(cur, row, r, clsText)
			for k := 1; k < cw; k++ {
				c.set(cur+k, row, cont, clsText)
			}
			cur += cw
		}
	}
}

// drawClassBox draws a compartmented box (class/ER) with divider rules.
func drawClassBox(c *canvas, p placed, sections [][]string) {
	drawBox(c, p, nil, shapeRect)
	inner := maxInt(p.w-(2*pad+2), 1)
	row := p.y + 1
	first := true
	for si, section := range sections {
		if len(section) == 0 {
			continue
		}
		if !first {
			c.set(p.x, row, '├', clsBorder)
			for x := p.x + 1; x < p.x+p.w-1; x++ {
				c.set(x, row, '─', clsBorder)
			}
			c.set(p.x+p.w-1, row, '┤', clsBorder)
			row++
		}
		first = false
		for _, line := range section {
			text := fitLabel(line, inner)
			var tx int
			if si == 0 {
				tx = p.x + 1 + pad + maxInt(inner-strWidth(text), 0)/2
			} else {
				tx = p.x + 1 + pad
			}
			drawSeqText(c, text, tx, row, clsText)
			row++
		}
	}
}

// drawFrame draws a titled frame around a pre-rendered sub-canvas (subgraph).
func drawFrame(c *canvas, p placed, title string, sub *canvas) {
	drawBox(c, p, nil, shapeRect)
	t := fitLabel(title, maxInt(p.w-4, 0))
	drawSeqText(c, " "+t+" ", p.x+1, p.y, clsText)
	ox := p.x + 1 + (p.w-2-sub.w)/2
	oy := p.y + 1 + (p.h-2-sub.h)/2
	c.blit(sub, ox, oy)
}

// drawSeqText writes text at (x,y), clearing any edge mask underneath so glyphs
// do not bleed through, and marking continuation cells for wide glyphs.
func drawSeqText(c *canvas, text string, x, y int, cls int) {
	cur := x
	for _, r := range text {
		cw := maxInt(charWidth(r), 1)
		for k := 0; k < cw; k++ {
			if cur+k >= 0 && cur+k < c.w && y >= 0 && y < c.h {
				c.mask[c.idx(cur+k, y)] = 0
			}
			glyph := r
			if k != 0 {
				glyph = cont
			}
			c.set(cur+k, y, glyph, cls)
		}
		cur += cw
	}
}

// placeLabel writes an edge label starting at start_x on row, stopping at the
// first occupied/drawn cell.
func placeLabel(c *canvas, label string, row, startX int) {
	if row < 0 || row >= c.h {
		return
	}
	text := fitLabel(label, maxLabel)
	x := startX
	for _, r := range text {
		cw := maxInt(charWidth(r), 1)
		if x+cw > c.w {
			break
		}
		blocked := false
		for k := 0; k < cw; k++ {
			if x+k < 0 {
				blocked = true
				break
			}
			i := c.idx(x+k, row)
			if c.ch[i] != ' ' || c.mask[i] != 0 || c.occupied[i] {
				blocked = true
				break
			}
		}
		if blocked {
			break
		}
		c.set(x, row, r, clsEdgeLabel)
		for k := 1; k < cw; k++ {
			c.set(x+k, row, cont, clsEdgeLabel)
		}
		x += cw
	}
}
