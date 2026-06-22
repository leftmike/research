package graph

import (
	"sort"
	"testing"

	"github.com/leftmike/research/cypher"
)

// fixture builds a small social graph:
//
//	Alice (34) -KNOWS-> Bob (27)
//	Alice (34) -KNOWS-> Carol (41)
//	Bob   (27) -KNOWS-> Carol (41)
//	Dave  (52) is isolated
func fixture() *Graph {
	g := New()
	alice := g.AddNode([]string{"Person"}, map[string]Value{"name": "Alice", "age": int64(34)})
	bob := g.AddNode([]string{"Person"}, map[string]Value{"name": "Bob", "age": int64(27)})
	carol := g.AddNode([]string{"Person"}, map[string]Value{"name": "Carol", "age": int64(41)})
	g.AddNode([]string{"Person"}, map[string]Value{"name": "Dave", "age": int64(52)})
	g.AddRelationship("KNOWS", alice, bob, map[string]Value{"since": int64(2015)})
	g.AddRelationship("KNOWS", alice, carol, map[string]Value{"since": int64(2018)})
	g.AddRelationship("KNOWS", bob, carol, map[string]Value{"since": int64(2020)})
	return g
}

// column returns the values of one result column as strings, sorted, for
// order-insensitive comparison.
func column(res *Result, idx int) []string {
	out := make([]string, len(res.Rows))
	for i, row := range res.Rows {
		out[i] = formatValue(row[idx])
	}
	sort.Strings(out)
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMatchNodes(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (n:Person) RETURN n.name")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"n.name"}; !eq(res.Columns, want) {
		t.Fatalf("columns = %v, want %v", res.Columns, want)
	}
	got := column(res, 0)
	want := []string{"Alice", "Bob", "Carol", "Dave"}
	if !eq(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
}

func TestMatchWhere(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (n:Person) WHERE n.age > 30 RETURN n.name")
	if err != nil {
		t.Fatal(err)
	}
	got := column(res, 0)
	want := []string{"Alice", "Carol", "Dave"}
	if !eq(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
}

func TestMatchOrderBySkipLimit(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (n:Person) RETURN n.name ORDER BY n.age DESC SKIP 1 LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	// Ages desc: Dave(52), Carol(41), Bob(27)... wait Alice(34). order: Dave, Carol, Alice, Bob.
	// Skip 1 -> Carol, Alice, Bob; limit 2 -> Carol, Alice.
	if len(res.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(res.Rows))
	}
	if formatValue(res.Rows[0][0]) != "Carol" || formatValue(res.Rows[1][0]) != "Alice" {
		t.Fatalf("rows = %v, want [Carol Alice]", [][]Value{res.Rows[0], res.Rows[1]})
	}
}

func TestMatchRelationship(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (a:Person)-[:KNOWS]->(b:Person) RETURN a.name, b.name")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(res.Rows))
	}
	// Alice knows Bob and Carol; Bob knows Carol.
	pairs := map[string]bool{}
	for _, row := range res.Rows {
		pairs[formatValue(row[0])+"->"+formatValue(row[1])] = true
	}
	for _, want := range []string{"Alice->Bob", "Alice->Carol", "Bob->Carol"} {
		if !pairs[want] {
			t.Errorf("missing pair %s", want)
		}
	}
}

func TestMatchRelationshipProperty(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (a)-[r:KNOWS]->(b) WHERE r.since >= 2018 RETURN a.name, b.name, r.since")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(res.Rows))
	}
}

func TestMatchInlineProperty(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (n:Person {name: 'Bob'}) RETURN n.age")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 1 || formatValue(res.Rows[0][0]) != "27" {
		t.Fatalf("rows = %v, want single 27", res.Rows)
	}
}

func TestReturnDistinct(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (a:Person)-[:KNOWS]->(b:Person) RETURN DISTINCT b.name")
	if err != nil {
		t.Fatal(err)
	}
	got := column(res, 0)
	want := []string{"Bob", "Carol"}
	if !eq(got, want) {
		t.Fatalf("distinct names = %v, want %v", got, want)
	}
}

func TestReturnStar(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (n:Person {name: 'Alice'})-[r:KNOWS]->(m) RETURN *")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"m", "n", "r"}
	if !eq(res.Columns, want) {
		t.Fatalf("columns = %v, want %v", res.Columns, want)
	}
}

func TestReturnAlias(t *testing.T) {
	g := fixture()
	res, err := g.Query("MATCH (n:Person {name: 'Alice'}) RETURN n.name AS who, n.age AS years")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"who", "years"}; !eq(res.Columns, want) {
		t.Fatalf("columns = %v, want %v", res.Columns, want)
	}
}

func TestRunWithParams(t *testing.T) {
	g := fixture()
	q := mustParse(t, "MATCH (n:Person) WHERE n.age >= $min RETURN n.name")
	res, err := g.RunWithParams(q, map[string]Value{"min": int64(41)})
	if err != nil {
		t.Fatal(err)
	}
	got := column(res, 0)
	want := []string{"Carol", "Dave"}
	if !eq(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
}

func TestUnsupportedFeatures(t *testing.T) {
	g := fixture()
	for _, src := range []string{
		"OPTIONAL MATCH (n:Person) RETURN n.name",
		"MATCH (a)-[:KNOWS*1..3]->(b) RETURN a.name",
		"UNWIND [1, 2, 3] AS x RETURN x",
		"MATCH (n:Person) WITH n RETURN n.name",
	} {
		if _, err := g.Query(src); err == nil {
			t.Errorf("expected error for %q, got nil", src)
		}
	}
}

func mustParse(t *testing.T, src string) *cypher.Query {
	t.Helper()
	q, err := cypher.Parse(src)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return q
}
