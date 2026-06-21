# cypher

A hand-written lexer and recursive-descent parser for the **read-only subset**
of the [openCypher](https://opencypher.org/) graph query language, written in
Go with no external dependencies. It produces a typed AST — it does not
evaluate or execute queries.

## Usage

```go
import "github.com/leftmike/research/cypher"

q, err := cypher.Parse("MATCH (n:Person) WHERE n.age > $min RETURN n.name AS name")
if err != nil {
	// err is a *cypher.ParseError with Line and Col.
	log.Fatal(err)
}
fmt.Println(q.String()) // canonical, fully-parenthesized rendering
```

`Parse` returns a `*Query`, an ordered list of `Clause` values. Every AST node
has a `String()`-renderable form via `Query.String()`, which emits canonical
Cypher with compound expressions fully parenthesized (handy for testing and
debugging).

## Supported grammar

- **Clauses:** `MATCH` / `OPTIONAL MATCH` (with `WHERE`), `UNWIND ... AS`,
  `WITH` (with `WHERE`), `RETURN` — including `DISTINCT`, `*`, `ORDER BY`
  (`ASC`/`DESC`), `SKIP`, `LIMIT`, and `AS` aliases.
- **Patterns:** node patterns `(v:Label:Label {props})`; relationships in all
  three directions (`-[]->`, `<-[]-`, `-[]-`); relationship types with `|`
  alternatives; variable-length paths (`*`, `*2`, `*1..3`, `*2..`, `*..5`);
  inline property maps or `$param`; named paths (`p = (a)-->(b)`); and
  comma-separated pattern parts.
- **Expressions:** `OR`/`XOR`/`AND`/`NOT`; comparison chains
  (`= <> < > <= >=`); `STARTS WITH` / `ENDS WITH` / `CONTAINS`; `IN`;
  `IS [NOT] NULL`; arithmetic (`+ - * / %`) and right-associative `^`; unary
  `+`/`-`; postfix property access (`.`), indexing (`[i]`), slicing
  (`[a..b]`), and label checks (`:Label`); function calls (with `DISTINCT` and
  `*`); `CASE` (simple and generic); list and map literals; parameters
  (`$name`, `$1`); and integer/float/string/boolean/null literals.
- **Lexer extras:** case-insensitive keywords, backtick-quoted identifiers,
  `//` line and `/* */` block comments, hex/octal integers, string escapes
  including `\uXXXX`.

## Out of scope (future work)

Write clauses (`CREATE`, `MERGE`, `SET`, `REMOVE`, `DELETE`, `FOREACH`),
procedure `CALL`, `UNION`, list/pattern comprehensions, and any query
evaluation engine.

## Tests

```
go test ./...
```
