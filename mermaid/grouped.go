package main

// item identifies either a plain node or a subgraph proxy within a scope.
type item struct {
	group bool
	id    int
}

type scopeEdge struct {
	f, t item
	ei   int
}

func renderGrouped(g *graph, maxWidth int) ([]span2D, int) {
	proxy := map[int]int{} // node index -> group index (for group-id nodes)
	for gi, gr := range g.groups {
		if ni, ok := g.index[gr.id]; ok {
			proxy[ni] = gi
		}
	}

	groupChain := func(gOpt int) []int {
		var chain []int
		cur := gOpt
		for cur >= 0 {
			chain = append(chain, cur)
			cur = g.groups[cur].parent
		}
		// reverse
		for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
			chain[i], chain[j] = chain[j], chain[i]
		}
		return chain
	}
	endpoint := func(n int) (item, []int) {
		if gi, ok := proxy[n]; ok {
			return item{group: true, id: gi}, groupChain(g.groups[gi].parent)
		}
		return item{group: false, id: n}, groupChain(g.nodeGroup[n])
	}

	scopeEdges := map[int][]scopeEdge{} // scope group index (-1 = top) -> edges
	referenced := make([]bool, len(g.groups))
	for ei, e := range g.edges {
		itemF, chainF := endpoint(e.from)
		itemT, chainT := endpoint(e.to)
		k := 0
		for k < len(chainF) && k < len(chainT) && chainF[k] == chainT[k] {
			k++
		}
		scope := -1
		if k != 0 {
			scope = chainF[k-1]
		}
		f := itemF
		if len(chainF) > k {
			f = item{group: true, id: chainF[k]}
		}
		t := itemT
		if len(chainT) > k {
			t = item{group: true, id: chainT[k]}
		}
		if f.group {
			referenced[f.id] = true
		}
		if t.group {
			referenced[t.id] = true
		}
		scopeEdges[scope] = append(scopeEdges[scope], scopeEdge{f, t, ei})
	}

	directNodes := map[int][]int{}
	for ni, gr := range g.nodeGroup {
		if _, ok := proxy[ni]; !ok {
			directNodes[gr] = append(directNodes[gr], ni)
		}
	}
	keep := make([]bool, len(g.groups))
	for gi := len(g.groups) - 1; gi >= 0; gi-- {
		hasNodes := len(directNodes[gi]) > 0
		hasChildren := false
		for c := 0; c < len(g.groups); c++ {
			if g.groups[c].parent == gi && keep[c] {
				hasChildren = true
				break
			}
		}
		keep[gi] = hasNodes || hasChildren || referenced[gi]
	}

	c, ov := buildScope(g, -1, scopeEdges, directNodes, keep, maxWidth)
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

func buildScope(g *graph, scope int, scopeEdges map[int][]scopeEdge, directNodes map[int][]int, keep []bool, maxWidth int) (*canvas, int) {
	var items []item
	for _, n := range directNodes[scope] {
		items = append(items, item{group: false, id: n})
	}
	var childGroups []int
	for gi := 0; gi < len(g.groups); gi++ {
		if g.groups[gi].parent == scope && keep[gi] {
			childGroups = append(childGroups, gi)
		}
	}
	for _, gi := range childGroups {
		items = append(items, item{group: true, id: gi})
	}
	if len(items) == 0 {
		return newCanvas(1, 1), ovOK
	}

	indexOf := map[item]int{}
	synth := newGraph(g.dir)
	var extras []nodeExtra
	for _, it := range items {
		indexOf[it] = len(synth.nodes)
		if it.group {
			sub, ov := buildScope(g, it.id, scopeEdges, directNodes, keep, 0)
			if ov != ovOK {
				return nil, ov
			}
			synth.nodes = append(synth.nodes, node{label: g.groups[it.id].label, shape: shapeRect})
			extras = append(extras, nodeExtra{kind: extraFrame, frame: sub})
		} else {
			synth.nodes = append(synth.nodes, node{label: g.nodes[it.id].label, shape: g.nodes[it.id].shape})
			extras = append(extras, nodeExtra{kind: extraPlain})
		}
	}

	for _, se := range scopeEdges[scope] {
		fi, okF := indexOf[se.f]
		ti, okT := indexOf[se.t]
		if !okF || !okT {
			continue
		}
		e := g.edges[se.ei]
		synth.edges = append(synth.edges, edge{
			from: fi, to: ti, label: e.label, hasLabel: e.hasLabel,
			headTo: e.headTo, headFrom: e.headFrom, line: e.line,
		})
	}

	return layoutCanvas(synth, extras, maxWidth)
}
