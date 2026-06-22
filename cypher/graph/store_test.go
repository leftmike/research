package graph

import "testing"

func TestStoreAddAndIndex(t *testing.T) {
	g := New()
	a := g.AddNode([]string{"Person"}, map[string]Value{"name": "Alice"})
	b := g.AddNode([]string{"Person"}, map[string]Value{"name": "Bob"})
	r := g.AddRelationship("KNOWS", a, b, map[string]Value{"since": int64(2020)})

	if g.Nodes() != 2 {
		t.Fatalf("Nodes() = %d, want 2", g.Nodes())
	}
	if g.Relationships() != 1 {
		t.Fatalf("Relationships() = %d, want 1", g.Relationships())
	}
	if a.ID == b.ID {
		t.Fatalf("node ids not unique: %d", a.ID)
	}
	if r.Start != a || r.End != b {
		t.Fatalf("relationship endpoints wrong")
	}

	out := g.step(a, dirRight, []string{"KNOWS"})
	if len(out) != 1 || out[0].node != b {
		t.Fatalf("outgoing step from a = %+v, want b", out)
	}
	if got := g.step(b, dirRight, nil); len(got) != 0 {
		t.Fatalf("b has no outgoing edges, got %d", len(got))
	}
	in := g.step(b, dirLeft, nil)
	if len(in) != 1 || in[0].node != a {
		t.Fatalf("incoming step to b = %+v, want a", in)
	}
	both := g.step(a, dirBoth, nil)
	if len(both) != 1 {
		t.Fatalf("undirected step from a = %d, want 1", len(both))
	}
}

func TestNodeLabels(t *testing.T) {
	g := New()
	n := g.AddNode([]string{"Person", "Admin"}, nil)
	if !n.hasLabels([]string{"Person", "Admin"}) {
		t.Fatalf("expected both labels to match")
	}
	if n.hasLabel("Robot") {
		t.Fatalf("did not expect Robot label")
	}
}
