package main

import (
	"math"
	"sort"
)

// Oversize outcomes from a layout attempt.
const (
	ovOK = iota
	ovWidth
	ovCells
)

func satSub(a, b int) int {
	if a > b {
		return a - b
	}
	return 0
}

// Node-extra kinds decide how a box is drawn.
const (
	extraPlain = iota
	extraFrame
	extraCompart
)

type nodeExtra struct {
	kind    int
	frame   *canvas
	compart [][]string
}

type nodeSizes struct {
	boxW, boxH         []int
	layW, layH         []int
	extraH, selfLabelW []int
}

type routePlan struct {
	canvasW, canvasH int
	bandEnd          []int
	edgeBus          []int
	laneBase         int
	edgeLane         []int
}

type trackSpan struct{ s, e, f, t, idx int }

type assignment struct{ idx, slot int }

func layoutFlowchart(g *graph, maxWidth int) ([]span2D, int) {
	extras := make([]nodeExtra, len(g.nodes))
	for i := range extras {
		extras[i] = nodeExtra{kind: extraPlain}
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

func layoutCanvas(g *graph, extras []nodeExtra, maxWidth int) (*canvas, int) {
	n := len(g.nodes)
	if n == 0 {
		return nil, ovCells
	}

	ranks := computeRanks(g)
	maxRank := 0
	for _, r := range ranks {
		maxRank = maxInt(maxRank, r)
	}

	byRank := make([][]int, maxRank+1)
	for idx, r := range ranks {
		byRank[r] = append(byRank[r], idx)
	}
	orderRanks(byRank, g.edges, ranks)

	wrapped := make([][]string, n)
	for i, nd := range g.nodes {
		wrapped[i] = wrapLabel(nd.label, wrapWidth, maxLines)
	}

	boxW := make([]int, n)
	for i := 0; i < n; i++ {
		switch extras[i].kind {
		case extraFrame:
			titleW := strWidth(fitLabel(g.nodes[i].label, wrapWidth))
			boxW[i] = maxInt(extras[i].frame.w+2, titleW+4)
		case extraCompart:
			mw := 1
			for _, sec := range extras[i].compart {
				for _, l := range sec {
					mw = maxInt(mw, strWidth(l))
				}
			}
			boxW[i] = mw + 2*pad + 2
		default:
			mw := 1
			for _, l := range wrapped[i] {
				mw = maxInt(mw, strWidth(l))
			}
			boxW[i] = mw + 2*pad + 2
		}
	}
	boxH := make([]int, n)
	for i := 0; i < n; i++ {
		switch extras[i].kind {
		case extraFrame:
			boxH[i] = extras[i].frame.h + 2
		case extraCompart:
			filled := 0
			total := 0
			for _, sec := range extras[i].compart {
				if len(sec) > 0 {
					filled++
				}
				total += len(sec)
			}
			boxH[i] = total + satSub(filled, 1) + 2
		default:
			boxH[i] = len(wrapped[i]) + 2
		}
	}

	extraH := make([]int, n)
	selfLabelW := make([]int, n)
	for _, e := range g.edges {
		if e.from == e.to {
			extraH[e.from] = 2
			if e.hasLabel {
				selfLabelW[e.from] = maxInt(selfLabelW[e.from], minInt(strWidth(e.label), maxLabel))
			}
		}
	}
	for i := 0; i < n; i++ {
		if extraH[i] > 0 {
			boxW[i] = maxInt(boxW[i], 7)
		}
	}
	layW := make([]int, n)
	layH := make([]int, n)
	for i := 0; i < n; i++ {
		extra := 0
		if selfLabelW[i] > 0 {
			extra = 2 * (selfLabelW[i] + 3)
		}
		layW[i] = boxW[i] + extra
		layH[i] = boxH[i] + extraH[i]
	}
	sizes := nodeSizes{boxW: boxW, boxH: boxH, layW: layW, layH: layH, extraH: extraH, selfLabelW: selfLabelW}

	placedNodes := make([]placed, n)

	vertical := g.dir == dirDown || g.dir == dirUp
	var plan routePlan
	if vertical {
		plan = placeTD(ranks, maxRank, byRank, sizes, g, placedNodes)
	} else {
		plan = placeLR(ranks, maxRank, byRank, sizes, g, placedNodes)
	}
	canvasW, canvasH := plan.canvasW, plan.canvasH

	if maxWidth > 0 && canvasW > maxWidth {
		return nil, ovWidth
	}
	if canvasW*canvasH > maxCanvasCell {
		return nil, ovCells
	}

	c := newCanvas(canvasW, canvasH)
	for idx := 0; idx < n; idx++ {
		switch extras[idx].kind {
		case extraFrame:
			drawFrame(c, placedNodes[idx], g.nodes[idx].label, extras[idx].frame)
		case extraCompart:
			drawClassBox(c, placedNodes[idx], extras[idx].compart)
		default:
			drawBox(c, placedNodes[idx], wrapped[idx], g.nodes[idx].shape)
		}
	}
	for i, e := range g.edges {
		switch e.line {
		case lineDotted:
			c.curStyle = styDot
		case lineThick:
			c.curStyle = styThick
		default:
			c.curStyle = stySolid
		}
		if e.from == e.to {
			routeSelf(c, placedNodes[e.from], e)
			continue
		}
		from, to := placedNodes[e.from], placedNodes[e.to]
		adjacent := to.rank == from.rank+1
		bus := plan.bandEnd[from.rank] + plan.edgeBus[i]
		lane := plan.laneBase + plan.edgeLane[i]
		switch {
		case vertical && adjacent:
			routeForward(c, from, to, e, bus)
		case vertical && !adjacent:
			routeBack(c, from, to, e, lane)
		case !vertical && adjacent:
			routeForwardLR(c, from, to, e, bus)
		default:
			routeBackLR(c, from, to, e, lane)
		}
	}

	c.finalizeMask()
	return c, ovOK
}

func computeRanks(g *graph) []int {
	n := len(g.nodes)
	children := make([][]int, n)
	indeg := make([]int, n)
	for _, e := range g.edges {
		if e.from != e.to {
			children[e.from] = append(children[e.from], e.to)
			indeg[e.to]++
		}
	}
	color := make([]uint8, n)
	dag := make([][]int, n)
	order := make([]int, 0, n)

	var starts []int
	for i := 0; i < n; i++ {
		if indeg[i] == 0 {
			starts = append(starts, i)
		}
	}
	for i := 0; i < n; i++ {
		starts = append(starts, i)
	}
	for _, start := range starts {
		if color[start] == 0 {
			dfsDAG(start, children, color, dag, &order)
		}
	}

	rank := make([]int, n)
	for i := len(order) - 1; i >= 0; i-- {
		u := order[i]
		for _, v := range dag[u] {
			rank[v] = maxInt(rank[v], rank[u]+1)
		}
	}
	return rank
}

func dfsDAG(start int, children [][]int, color []uint8, dag [][]int, order *[]int) {
	type frame struct{ u, i int }
	stack := []frame{{start, 0}}
	color[start] = 1
	for len(stack) > 0 {
		f := &stack[len(stack)-1]
		u := f.u
		if f.i < len(children[u]) {
			v := children[u][f.i]
			f.i++
			if color[v] == 1 {
				continue
			}
			dag[u] = append(dag[u], v)
			if color[v] == 0 {
				color[v] = 1
				stack = append(stack, frame{v, 0})
			}
		} else {
			color[u] = 2
			*order = append(*order, u)
			stack = stack[:len(stack)-1]
		}
	}
}

func orderRanks(byRank [][]int, edges []edge, ranks []int) {
	n := len(ranks)
	if len(byRank) < 2 || n < 3 {
		return
	}
	parents := make([][]int, n)
	children := make([][]int, n)
	for _, e := range edges {
		if e.from != e.to && ranks[e.to] > ranks[e.from] {
			parents[e.to] = append(parents[e.to], e.from)
			children[e.from] = append(children[e.from], e.to)
		}
	}
	pos := make([]int, n)
	setPos := func() {
		for _, row := range byRank {
			for i, v := range row {
				pos[v] = i
			}
		}
	}
	setPos()

	best := cloneRows(byRank)
	bestCrossings := countCrossings(edges, ranks, pos)
	if bestCrossings == 0 {
		return
	}
	for it := 0; it < 8; it++ {
		if it%2 == 0 {
			for r := 1; r < len(byRank); r++ {
				sortByBarycenter(byRank[r], parents, pos)
				for i, v := range byRank[r] {
					pos[v] = i
				}
			}
		} else {
			for r := len(byRank) - 2; r >= 0; r-- {
				sortByBarycenter(byRank[r], children, pos)
				for i, v := range byRank[r] {
					pos[v] = i
				}
			}
		}
		crossings := countCrossings(edges, ranks, pos)
		if crossings < bestCrossings {
			bestCrossings = crossings
			best = cloneRows(byRank)
		}
		if bestCrossings == 0 {
			break
		}
	}
	for i := range byRank {
		byRank[i] = best[i]
	}
}

func cloneRows(rows [][]int) [][]int {
	out := make([][]int, len(rows))
	for i, r := range rows {
		out[i] = append([]int(nil), r...)
	}
	return out
}

func sortByBarycenter(row []int, neigh [][]int, pos []int) {
	type keyed struct {
		key float64
		v   int
	}
	ks := make([]keyed, len(row))
	for i, v := range row {
		var key float64
		if len(neigh[v]) == 0 {
			key = float64(pos[v])
		} else {
			sum := 0.0
			for _, u := range neigh[v] {
				sum += float64(pos[u])
			}
			key = sum / float64(len(neigh[v]))
		}
		ks[i] = keyed{key, v}
	}
	sort.SliceStable(ks, func(a, b int) bool { return ks[a].key < ks[b].key })
	for i := range row {
		row[i] = ks[i].v
	}
}

func countCrossings(edges []edge, ranks []int, pos []int) int {
	type adj struct{ r, a, b int }
	var adjacent []adj
	for _, e := range edges {
		if e.from != e.to && ranks[e.to] == ranks[e.from]+1 {
			adjacent = append(adjacent, adj{ranks[e.from], pos[e.from], pos[e.to]})
		}
	}
	crossings := 0
	for i := range adjacent {
		a := adjacent[i]
		for j := i + 1; j < len(adjacent); j++ {
			b := adjacent[j]
			if a.r == b.r && ((a.a < b.a && a.b > b.b) || (a.a > b.a && a.b < b.b)) {
				crossings++
			}
		}
	}
	return crossings
}

func assignPositions(byRank [][]int, size []int, sep int, edges []edge, ranks []int) []int {
	n := len(size)
	parents := make([][]int, n)
	children := make([][]int, n)
	for _, e := range edges {
		if e.from != e.to && ranks[e.to] > ranks[e.from] {
			parents[e.to] = append(parents[e.to], e.from)
			children[e.from] = append(children[e.from], e.to)
		}
	}
	pos := make([]float64, n)
	for _, row := range byRank {
		x := 0.0
		for _, v := range row {
			half := float64(size[v]) / 2.0
			x += half
			pos[v] = x
			x += half + float64(sep)
		}
	}
	for it := 0; it < 10; it++ {
		if it%2 == 0 {
			for _, row := range byRank {
				relaxRank(row, parents, pos, size, sep)
			}
		} else {
			for r := len(byRank) - 1; r >= 0; r-- {
				relaxRank(byRank[r], children, pos, size, sep)
			}
		}
	}
	minLeft := math.Inf(1)
	for v := 0; v < n; v++ {
		l := pos[v] - float64(size[v])/2.0
		if l < minLeft {
			minLeft = l
		}
	}
	if math.IsInf(minLeft, 0) {
		minLeft = 0
	}
	out := make([]int, n)
	for v := 0; v < n; v++ {
		out[v] = maxInt(int(math.Round(pos[v]-minLeft)), 0)
	}
	return out
}

func relaxRank(nodes []int, neigh [][]int, pos []float64, size []int, sep int) {
	n := len(nodes)
	if n == 0 {
		return
	}
	desired := make([]float64, n)
	for i, v := range nodes {
		if len(neigh[v]) == 0 {
			desired[i] = pos[v]
		} else {
			sum := 0.0
			for _, u := range neigh[v] {
				sum += pos[u]
			}
			desired[i] = sum / float64(len(neigh[v]))
		}
	}
	half := func(i int) float64 { return float64(size[nodes[i]]) / 2.0 }
	left := make([]float64, n)
	right := make([]float64, n)
	for i := 0; i < n; i++ {
		if i == 0 {
			left[i] = desired[i]
		} else {
			left[i] = math.Max(desired[i], left[i-1]+half(i-1)+float64(sep)+half(i))
		}
	}
	for i := n - 1; i >= 0; i-- {
		if i == n-1 {
			right[i] = desired[i]
		} else {
			right[i] = math.Min(desired[i], right[i+1]-half(i+1)-float64(sep)-half(i))
		}
	}
	for i := 0; i < n; i++ {
		pos[nodes[i]] = (left[i] + right[i]) / 2.0
	}
	for i := 1; i < n; i++ {
		minP := pos[nodes[i-1]] + half(i-1) + float64(sep) + half(i)
		if pos[nodes[i]] < minP {
			pos[nodes[i]] = minP
		}
	}
}

func assignTracks(spans []trackSpan) ([]assignment, int) {
	sorted := append([]trackSpan(nil), spans...)
	sort.Slice(sorted, func(a, b int) bool {
		x, y := sorted[a], sorted[b]
		if x.s != y.s {
			return x.s < y.s
		}
		if x.e != y.e {
			return x.e < y.e
		}
		if x.f != y.f {
			return x.f < y.f
		}
		if x.t != y.t {
			return x.t < y.t
		}
		return x.idx < y.idx
	})
	type member struct{ s, e, f, t int }
	var tracks [][]member
	out := make([]assignment, 0, len(sorted))
	for _, sp := range sorted {
		slot := -1
		for ti, members := range tracks {
			compatible := true
			for _, m := range members {
				if !(m.e+2 <= sp.s || sp.e+2 <= m.s || m.f == sp.f || m.t == sp.t) {
					compatible = false
					break
				}
			}
			if compatible {
				slot = ti
				break
			}
		}
		if slot < 0 {
			tracks = append(tracks, nil)
			slot = len(tracks) - 1
		}
		tracks[slot] = append(tracks[slot], member{sp.s, sp.e, sp.f, sp.t})
		out = append(out, assignment{sp.idx, slot})
	}
	return out, len(tracks)
}

func busSpansTD(g *graph, ranks, centers []int, r int, exact bool) []trackSpan {
	var out []trackSpan
	for i, e := range g.edges {
		if e.from == e.to || ranks[e.from] != r || ranks[e.to] != r+1 {
			continue
		}
		var jogs bool
		if exact {
			jogs = centers[e.from] != centers[e.to]
		} else {
			jogs = absDiff(centers[e.from], centers[e.to]) > 1
		}
		if !jogs {
			continue
		}
		a := minInt(centers[e.from], centers[e.to])
		b := maxInt(centers[e.from], centers[e.to])
		out = append(out, trackSpan{a, b, e.from, e.to, i})
	}
	return out
}

func laneSpans(g *graph, ranks []int, placedNodes []placed, vertical bool) []trackSpan {
	var out []trackSpan
	for i, e := range g.edges {
		if e.from == e.to || ranks[e.to] == ranks[e.from]+1 {
			continue
		}
		pf, pt := placedNodes[e.from], placedNodes[e.to]
		var a, b int
		if vertical {
			a, b = minInt(pf.cy, pt.cy), maxInt(pf.cy, pt.cy)
		} else {
			a, b = minInt(pf.cx, pt.cx), maxInt(pf.cx, pt.cx)
		}
		out = append(out, trackSpan{a, b, e.from, e.to, i})
	}
	return out
}

func placeTD(ranks []int, maxRank int, byRank [][]int, sizes nodeSizes, g *graph, placedNodes []placed) routePlan {
	centers := assignPositions(byRank, sizes.layW, gapX, g.edges, ranks)

	edgeBus := make([]int, len(g.edges))
	busTracks := make([]int, maxRank+1)
	for r := 0; r < maxRank; r++ {
		spans := busSpansTD(g, ranks, centers, r, false)
		if len(spans) == 0 {
			continue
		}
		assigned, count := assignTracks(spans)
		for _, a := range assigned {
			edgeBus[a.idx] = a.slot
		}
		busTracks[r] = count
	}

	rankH := make([]int, len(byRank))
	for r, row := range byRank {
		mh := 0
		for _, i := range row {
			mh = maxInt(mh, sizes.boxH[i]+sizes.extraH[i])
		}
		if mh == 0 {
			mh = 3
		}
		rankH[r] = mh
	}
	rankY := make([]int, maxRank+1)
	for r := 1; r <= maxRank; r++ {
		gap := maxInt(gapY, busTracks[r-1]+1)
		rankY[r] = rankY[r-1] + rankH[r-1] + gap
	}
	canvasH := rankY[maxRank] + rankH[maxRank]
	bandEnd := make([]int, maxRank+1)
	for r := 0; r <= maxRank; r++ {
		bandEnd[r] = rankY[r] + rankH[r]
	}

	diagramW := 1
	for r, row := range byRank {
		for _, idx := range row {
			w := sizes.boxW[idx]
			h := sizes.boxH[idx]
			cx := centers[idx]
			x := satSub(cx, w/2)
			y := rankY[r] + (rankH[r]-h-sizes.extraH[idx])/2
			placedNodes[idx] = placed{x: x, y: y, w: w, h: h, cx: cx, cy: y + h/2, rank: r}
			diagramW = maxInt(diagramW, x+w)
			if sizes.extraH[idx] > 0 && sizes.selfLabelW[idx] > 0 {
				diagramW = maxInt(diagramW, x+w+2+sizes.selfLabelW[idx])
			}
		}
	}

	contentW := diagramW
	for _, e := range g.edges {
		if e.from == e.to || !e.hasLabel {
			continue
		}
		lw := minInt(strWidth(e.label), maxLabel)
		if ranks[e.to] == ranks[e.from]+1 {
			contentW = maxInt(contentW, placedNodes[e.to].cx+2+lw)
		} else {
			contentW = maxInt(contentW, diagramW+lw+1)
		}
	}

	edgeLane := make([]int, len(g.edges))
	lanes := laneSpans(g, ranks, placedNodes, true)
	canvasW := contentW
	laneBase := 0
	if len(lanes) > 0 {
		assigned, count := assignTracks(lanes)
		for _, a := range assigned {
			edgeLane[a.idx] = a.slot
		}
		canvasW = contentW + 1 + count
		laneBase = contentW + 1
	}

	return routePlan{canvasW: canvasW, canvasH: canvasH, bandEnd: bandEnd, edgeBus: edgeBus, laneBase: laneBase, edgeLane: edgeLane}
}

func placeLR(ranks []int, maxRank int, byRank [][]int, sizes nodeSizes, g *graph, placedNodes []placed) routePlan {
	colW := make([]int, len(byRank))
	for r, row := range byRank {
		mw := 0
		for _, i := range row {
			mw = maxInt(mw, sizes.boxW[i])
		}
		colW[r] = mw
	}

	maxLabelW := 0
	for _, e := range g.edges {
		if !(e.from == e.to || ranks[e.to] == ranks[e.from]+1) {
			continue
		}
		if e.hasLabel {
			maxLabelW = maxInt(maxLabelW, minInt(strWidth(e.label), maxLabel))
		}
	}
	baseGap := maxInt(gapX+1, maxLabelW+3)

	centers := assignPositions(byRank, sizes.layH, 1, g.edges, ranks)

	edgeBus := make([]int, len(g.edges))
	busTracks := make([]int, maxRank+1)
	for r := 0; r < maxRank; r++ {
		spans := busSpansTD(g, ranks, centers, r, true)
		if len(spans) == 0 {
			continue
		}
		assigned, count := assignTracks(spans)
		for _, a := range assigned {
			edgeBus[a.idx] = a.slot
		}
		busTracks[r] = count
	}

	rankX := make([]int, maxRank+1)
	for r := 1; r <= maxRank; r++ {
		gap := maxInt(baseGap, busTracks[r-1]+1)
		rankX[r] = rankX[r-1] + colW[r-1] + gap
	}
	tail := 0
	for _, i := range byRank[maxRank] {
		if sizes.extraH[i] > 0 && sizes.selfLabelW[i] > 0 {
			tail = maxInt(tail, 2+sizes.selfLabelW[i])
		}
	}
	canvasW := rankX[maxRank] + colW[maxRank] + tail
	bandEnd := make([]int, maxRank+1)
	for r := 0; r <= maxRank; r++ {
		bandEnd[r] = rankX[r] + colW[r]
	}

	diagramH := 1
	for r, row := range byRank {
		x := rankX[r]
		for _, idx := range row {
			w := sizes.boxW[idx]
			h := sizes.boxH[idx]
			cy := centers[idx]
			y := satSub(cy, (h+sizes.extraH[idx])/2)
			placedNodes[idx] = placed{x: x, y: y, w: w, h: h, cx: x + w/2, cy: y + h/2, rank: r}
			diagramH = maxInt(diagramH, y+h+sizes.extraH[idx])
		}
	}

	edgeLane := make([]int, len(g.edges))
	lanes := laneSpans(g, ranks, placedNodes, false)
	canvasH := diagramH
	laneBase := 0
	if len(lanes) > 0 {
		assigned, count := assignTracks(lanes)
		for _, a := range assigned {
			edgeLane[a.idx] = a.slot
		}
		canvasH = diagramH + 1 + count
		laneBase = diagramH + 1
	}

	return routePlan{canvasW: canvasW, canvasH: canvasH, bandEnd: bandEnd, edgeBus: edgeBus, laneBase: laneBase, edgeLane: edgeLane}
}
