package main

func routeForward(c *canvas, from, to placed, e edge, bus int) {
	tx := to.cx
	bx := from.cx
	if absDiff(from.cx, tx) <= 1 {
		bx = tx
	}
	by := from.y + from.h - 1
	headRow := to.y - 1

	c.junction(bx, by, bD)
	c.segV(bx, by, bus)
	if bx == tx {
		c.segV(bx, bus, headRow)
	} else {
		c.segH(bus, bx, tx)
		c.segV(tx, bus, headRow)
	}

	if e.headTo == headNone {
		c.addBits(tx, headRow, bU)
	} else {
		c.set(tx, headRow, headGlyph(e.headTo, '▼'), clsEdge)
	}
	if e.headFrom != headNone {
		c.set(bx, by, headGlyph(e.headFrom, '▲'), clsEdge)
	}
	if e.hasLabel {
		placeLabel(c, e.label, headRow, tx+1)
	}
}

func routeSelf(c *canvas, p placed, e edge) {
	bottom := p.y + p.h - 1
	exitX := p.cx + 1
	retX := p.x + p.w - 2
	if retX <= exitX || bottom+2 >= c.h {
		return
	}
	var v, h, bl, br rune
	switch e.line {
	case lineDotted:
		v, h, bl, br = '╎', '╌', '╰', '╯'
	case lineThick:
		v, h, bl, br = '┃', '━', '┗', '┛'
	default:
		v, h, bl, br = '│', '─', '╰', '╯'
	}
	c.junction(exitX, bottom, bD)
	c.set(exitX, bottom+1, v, clsEdge)
	c.set(exitX, bottom+2, bl, clsEdge)
	for x := exitX + 1; x < retX; x++ {
		c.set(x, bottom+2, h, clsEdge)
	}
	c.set(retX, bottom+2, br, clsEdge)
	c.set(retX, bottom+1, headGlyph(e.headTo, '▲'), clsEdge)
	if e.hasLabel {
		placeLabel(c, e.label, bottom+1, p.x+p.w+1)
	}
}

func routeBack(c *canvas, from, to placed, e edge, laneX int) {
	sx := from.x + from.w - 1
	sy := from.cy
	tx := to.x + to.w - 1
	tyc := to.cy

	c.junction(sx, sy, bR)
	c.segH(sy, sx, laneX)
	c.segV(laneX, sy, tyc)
	c.segH(tyc, tx+1, laneX)

	if e.headTo == headNone {
		c.addBits(tx+1, tyc, bR)
	} else {
		c.set(tx+1, tyc, headGlyph(e.headTo, '◄'), clsEdge)
	}
	if e.headFrom != headNone {
		c.set(sx, sy, headGlyph(e.headFrom, '◄'), clsEdge)
	}
	if e.hasLabel {
		placeLabel(c, e.label, satSub(tyc, 1), satSub(laneX, strWidth(e.label)+1))
	}
}

func routeForwardLR(c *canvas, from, to placed, e edge, bus int) {
	rx := from.x + from.w - 1
	ry := from.cy
	ly := to.cy
	headCol := to.x - 1

	c.junction(rx, ry, bR)
	c.segH(ry, rx, bus)
	if ry == ly {
		c.segH(ry, bus, headCol)
	} else {
		c.segV(bus, ry, ly)
		c.segH(ly, bus, headCol)
	}

	if e.headTo == headNone {
		c.addBits(headCol, ly, bR)
	} else {
		c.set(headCol, ly, headGlyph(e.headTo, '▶'), clsEdge)
	}
	if e.headFrom != headNone {
		c.set(rx, ry, headGlyph(e.headFrom, '◄'), clsEdge)
	}
	if e.hasLabel {
		placeLabel(c, e.label, satSub(ly, 1), bus+1)
	}
}

func routeBackLR(c *canvas, from, to placed, e edge, laneY int) {
	sx := from.cx
	sy := from.y + from.h - 1
	tx := to.cx
	ty := to.y + to.h - 1

	c.junction(sx, sy, bD)
	c.segV(sx, sy, laneY)
	c.segH(laneY, sx, tx)
	c.segV(tx, laneY, ty+1)

	if e.headTo == headNone {
		c.addBits(tx, ty+1, bD)
	} else {
		c.set(tx, ty+1, headGlyph(e.headTo, '▲'), clsEdge)
	}
	if e.headFrom != headNone {
		c.set(sx, sy, headGlyph(e.headFrom, '▲'), clsEdge)
	}
	if e.hasLabel {
		placeLabel(c, e.label, satSub(laneY, 1), (sx+tx)/2)
	}
}
