package graph

import (
	"fmt"
	"math"
	"strings"

	"github.com/leftmike/research/cypher"
)

// scope is the set of variable bindings for a single candidate match.
type scope map[string]Value

// evaluator evaluates expressions against a scope. It carries query parameters
// referenced by $name / $1.
type evaluator struct {
	params map[string]Value
}

// eval computes the value of an expression in the given scope.
func (ev *evaluator) eval(e cypher.Expr, s scope) (Value, error) {
	switch x := e.(type) {
	case *cypher.Literal:
		return literalValue(x), nil

	case *cypher.Variable:
		v, ok := s[x.Name]
		if !ok {
			return nil, fmt.Errorf("graph: unknown variable %q", x.Name)
		}
		return v, nil

	case *cypher.Parameter:
		v, ok := ev.params[x.Name]
		if !ok {
			return nil, fmt.Errorf("graph: missing parameter $%s", x.Name)
		}
		return v, nil

	case *cypher.PropertyAccess:
		operand, err := ev.eval(x.Operand, s)
		if err != nil {
			return nil, err
		}
		return propertyOf(operand, x.Name), nil

	case *cypher.ListLiteral:
		out := make([]Value, len(x.Elements))
		for i, el := range x.Elements {
			v, err := ev.eval(el, s)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil

	case *cypher.MapLiteral:
		m := make(map[string]Value, len(x.Keys))
		for i, k := range x.Keys {
			v, err := ev.eval(x.Values[i], s)
			if err != nil {
				return nil, err
			}
			m[k] = v
		}
		return m, nil

	case *cypher.UnaryExpr:
		return ev.evalUnary(x, s)

	case *cypher.BinaryExpr:
		return ev.evalBinary(x, s)

	case *cypher.Not:
		v, err := ev.eval(x.Operand, s)
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, nil
		}
		return !truthy(v), nil

	case *cypher.Comparison:
		return ev.evalComparison(x, s)

	case *cypher.StringOp:
		return ev.evalStringOp(x, s)

	case *cypher.In:
		return ev.evalIn(x, s)

	case *cypher.IsNull:
		v, err := ev.eval(x.Operand, s)
		if err != nil {
			return nil, err
		}
		isNull := v == nil
		if x.Negated {
			return !isNull, nil
		}
		return isNull, nil

	case *cypher.LabelsCheck:
		v, err := ev.eval(x.Operand, s)
		if err != nil {
			return nil, err
		}
		n, ok := v.(*Node)
		if !ok {
			return false, nil
		}
		return n.hasLabels(x.Labels), nil

	case *cypher.Index:
		return ev.evalIndex(x, s)

	case *cypher.Slice:
		return ev.evalSlice(x, s)

	case *cypher.CaseExpr:
		return ev.evalCase(x, s)

	case *cypher.FunctionCall:
		return ev.evalFunction(x, s)

	default:
		return nil, fmt.Errorf("graph: unsupported expression %T", e)
	}
}

func literalValue(l *cypher.Literal) Value {
	switch l.Kind {
	case cypher.IntLit:
		return l.Int
	case cypher.FloatLit:
		return l.Float
	case cypher.StringLit:
		return l.Str
	case cypher.BoolLit:
		return l.Bool
	default: // NullLit
		return nil
	}
}

// propertyOf returns the named property of a node, relationship or map, or nil
// if absent.
func propertyOf(operand Value, name string) Value {
	switch o := operand.(type) {
	case *Node:
		return o.Props[name]
	case *Relationship:
		return o.Props[name]
	case map[string]Value:
		return o[name]
	default:
		return nil
	}
}

func (ev *evaluator) evalUnary(x *cypher.UnaryExpr, s scope) (Value, error) {
	v, err := ev.eval(x.Operand, s)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	if x.Op == cypher.PLUS {
		return v, nil
	}
	// MINUS
	switch n := v.(type) {
	case int64:
		return -n, nil
	case float64:
		return -n, nil
	default:
		return nil, fmt.Errorf("graph: cannot negate %T", v)
	}
}

func (ev *evaluator) evalBinary(x *cypher.BinaryExpr, s scope) (Value, error) {
	// Logical operators short-circuit and tolerate nulls.
	switch x.Op {
	case cypher.AND, cypher.OR, cypher.XOR:
		return ev.evalLogical(x, s)
	}

	left, err := ev.eval(x.Left, s)
	if err != nil {
		return nil, err
	}
	right, err := ev.eval(x.Right, s)
	if err != nil {
		return nil, err
	}
	if left == nil || right == nil {
		return nil, nil
	}

	// String concatenation with +.
	if x.Op == cypher.PLUS {
		if ls, ok := left.(string); ok {
			if rs, ok := right.(string); ok {
				return ls + rs, nil
			}
		}
	}

	lf, lok := asFloat(left)
	rf, rok := asFloat(right)
	if !lok || !rok {
		return nil, fmt.Errorf("graph: non-numeric operand to %s", x.Op)
	}
	_, lInt := left.(int64)
	_, rInt := right.(int64)
	bothInt := lInt && rInt

	switch x.Op {
	case cypher.PLUS:
		if bothInt {
			return left.(int64) + right.(int64), nil
		}
		return lf + rf, nil
	case cypher.MINUS:
		if bothInt {
			return left.(int64) - right.(int64), nil
		}
		return lf - rf, nil
	case cypher.STAR:
		if bothInt {
			return left.(int64) * right.(int64), nil
		}
		return lf * rf, nil
	case cypher.SLASH:
		if bothInt {
			if right.(int64) == 0 {
				return nil, fmt.Errorf("graph: integer division by zero")
			}
			return left.(int64) / right.(int64), nil
		}
		return lf / rf, nil
	case cypher.PERCENT:
		if bothInt {
			if right.(int64) == 0 {
				return nil, fmt.Errorf("graph: integer modulo by zero")
			}
			return left.(int64) % right.(int64), nil
		}
		return math.Mod(lf, rf), nil
	case cypher.CARET:
		return math.Pow(lf, rf), nil
	default:
		return nil, fmt.Errorf("graph: unsupported binary operator %s", x.Op)
	}
}

func (ev *evaluator) evalLogical(x *cypher.BinaryExpr, s scope) (Value, error) {
	left, err := ev.eval(x.Left, s)
	if err != nil {
		return nil, err
	}
	right, err := ev.eval(x.Right, s)
	if err != nil {
		return nil, err
	}
	lb, lok := left.(bool)
	rb, rok := right.(bool)

	switch x.Op {
	case cypher.AND:
		if (lok && !lb) || (rok && !rb) {
			return false, nil
		}
		if lok && rok {
			return true, nil
		}
		return nil, nil
	case cypher.OR:
		if (lok && lb) || (rok && rb) {
			return true, nil
		}
		if lok && rok {
			return false, nil
		}
		return nil, nil
	default: // XOR
		if !lok || !rok {
			return nil, nil
		}
		return lb != rb, nil
	}
}

func (ev *evaluator) evalComparison(x *cypher.Comparison, s scope) (Value, error) {
	prev, err := ev.eval(x.Operands[0], s)
	if err != nil {
		return nil, err
	}
	result := true
	for i, op := range x.Ops {
		cur, err := ev.eval(x.Operands[i+1], s)
		if err != nil {
			return nil, err
		}
		ok, isNull := compareOp(op, prev, cur)
		if isNull {
			return nil, nil
		}
		if !ok {
			result = false
		}
		prev = cur
	}
	return result, nil
}

// compareOp applies a single comparison operator, returning the boolean result
// and whether the comparison is null (an operand was null or unorderable).
func compareOp(op cypher.TokenType, a, b Value) (result bool, isNull bool) {
	if op == cypher.EQ {
		if a == nil || b == nil {
			return false, true
		}
		return equal(a, b), false
	}
	if op == cypher.NE {
		if a == nil || b == nil {
			return false, true
		}
		return !equal(a, b), false
	}
	c, ok := compare(a, b)
	if !ok {
		return false, true
	}
	switch op {
	case cypher.LT:
		return c < 0, false
	case cypher.GT:
		return c > 0, false
	case cypher.LE:
		return c <= 0, false
	case cypher.GE:
		return c >= 0, false
	default:
		return false, true
	}
}

func (ev *evaluator) evalStringOp(x *cypher.StringOp, s scope) (Value, error) {
	left, err := ev.eval(x.Left, s)
	if err != nil {
		return nil, err
	}
	right, err := ev.eval(x.Right, s)
	if err != nil {
		return nil, err
	}
	ls, lok := left.(string)
	rs, rok := right.(string)
	if !lok || !rok {
		return nil, nil
	}
	switch x.Kind {
	case cypher.StartsWith:
		return strings.HasPrefix(ls, rs), nil
	case cypher.EndsWith:
		return strings.HasSuffix(ls, rs), nil
	default: // Contains
		return strings.Contains(ls, rs), nil
	}
}

func (ev *evaluator) evalIn(x *cypher.In, s scope) (Value, error) {
	left, err := ev.eval(x.Left, s)
	if err != nil {
		return nil, err
	}
	right, err := ev.eval(x.Right, s)
	if err != nil {
		return nil, err
	}
	list, ok := right.([]Value)
	if !ok {
		if right == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("graph: IN requires a list, got %T", right)
	}
	if left == nil {
		return nil, nil
	}
	sawNull := false
	for _, el := range list {
		if el == nil {
			sawNull = true
			continue
		}
		if equal(left, el) {
			return true, nil
		}
	}
	if sawNull {
		return nil, nil
	}
	return false, nil
}

func (ev *evaluator) evalIndex(x *cypher.Index, s scope) (Value, error) {
	operand, err := ev.eval(x.Operand, s)
	if err != nil {
		return nil, err
	}
	idx, err := ev.eval(x.Index, s)
	if err != nil {
		return nil, err
	}
	if operand == nil || idx == nil {
		return nil, nil
	}
	switch o := operand.(type) {
	case []Value:
		i, ok := idx.(int64)
		if !ok {
			return nil, fmt.Errorf("graph: list index must be an integer")
		}
		i = normalizeIndex(i, len(o))
		if i < 0 || i >= int64(len(o)) {
			return nil, nil
		}
		return o[i], nil
	case map[string]Value:
		key, ok := idx.(string)
		if !ok {
			return nil, fmt.Errorf("graph: map index must be a string")
		}
		return o[key], nil
	default:
		// Property access via string index on a node/relationship.
		if key, ok := idx.(string); ok {
			return propertyOf(operand, key), nil
		}
		return nil, fmt.Errorf("graph: cannot index %T", operand)
	}
}

func (ev *evaluator) evalSlice(x *cypher.Slice, s scope) (Value, error) {
	operand, err := ev.eval(x.Operand, s)
	if err != nil {
		return nil, err
	}
	list, ok := operand.([]Value)
	if !ok {
		if operand == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("graph: slice requires a list, got %T", operand)
	}
	n := int64(len(list))
	low, high := int64(0), n
	if x.Low != nil {
		lv, err := ev.eval(x.Low, s)
		if err != nil {
			return nil, err
		}
		if i, ok := lv.(int64); ok {
			low = clampIndex(normalizeIndex(i, len(list)), n)
		}
	}
	if x.High != nil {
		hv, err := ev.eval(x.High, s)
		if err != nil {
			return nil, err
		}
		if i, ok := hv.(int64); ok {
			high = clampIndex(normalizeIndex(i, len(list)), n)
		}
	}
	if low > high {
		return []Value{}, nil
	}
	out := make([]Value, high-low)
	copy(out, list[low:high])
	return out, nil
}

// normalizeIndex turns a negative index into one counted from the end.
func normalizeIndex(i int64, length int) int64 {
	if i < 0 {
		return int64(length) + i
	}
	return i
}

func clampIndex(i, n int64) int64 {
	if i < 0 {
		return 0
	}
	if i > n {
		return n
	}
	return i
}

func (ev *evaluator) evalCase(x *cypher.CaseExpr, s scope) (Value, error) {
	if x.Subject != nil {
		subject, err := ev.eval(x.Subject, s)
		if err != nil {
			return nil, err
		}
		for _, w := range x.Whens {
			wv, err := ev.eval(w.When, s)
			if err != nil {
				return nil, err
			}
			if equal(subject, wv) {
				return ev.eval(w.Then, s)
			}
		}
	} else {
		for _, w := range x.Whens {
			wv, err := ev.eval(w.When, s)
			if err != nil {
				return nil, err
			}
			if truthy(wv) {
				return ev.eval(w.Then, s)
			}
		}
	}
	if x.Else != nil {
		return ev.eval(x.Else, s)
	}
	return nil, nil
}
