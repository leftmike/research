// Package cypher implements a hand-written lexer and recursive-descent parser
// for the read-only subset of the openCypher graph query language. It produces
// a typed abstract syntax tree (see ast.go); it does not evaluate or execute
// queries.
//
// The supported subset covers MATCH / OPTIONAL MATCH (with WHERE), UNWIND,
// WITH, and RETURN, including DISTINCT, ORDER BY, SKIP and LIMIT; node and
// relationship patterns (labels, types, variable-length paths, inline
// properties); and the full expression grammar (arithmetic, boolean,
// comparison, string predicates, IN, IS NULL, property/index/slice access,
// function calls, CASE, list and map literals, parameters and literals).
//
// Write clauses (CREATE, MERGE, SET, REMOVE, DELETE, FOREACH), procedure CALL,
// UNION, and comprehensions are intentionally out of scope but the AST and
// parser are structured so they can be added later.
//
// Typical use:
//
//	q, err := cypher.Parse("MATCH (n:Person) WHERE n.age > $min RETURN n.name")
//	if err != nil {
//		// *cypher.ParseError carries the line and column of the failure.
//	}
//	_ = q.Clauses
package cypher
