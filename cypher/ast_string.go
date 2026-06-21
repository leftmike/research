package cypher

import (
	"strconv"
	"strings"
)

// String renders a canonical Cypher form of the query. Compound expressions
// are fully parenthesized so the output unambiguously reflects the parsed
// structure; this makes it convenient for round-trip testing.
func (q *Query) String() string {
	parts := make([]string, len(q.Clauses))
	for i, c := range q.Clauses {
		parts[i] = clauseString(c)
	}
	return strings.Join(parts, " ")
}

func clauseString(c Clause) string {
	switch c := c.(type) {
	case *Match:
		var sb strings.Builder
		if c.Optional {
			sb.WriteString("OPTIONAL ")
		}
		sb.WriteString("MATCH ")
		sb.WriteString(patternString(c.Pattern))
		if c.Where != nil {
			sb.WriteString(" WHERE ")
			sb.WriteString(exprString(c.Where))
		}
		return sb.String()
	case *Unwind:
		return "UNWIND " + exprString(c.Expr) + " AS " + c.Variable
	case *With:
		s := "WITH " + projectionString(c.Projection)
		if c.Where != nil {
			s += " WHERE " + exprString(c.Where)
		}
		return s
	case *Return:
		return "RETURN " + projectionString(c.Projection)
	default:
		return "<?clause>"
	}
}

func projectionString(p Projection) string {
	var sb strings.Builder
	if p.Distinct {
		sb.WriteString("DISTINCT ")
	}
	if p.Star {
		sb.WriteString("*")
	} else {
		items := make([]string, len(p.Items))
		for i, it := range p.Items {
			s := exprString(it.Expr)
			if it.Alias != "" {
				s += " AS " + it.Alias
			}
			items[i] = s
		}
		sb.WriteString(strings.Join(items, ", "))
	}
	if len(p.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		sorts := make([]string, len(p.OrderBy))
		for i, si := range p.OrderBy {
			s := exprString(si.Expr)
			if si.Dir == Descending {
				s += " DESC"
			}
			sorts[i] = s
		}
		sb.WriteString(strings.Join(sorts, ", "))
	}
	if p.Skip != nil {
		sb.WriteString(" SKIP " + exprString(p.Skip))
	}
	if p.Limit != nil {
		sb.WriteString(" LIMIT " + exprString(p.Limit))
	}
	return sb.String()
}

func patternString(parts []PatternPart) string {
	ps := make([]string, len(parts))
	for i, part := range parts {
		s := ""
		if part.Variable != "" {
			s = part.Variable + " = "
		}
		s += patternElementString(part.Element)
		ps[i] = s
	}
	return strings.Join(ps, ", ")
}

func patternElementString(el PatternElement) string {
	var sb strings.Builder
	sb.WriteString(nodeString(el.Nodes[0]))
	for i, rel := range el.Rels {
		sb.WriteString(relString(rel))
		sb.WriteString(nodeString(el.Nodes[i+1]))
	}
	return sb.String()
}

func nodeString(n NodePattern) string {
	var sb strings.Builder
	sb.WriteString("(")
	sb.WriteString(n.Variable)
	for _, l := range n.Labels {
		sb.WriteString(":" + l)
	}
	if n.Properties != nil {
		if n.Variable != "" || len(n.Labels) > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(exprString(n.Properties))
	}
	sb.WriteString(")")
	return sb.String()
}

func relString(r RelationshipPattern) string {
	detail := relDetailString(r)
	switch r.Direction {
	case RelLeft:
		return "<-" + detail + "-"
	case RelRight:
		return "-" + detail + "->"
	default:
		return "-" + detail + "-"
	}
}

func relDetailString(r RelationshipPattern) string {
	var sb strings.Builder
	sb.WriteString(r.Variable)
	if len(r.Types) > 0 {
		sb.WriteString(":" + strings.Join(r.Types, "|"))
	}
	if r.VarLength {
		sb.WriteString("*")
		sb.WriteString(hopsString(r.MinHops, r.MaxHops))
	}
	if r.Properties != nil {
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(exprString(r.Properties))
	}
	if sb.Len() == 0 {
		return ""
	}
	return "[" + sb.String() + "]"
}

func hopsString(min, max *int) string {
	switch {
	case min != nil && max != nil && *min == *max:
		return strconv.Itoa(*min)
	case min != nil && max != nil:
		return strconv.Itoa(*min) + ".." + strconv.Itoa(*max)
	case min != nil:
		return strconv.Itoa(*min) + ".."
	case max != nil:
		return ".." + strconv.Itoa(*max)
	default:
		return ""
	}
}

func exprString(e Expr) string {
	switch e := e.(type) {
	case *BinaryExpr:
		return "(" + exprString(e.Left) + " " + e.Op.String() + " " + exprString(e.Right) + ")"
	case *Comparison:
		var sb strings.Builder
		sb.WriteString("(")
		sb.WriteString(exprString(e.Operands[0]))
		for i, op := range e.Ops {
			sb.WriteString(" " + op.String() + " ")
			sb.WriteString(exprString(e.Operands[i+1]))
		}
		sb.WriteString(")")
		return sb.String()
	case *UnaryExpr:
		return "(" + e.Op.String() + exprString(e.Operand) + ")"
	case *Not:
		return "(NOT " + exprString(e.Operand) + ")"
	case *StringOp:
		op := map[StringOpKind]string{
			StartsWith: "STARTS WITH", EndsWith: "ENDS WITH", Contains: "CONTAINS",
		}[e.Kind]
		return "(" + exprString(e.Left) + " " + op + " " + exprString(e.Right) + ")"
	case *In:
		return "(" + exprString(e.Left) + " IN " + exprString(e.Right) + ")"
	case *IsNull:
		if e.Negated {
			return "(" + exprString(e.Operand) + " IS NOT NULL)"
		}
		return "(" + exprString(e.Operand) + " IS NULL)"
	case *PropertyAccess:
		return exprString(e.Operand) + "." + e.Name
	case *Index:
		return exprString(e.Operand) + "[" + exprString(e.Index) + "]"
	case *Slice:
		low, high := "", ""
		if e.Low != nil {
			low = exprString(e.Low)
		}
		if e.High != nil {
			high = exprString(e.High)
		}
		return exprString(e.Operand) + "[" + low + ".." + high + "]"
	case *LabelsCheck:
		return exprString(e.Operand) + ":" + strings.Join(e.Labels, ":")
	case *FunctionCall:
		var sb strings.Builder
		sb.WriteString(e.Name + "(")
		if e.Distinct {
			sb.WriteString("DISTINCT ")
		}
		if e.Star {
			sb.WriteString("*")
		} else {
			args := make([]string, len(e.Args))
			for i, a := range e.Args {
				args[i] = exprString(a)
			}
			sb.WriteString(strings.Join(args, ", "))
		}
		sb.WriteString(")")
		return sb.String()
	case *CaseExpr:
		var sb strings.Builder
		sb.WriteString("CASE")
		if e.Subject != nil {
			sb.WriteString(" " + exprString(e.Subject))
		}
		for _, w := range e.Whens {
			sb.WriteString(" WHEN " + exprString(w.When) + " THEN " + exprString(w.Then))
		}
		if e.Else != nil {
			sb.WriteString(" ELSE " + exprString(e.Else))
		}
		sb.WriteString(" END")
		return sb.String()
	case *ListLiteral:
		elems := make([]string, len(e.Elements))
		for i, el := range e.Elements {
			elems[i] = exprString(el)
		}
		return "[" + strings.Join(elems, ", ") + "]"
	case *MapLiteral:
		pairs := make([]string, len(e.Keys))
		for i := range e.Keys {
			pairs[i] = e.Keys[i] + ": " + exprString(e.Values[i])
		}
		return "{" + strings.Join(pairs, ", ") + "}"
	case *Variable:
		return e.Name
	case *Parameter:
		return "$" + e.Name
	case *Literal:
		return literalString(e)
	default:
		return "<?expr>"
	}
}

func literalString(l *Literal) string {
	switch l.Kind {
	case IntLit:
		return strconv.FormatInt(l.Int, 10)
	case FloatLit:
		return strconv.FormatFloat(l.Float, 'g', -1, 64)
	case StringLit:
		return quoteString(l.Str)
	case BoolLit:
		if l.Bool {
			return "true"
		}
		return "false"
	case NullLit:
		return "null"
	default:
		return "<?lit>"
	}
}

func quoteString(s string) string {
	var sb strings.Builder
	sb.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\'':
			sb.WriteString(`\'`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('\'')
	return sb.String()
}
