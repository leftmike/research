package graph

import "strings"

// Result is the tabular output of a query: one column per RETURN item and one
// row per match.
type Result struct {
	Columns []string
	Rows    [][]Value
}

// String renders the result as a simple text table, useful for debugging and
// examples.
func (r *Result) String() string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.Join(r.Columns, " | "))
	b.WriteByte('\n')
	for _, row := range r.Rows {
		cells := make([]string, len(row))
		for i, v := range row {
			cells[i] = formatValue(v)
		}
		b.WriteString(strings.Join(cells, " | "))
		b.WriteByte('\n')
	}
	return b.String()
}
