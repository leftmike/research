package graph

import (
	"fmt"
	"sort"

	"github.com/leftmike/research/cypher"
)

// Query parses src and runs the resulting query against the graph.
func (g *Graph) Query(src string) (*Result, error) {
	q, err := cypher.Parse(src)
	if err != nil {
		return nil, err
	}
	return g.Run(q)
}

// Run executes a parsed query against the graph with no parameters.
func (g *Graph) Run(q *cypher.Query) (*Result, error) {
	return g.RunWithParams(q, nil)
}

// RunWithParams executes a parsed query, resolving $params from the given map.
//
// The supported shape is zero or more MATCH clauses followed by a single
// terminal RETURN. WITH pipelines, UNWIND, OPTIONAL MATCH and variable-length
// paths are not yet supported and produce an error.
func (g *Graph) RunWithParams(q *cypher.Query, params map[string]Value) (*Result, error) {
	ex := &executor{graph: g, ev: &evaluator{params: params}}

	scopes := []scope{{}}
	for i, clause := range q.Clauses {
		switch c := clause.(type) {
		case *cypher.Match:
			if c.Optional {
				return nil, fmt.Errorf("graph: OPTIONAL MATCH is not supported")
			}
			next, err := ex.runMatch(c, scopes)
			if err != nil {
				return nil, err
			}
			scopes = next
		case *cypher.Return:
			if i != len(q.Clauses)-1 {
				return nil, fmt.Errorf("graph: RETURN must be the final clause")
			}
			return ex.project(&c.Projection, scopes)
		case *cypher.With:
			return nil, fmt.Errorf("graph: WITH is not supported")
		case *cypher.Unwind:
			return nil, fmt.Errorf("graph: UNWIND is not supported")
		default:
			return nil, fmt.Errorf("graph: unsupported clause %T", clause)
		}
	}
	return nil, fmt.Errorf("graph: query has no RETURN clause")
}

type executor struct {
	graph *Graph
	ev    *evaluator
}

// runMatch expands each input scope with every match of the clause's pattern,
// then applies the optional WHERE predicate.
func (ex *executor) runMatch(m *cypher.Match, in []scope) ([]scope, error) {
	var out []scope
	for _, s := range in {
		matched, err := ex.matchParts(m.Pattern, s)
		if err != nil {
			return nil, err
		}
		out = append(out, matched...)
	}
	if m.Where == nil {
		return out, nil
	}
	var filtered []scope
	for _, s := range out {
		v, err := ex.ev.eval(m.Where, s)
		if err != nil {
			return nil, err
		}
		if truthy(v) {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

// matchParts matches every comma-separated pattern part against the input
// scope, joining on shared variables.
func (ex *executor) matchParts(parts []cypher.PatternPart, in scope) ([]scope, error) {
	cur := []scope{in}
	for _, part := range parts {
		if part.Variable != "" {
			return nil, fmt.Errorf("graph: named path variables are not supported")
		}
		var next []scope
		for _, s := range cur {
			matched, err := ex.matchElement(part.Element, s)
			if err != nil {
				return nil, err
			}
			next = append(next, matched...)
		}
		cur = next
	}
	return cur, nil
}

// matchElement matches a single node/relationship chain against a scope.
func (ex *executor) matchElement(el cypher.PatternElement, in scope) ([]scope, error) {
	var out []scope
	firsts, err := ex.candidateNodes(el.Nodes[0], in)
	if err != nil {
		return nil, err
	}
	for _, cand := range firsts {
		if err := ex.expandRels(el, 0, cand.node, cand.scope, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// candidate pairs a partial scope with the node bound at the current position.
type candidate struct {
	scope scope
	node  *Node
}

// candidateNodes returns the nodes (and extended scopes) that satisfy a node
// pattern given the bindings already present in s.
func (ex *executor) candidateNodes(np cypher.NodePattern, s scope) ([]candidate, error) {
	// If the variable is already bound, it must be a matching node.
	if np.Variable != "" {
		if bound, ok := s[np.Variable]; ok {
			n, ok := bound.(*Node)
			if !ok {
				return nil, nil
			}
			matches, err := ex.nodeMatches(np, n, s)
			if err != nil || !matches {
				return nil, err
			}
			return []candidate{{scope: s, node: n}}, nil
		}
	}

	var out []candidate
	for _, n := range ex.graph.nodes {
		matches, err := ex.nodeMatches(np, n, s)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}
		ns := cloneScope(s)
		if np.Variable != "" {
			ns[np.Variable] = n
		}
		out = append(out, candidate{scope: ns, node: n})
	}
	return out, nil
}

// expandRels recursively walks the relationship chain starting at rel index i,
// with cur bound to the node at position i, appending complete matches to out.
func (ex *executor) expandRels(el cypher.PatternElement, i int, cur *Node, s scope, out *[]scope) error {
	if i == len(el.Rels) {
		*out = append(*out, s)
		return nil
	}
	rel := el.Rels[i]
	if rel.VarLength {
		return fmt.Errorf("graph: variable-length relationships are not supported")
	}
	nextNP := el.Nodes[i+1]
	dir := traversalDir(rel.Direction)

	for _, st := range ex.graph.step(cur, dir, rel.Types) {
		// Relationship variable consistency and property match.
		if rel.Variable != "" {
			if bound, ok := s[rel.Variable]; ok {
				if br, ok := bound.(*Relationship); !ok || br != st.rel {
					continue
				}
			}
		}
		relOK, err := ex.propsMatch(st.rel.Props, rel.Properties, s)
		if err != nil {
			return err
		}
		if !relOK {
			continue
		}
		// Target node must satisfy its pattern.
		nodeOK, err := ex.nodeMatches(nextNP, st.node, s)
		if err != nil {
			return err
		}
		if !nodeOK {
			continue
		}
		if nextNP.Variable != "" {
			if bound, ok := s[nextNP.Variable]; ok {
				if bn, ok := bound.(*Node); !ok || bn != st.node {
					continue
				}
			}
		}

		ns := cloneScope(s)
		if rel.Variable != "" {
			ns[rel.Variable] = st.rel
		}
		if nextNP.Variable != "" {
			ns[nextNP.Variable] = st.node
		}
		if err := ex.expandRels(el, i+1, st.node, ns, out); err != nil {
			return err
		}
	}
	return nil
}

// nodeMatches reports whether node satisfies the labels and inline properties
// of the pattern. Variable binding consistency is checked by the caller.
func (ex *executor) nodeMatches(np cypher.NodePattern, node *Node, s scope) (bool, error) {
	if !node.hasLabels(np.Labels) {
		return false, nil
	}
	return ex.propsMatch(node.Props, np.Properties, s)
}

// propsMatch reports whether the entity's properties contain every key/value in
// the pattern's inline property map.
func (ex *executor) propsMatch(entity map[string]Value, propExpr cypher.Expr, s scope) (bool, error) {
	if propExpr == nil {
		return true, nil
	}
	want, err := ex.ev.eval(propExpr, s)
	if err != nil {
		return false, err
	}
	wm, ok := want.(map[string]Value)
	if !ok {
		return false, fmt.Errorf("graph: pattern properties must be a map")
	}
	for k, v := range wm {
		ev, ok := entity[k]
		if !ok || !equal(ev, v) {
			return false, nil
		}
	}
	return true, nil
}

func traversalDir(d cypher.RelDirection) relDir {
	switch d {
	case cypher.RelRight:
		return dirRight
	case cypher.RelLeft:
		return dirLeft
	default:
		return dirBoth
	}
}

func cloneScope(s scope) scope {
	ns := make(scope, len(s)+1)
	for k, v := range s {
		ns[k] = v
	}
	return ns
}

// projRow is a projected output row paired with the scope it came from (used
// for ORDER BY, which may reference bound variables).
type projRow struct {
	vals  []Value
	scope scope
}

// project builds the final Result from the matched scopes, applying the
// projection's items, DISTINCT, ORDER BY, SKIP and LIMIT.
func (ex *executor) project(p *cypher.Projection, scopes []scope) (*Result, error) {
	columns, items, err := ex.projectionColumns(p, scopes)
	if err != nil {
		return nil, err
	}

	rows := make([]projRow, 0, len(scopes))
	for _, s := range scopes {
		vals := make([]Value, len(items))
		for i, it := range items {
			v, err := ex.ev.eval(it.Expr, s)
			if err != nil {
				return nil, err
			}
			vals[i] = v
		}
		rows = append(rows, projRow{vals: vals, scope: s})
	}

	if p.Distinct {
		rows = distinctRows(rows)
	}

	if len(p.OrderBy) > 0 {
		if err := ex.sortRows(rows, p.OrderBy); err != nil {
			return nil, err
		}
	}

	rows, err = ex.applySkipLimit(rows, p)
	if err != nil {
		return nil, err
	}

	res := &Result{Columns: columns}
	for _, r := range rows {
		res.Rows = append(res.Rows, r.vals)
	}
	return res, nil
}

// projectionColumns resolves the column names and projection items, expanding a
// RETURN */WITH * over the variables bound in the matched scopes.
func (ex *executor) projectionColumns(p *cypher.Projection, scopes []scope) ([]string, []cypher.ProjectionItem, error) {
	var columns []string
	var items []cypher.ProjectionItem

	if p.Star {
		names := boundVariableNames(scopes)
		for _, name := range names {
			columns = append(columns, name)
			items = append(items, cypher.ProjectionItem{Expr: &cypher.Variable{Name: name}})
		}
	}
	for _, it := range p.Items {
		columns = append(columns, columnName(it))
		items = append(items, it)
	}
	if len(items) == 0 {
		return nil, nil, fmt.Errorf("graph: RETURN has no items")
	}
	return columns, items, nil
}

// boundVariableNames returns the sorted union of variable names bound across
// all scopes.
func boundVariableNames(scopes []scope) []string {
	set := map[string]struct{}{}
	for _, s := range scopes {
		for k := range s {
			set[k] = struct{}{}
		}
	}
	names := make([]string, 0, len(set))
	for k := range set {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func distinctRows(rows []projRow) []projRow {
	seen := map[string]struct{}{}
	out := rows[:0]
	for _, r := range rows {
		key := rowKey(r.vals)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

func rowKey(vals []Value) string {
	var b []byte
	for _, v := range vals {
		b = append(b, fmt.Sprintf("%T:%v|", v, v)...)
	}
	return string(b)
}

func (ex *executor) sortRows(rows []projRow, orderBy []cypher.SortItem) error {
	var sortErr error
	sort.SliceStable(rows, func(i, j int) bool {
		for _, item := range orderBy {
			ai, err := ex.ev.eval(item.Expr, rows[i].scope)
			if err != nil {
				sortErr = err
				return false
			}
			aj, err := ex.ev.eval(item.Expr, rows[j].scope)
			if err != nil {
				sortErr = err
				return false
			}
			c, ok := compare(ai, aj)
			if !ok || c == 0 {
				continue
			}
			if item.Dir == cypher.Descending {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	return sortErr
}

func (ex *executor) applySkipLimit(rows []projRow, p *cypher.Projection) ([]projRow, error) {
	if p.Skip != nil {
		n, err := ex.evalCount(p.Skip)
		if err != nil {
			return nil, err
		}
		if n > int64(len(rows)) {
			n = int64(len(rows))
		}
		rows = rows[n:]
	}
	if p.Limit != nil {
		n, err := ex.evalCount(p.Limit)
		if err != nil {
			return nil, err
		}
		if n < int64(len(rows)) {
			rows = rows[:n]
		}
	}
	return rows, nil
}

// evalCount evaluates a SKIP/LIMIT expression to a non-negative integer.
func (ex *executor) evalCount(e cypher.Expr) (int64, error) {
	v, err := ex.ev.eval(e, scope{})
	if err != nil {
		return 0, err
	}
	n, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("graph: SKIP/LIMIT must be an integer")
	}
	if n < 0 {
		return 0, fmt.Errorf("graph: SKIP/LIMIT must be non-negative")
	}
	return n, nil
}

// columnName derives a column header for a projection item.
func columnName(it cypher.ProjectionItem) string {
	if it.Alias != "" {
		return it.Alias
	}
	return exprName(it.Expr)
}

func exprName(e cypher.Expr) string {
	switch x := e.(type) {
	case *cypher.Variable:
		return x.Name
	case *cypher.PropertyAccess:
		return exprName(x.Operand) + "." + x.Name
	default:
		return fmt.Sprintf("%T", e)
	}
}
