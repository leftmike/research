// Package graph is a simple in-memory graph database that executes the
// read-only openCypher subset parsed by github.com/leftmike/research/cypher.
//
// A Graph holds nodes and relationships in memory. Data is loaded through the
// Go API (AddNode, AddRelationship) because the parser does not support write
// clauses. Queries are executed by parsing source into a *cypher.Query and
// evaluating it against the stored graph.
//
// The executor supports the common read path: MATCH over node and relationship
// patterns (labels, types, direction, inline properties) with an optional
// WHERE predicate, followed by a RETURN projection with DISTINCT, ORDER BY,
// SKIP and LIMIT. Variable-length paths, OPTIONAL MATCH, UNWIND and WITH
// pipelines are not yet evaluated and return a clear "unsupported" error.
//
// Typical use:
//
//	g := graph.New()
//	alice := g.AddNode([]string{"Person"}, map[string]graph.Value{"name": "Alice", "age": int64(34)})
//	bob := g.AddNode([]string{"Person"}, map[string]graph.Value{"name": "Bob", "age": int64(27)})
//	g.AddRelationship("KNOWS", alice, bob, nil)
//
//	res, err := g.Query("MATCH (a:Person)-[:KNOWS]->(b) RETURN a.name, b.name")
//	if err != nil {
//		// handle parse or execution error
//	}
//	fmt.Print(res)
package graph
