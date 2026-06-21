package cypher

import "testing"

// TestParseRoundTrip parses each source and compares the canonical String()
// rendering of the AST to the expected normalized form.
func TestParseRoundTrip(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		// Clauses.
		{src: "MATCH (n) RETURN n", want: "MATCH (n) RETURN n"},
		{src: "match (n) return n", want: "MATCH (n) RETURN n"},
		{src: "OPTIONAL MATCH (n) RETURN n", want: "OPTIONAL MATCH (n) RETURN n"},
		{src: "RETURN 1", want: "RETURN 1"},
		{src: "RETURN *", want: "RETURN *"},
		{src: "RETURN DISTINCT n", want: "RETURN DISTINCT n"},
		{src: "UNWIND [1, 2, 3] AS x RETURN x", want: "UNWIND [1, 2, 3] AS x RETURN x"},
		{
			src:  "MATCH (n) WITH n WHERE n.age > 18 RETURN n",
			want: "MATCH (n) WITH n WHERE (n.age > 18) RETURN n",
		},

		// Projection modifiers.
		{
			src:  "MATCH (n) RETURN n.name AS name ORDER BY n.age DESC, name SKIP 5 LIMIT 10",
			want: "MATCH (n) RETURN n.name AS name ORDER BY n.age DESC, name SKIP 5 LIMIT 10",
		},
		{src: "RETURN x ORDER BY x ASC", want: "RETURN x ORDER BY x"},

		// Node patterns.
		{src: "MATCH (:Person) RETURN 1", want: "MATCH (:Person) RETURN 1"},
		{src: "MATCH (n:Person:Admin) RETURN n", want: "MATCH (n:Person:Admin) RETURN n"},
		{
			src:  "MATCH (n:Person {name: 'Bob', age: 30}) RETURN n",
			want: "MATCH (n:Person {name: 'Bob', age: 30}) RETURN n",
		},
		{src: "MATCH (n {x: 1}) RETURN n", want: "MATCH (n {x: 1}) RETURN n"},
		{src: "MATCH () RETURN 1", want: "MATCH () RETURN 1"},

		// Relationship patterns and directions.
		{
			src:  "MATCH (a)-[:KNOWS]->(b) RETURN a",
			want: "MATCH (a)-[:KNOWS]->(b) RETURN a",
		},
		{
			src:  "MATCH (a)<-[r:KNOWS]-(b) RETURN r",
			want: "MATCH (a)<-[r:KNOWS]-(b) RETURN r",
		},
		{
			src:  "MATCH (a)-[r]-(b) RETURN r",
			want: "MATCH (a)-[r]-(b) RETURN r",
		},
		{
			src:  "MATCH (a)-->(b) RETURN a",
			want: "MATCH (a)-->(b) RETURN a",
		},
		{
			src:  "MATCH (a)-[:A|B|C]->(b) RETURN a",
			want: "MATCH (a)-[:A|B|C]->(b) RETURN a",
		},
		{
			src:  "MATCH (a)-[:KNOWS*1..3]->(b) RETURN a",
			want: "MATCH (a)-[:KNOWS*1..3]->(b) RETURN a",
		},
		{
			src:  "MATCH (a)-[*]->(b) RETURN a",
			want: "MATCH (a)-[*]->(b) RETURN a",
		},
		{
			src:  "MATCH (a)-[*2]->(b) RETURN a",
			want: "MATCH (a)-[*2]->(b) RETURN a",
		},
		{
			src:  "MATCH (a)-[*2..]->(b) RETURN a",
			want: "MATCH (a)-[*2..]->(b) RETURN a",
		},
		{
			src:  "MATCH (a)-[*..5]->(b) RETURN a",
			want: "MATCH (a)-[*..5]->(b) RETURN a",
		},
		{
			src:  "MATCH p = (a)-[:R]->(b)-[:R]->(c) RETURN p",
			want: "MATCH p = (a)-[:R]->(b)-[:R]->(c) RETURN p",
		},
		{
			src:  "MATCH (a), (b) RETURN a, b",
			want: "MATCH (a), (b) RETURN a, b",
		},

		// Expression precedence and operators.
		{src: "RETURN 1 + 2 * 3", want: "RETURN (1 + (2 * 3))"},
		{src: "RETURN (1 + 2) * 3", want: "RETURN ((1 + 2) * 3)"},
		{src: "RETURN 2 ^ 3 ^ 2", want: "RETURN (2 ^ (3 ^ 2))"},
		{src: "RETURN -1 + 2", want: "RETURN ((-1) + 2)"},
		{src: "RETURN a OR b AND c", want: "RETURN (a OR (b AND c))"},
		{src: "RETURN NOT a AND b", want: "RETURN ((NOT a) AND b)"},
		{src: "RETURN a = b", want: "RETURN (a = b)"},
		{src: "RETURN 1 < 2 <= 3", want: "RETURN (1 < 2 <= 3)"},
		{src: "RETURN a <> b", want: "RETURN (a <> b)"},
		{src: "RETURN n.name STARTS WITH 'A'", want: "RETURN (n.name STARTS WITH 'A')"},
		{src: "RETURN n.name ENDS WITH 'z'", want: "RETURN (n.name ENDS WITH 'z')"},
		{src: "RETURN n.name CONTAINS 'b'", want: "RETURN (n.name CONTAINS 'b')"},
		{src: "RETURN x IN [1, 2, 3]", want: "RETURN (x IN [1, 2, 3])"},
		{src: "RETURN x IS NULL", want: "RETURN (x IS NULL)"},
		{src: "RETURN x IS NOT NULL", want: "RETURN (x IS NOT NULL)"},
		{src: "RETURN n:Person", want: "RETURN n:Person"},
		{src: "RETURN n:Person:Admin", want: "RETURN n:Person:Admin"},

		// Postfix: property, index, slice.
		{src: "RETURN a.b.c", want: "RETURN a.b.c"},
		{src: "RETURN xs[0]", want: "RETURN xs[0]"},
		{src: "RETURN xs[1..3]", want: "RETURN xs[1..3]"},
		{src: "RETURN xs[..3]", want: "RETURN xs[..3]"},
		{src: "RETURN xs[1..]", want: "RETURN xs[1..]"},

		// Functions.
		{src: "RETURN count(*)", want: "RETURN count(*)"},
		{src: "RETURN count(DISTINCT n)", want: "RETURN count(DISTINCT n)"},
		{src: "RETURN toUpper(n.name)", want: "RETURN toUpper(n.name)"},
		{src: "RETURN coalesce(a, b, 0)", want: "RETURN coalesce(a, b, 0)"},

		// CASE.
		{
			src:  "RETURN CASE WHEN n.age > 18 THEN 'adult' ELSE 'minor' END",
			want: "RETURN CASE WHEN (n.age > 18) THEN 'adult' ELSE 'minor' END",
		},
		{
			src:  "RETURN CASE n.x WHEN 1 THEN 'a' WHEN 2 THEN 'b' END",
			want: "RETURN CASE n.x WHEN 1 THEN 'a' WHEN 2 THEN 'b' END",
		},

		// Literals and parameters.
		{src: "RETURN true, false, null", want: "RETURN true, false, null"},
		{src: "RETURN 0x1f, 0o17", want: "RETURN 31, 15"},
		{src: "RETURN 1.5, .5", want: "RETURN 1.5, 0.5"},
		{src: "RETURN $name, $1", want: "RETURN $name, $1"},
		{src: "RETURN {a: 1, b: 'two'}", want: "RETURN {a: 1, b: 'two'}"},
		{src: "RETURN []", want: "RETURN []"},

		// Comments and whitespace are ignored.
		{src: "MATCH (n) // comment\n RETURN n", want: "MATCH (n) RETURN n"},
		{src: "RETURN /* inline */ 1", want: "RETURN 1"},

		// Trailing semicolon.
		{src: "RETURN 1;", want: "RETURN 1"},

		// A larger end-to-end query.
		{
			src: "MATCH (a:Person)-[:KNOWS*1..3]->(b) WHERE a.age > $min " +
				"RETURN DISTINCT b.name AS name ORDER BY name SKIP 5 LIMIT 10",
			want: "MATCH (a:Person)-[:KNOWS*1..3]->(b) WHERE (a.age > $min) " +
				"RETURN DISTINCT b.name AS name ORDER BY name SKIP 5 LIMIT 10",
		},
	}

	for _, c := range cases {
		q, err := Parse(c.src)
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", c.src, err)
			continue
		}
		if got := q.String(); got != c.want {
			t.Errorf("Parse(%q):\n  got  %q\n  want %q", c.src, got, c.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		"",                         // no clauses
		"MATCH",                    // missing pattern
		"MATCH (n",                 // unclosed node
		"MATCH n RETURN n",         // pattern must start with (
		"RETURN",                   // missing projection
		"RETURN 1 +",               // dangling operator
		"RETURN (1",                // unclosed paren
		"MATCH (a)--(b",            // unclosed node after rel
		"RETURN CASE END",          // CASE without WHEN
		"MATCH (n) RETURN n EXTRA", // trailing junk
		"RETURN $",                 // bad parameter
		"WITH",                     // missing projection
		"UNWIND [1] RETURN x",      // UNWIND without AS
	}
	for _, src := range cases {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q): expected error, got none", src)
		}
	}
}
