package graph

import (
	"testing"

	"github.com/leftmike/research/cypher"
)

// evalExprString parses src as a RETURN expression and evaluates it in the
// given scope with the given parameters.
func evalExprString(t *testing.T, src string, s scope, params map[string]Value) Value {
	t.Helper()
	q, err := cypher.Parse("RETURN " + src)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	ret := q.Clauses[len(q.Clauses)-1].(*cypher.Return)
	ev := &evaluator{params: params}
	v, err := ev.eval(ret.Items[0].Expr, s)
	if err != nil {
		t.Fatalf("eval %q: %v", src, err)
	}
	return v
}

func TestEvalExpressions(t *testing.T) {
	tests := []struct {
		src  string
		want Value
	}{
		{"1 + 2", int64(3)},
		{"3 * 4 - 2", int64(10)},
		{"7 / 2", int64(3)},
		{"7.0 / 2", 3.5},
		{"2 ^ 10", float64(1024)},
		{"1 < 2", true},
		{"2 <= 2", true},
		{"1 = 1", true},
		{"1 <> 2", true},
		{"1 < 2 AND 2 < 3", true},
		{"1 < 2 AND 2 > 3", false},
		{"true OR false", true},
		{"NOT false", true},
		{"'foo' + 'bar'", "foobar"},
		{"'hello' STARTS WITH 'he'", true},
		{"'hello' ENDS WITH 'lo'", true},
		{"'hello' CONTAINS 'ell'", true},
		{"2 IN [1, 2, 3]", true},
		{"5 IN [1, 2, 3]", false},
		{"null IS NULL", true},
		{"1 IS NOT NULL", true},
		{"[10, 20, 30][1]", int64(20)},
		{"[10, 20, 30][-1]", int64(30)},
		{"[1, 2, 3, 4][1..3]", []Value{int64(2), int64(3)}},
		{"CASE WHEN 1 > 2 THEN 'a' ELSE 'b' END", "b"},
		{"CASE 2 WHEN 1 THEN 'a' WHEN 2 THEN 'b' END", "b"},
		{"toUpper('abc')", "ABC"},
		{"toLower('ABC')", "abc"},
		{"size([1, 2, 3])", int64(3)},
		{"length('hello')", int64(5)},
		{"coalesce(null, null, 7)", int64(7)},
	}
	for _, tt := range tests {
		got := evalExprString(t, tt.src, scope{}, nil)
		if !valuesEqual(got, tt.want) {
			t.Errorf("%s = %#v, want %#v", tt.src, got, tt.want)
		}
	}
}

func TestEvalParameter(t *testing.T) {
	got := evalExprString(t, "$min + 1", scope{}, map[string]Value{"min": int64(41)})
	if got != int64(42) {
		t.Errorf("got %v, want 42", got)
	}
}

func TestEvalPropertyAccess(t *testing.T) {
	g := New()
	n := g.AddNode([]string{"Person"}, map[string]Value{"name": "Alice", "age": int64(30)})
	s := scope{"n": n}
	if got := evalExprString(t, "n.name", s, nil); got != "Alice" {
		t.Errorf("n.name = %v, want Alice", got)
	}
	if got := evalExprString(t, "n.missing", s, nil); got != nil {
		t.Errorf("n.missing = %v, want nil", got)
	}
}

// valuesEqual is a test helper handling lists, which the runtime equal() treats
// element-wise but never returns for the nil case.
func valuesEqual(a, b Value) bool {
	al, aok := a.([]Value)
	bl, bok := b.([]Value)
	if aok || bok {
		if !aok || !bok || len(al) != len(bl) {
			return false
		}
		for i := range al {
			if !valuesEqual(al[i], bl[i]) {
				return false
			}
		}
		return true
	}
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return equal(a, b)
}
