package graph

import (
	"fmt"
	"strings"
)

// Value is a runtime value produced by evaluating an expression or stored as a
// property. The supported dynamic types are:
//
//	nil               null
//	bool              boolean
//	int64             integer
//	float64           float
//	string            string
//	[]Value           list
//	map[string]Value  map
//	*Node, *Relationship  graph entities (e.g. from RETURN n)
type Value = any

// truthy reports whether v counts as true in a WHERE predicate. Following
// Cypher, null is not true and non-boolean values are an error at the predicate
// level; here we treat only an explicit true as true and null/false as false.
func truthy(v Value) bool {
	b, ok := v.(bool)
	return ok && b
}

// asFloat converts a numeric value to float64 for arithmetic and comparison.
func asFloat(v Value) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// equal reports value equality. null is not equal to anything, including null.
func equal(a, b Value) bool {
	if a == nil || b == nil {
		return false
	}
	// Numbers compare across int/float.
	if af, ok := asFloat(a); ok {
		if bf, ok := asFloat(b); ok {
			return af == bf
		}
		return false
	}
	switch av := a.(type) {
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []Value:
		bv, ok := b.([]Value)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equal(av[i], bv[i]) {
				return false
			}
		}
		return true
	case *Node:
		bv, ok := b.(*Node)
		return ok && av == bv
	case *Relationship:
		bv, ok := b.(*Relationship)
		return ok && av == bv
	default:
		return false
	}
}

// compare orders two values, returning -1, 0 or 1. The bool result reports
// whether the values are mutually orderable; it is false when either is null or
// the types are not comparable. Used by comparison operators and ORDER BY.
func compare(a, b Value) (int, bool) {
	if a == nil || b == nil {
		return 0, false
	}
	if af, ok := asFloat(a); ok {
		if bf, ok := asFloat(b); ok {
			switch {
			case af < bf:
				return -1, true
			case af > bf:
				return 1, true
			default:
				return 0, true
			}
		}
		return 0, false
	}
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		if !ok {
			return 0, false
		}
		return strings.Compare(av, bv), true
	case bool:
		bv, ok := b.(bool)
		if !ok {
			return 0, false
		}
		switch {
		case !av && bv:
			return -1, true
		case av && !bv:
			return 1, true
		default:
			return 0, true
		}
	default:
		return 0, false
	}
}

// formatValue renders a value for the Result table and tests.
func formatValue(v Value) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int64:
		return fmt.Sprintf("%d", x)
	case float64:
		return fmt.Sprintf("%g", x)
	case []Value:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = formatValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Node:
		return fmt.Sprintf("(%d%s)", x.ID, labelString(x.Labels))
	case *Relationship:
		return fmt.Sprintf("[%d:%s]", x.ID, x.Type)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func labelString(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	return ":" + strings.Join(labels, ":")
}
