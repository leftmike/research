package cypher

// This file defines the abstract syntax tree produced by the parser for the
// read-only subset of openCypher. Write clauses (CREATE/MERGE/SET/...) are
// intentionally omitted; the node set is structured so they can be added
// later without reworking the core.

// Pos is the 1-based source position of a node. It is embedded in nodes that
// want to report where they began.
type Pos struct {
	Line int
	Col  int
}

// Query is a complete parsed query: an ordered sequence of clauses.
type Query struct {
	Clauses []Clause
}

// Clause is a top-level reading clause (MATCH, UNWIND, WITH, RETURN).
type Clause interface {
	clause()
}

// Match is a [OPTIONAL] MATCH clause with an optional WHERE predicate.
type Match struct {
	Optional bool
	Pattern  []PatternPart
	Where    Expr // nil if absent
}

// Unwind is `UNWIND <expr> AS <variable>`.
type Unwind struct {
	Expr     Expr
	Variable string
}

// With is a `WITH` projection that pipes results to following clauses. It may
// carry its own WHERE predicate.
type With struct {
	Projection
	Where Expr // nil if absent
}

// Return is a terminal `RETURN` projection.
type Return struct {
	Projection
}

func (*Match) clause()  {}
func (*Unwind) clause() {}
func (*With) clause()   {}
func (*Return) clause() {}

// Projection is the shared shape of WITH and RETURN: projected items plus the
// optional ORDER BY / SKIP / LIMIT modifiers.
type Projection struct {
	Distinct bool
	Star     bool // RETURN * / WITH *
	Items    []ProjectionItem
	OrderBy  []SortItem
	Skip     Expr // nil if absent
	Limit    Expr // nil if absent
}

// ProjectionItem is a single projected expression with an optional alias.
type ProjectionItem struct {
	Expr  Expr
	Alias string // "" if no AS alias
}

// SortDir is the direction of an ORDER BY item.
type SortDir int

const (
	Ascending SortDir = iota
	Descending
)

// SortItem is one ORDER BY entry.
type SortItem struct {
	Expr Expr
	Dir  SortDir
}

// PatternPart is one comma-separated element of a MATCH pattern, optionally
// bound to a path variable (`p = (a)-->(b)`).
type PatternPart struct {
	Variable string // "" if anonymous
	Element  PatternElement
}

// PatternElement is a chain of nodes connected by relationships. Nodes has
// len(Rels)+1 entries; Rels[i] connects Nodes[i] to Nodes[i+1].
type PatternElement struct {
	Nodes []NodePattern
	Rels  []RelationshipPattern
}

// NodePattern is `( [var] [:Label...] [ {props} ] )`.
type NodePattern struct {
	Pos
	Variable   string
	Labels     []string
	Properties Expr // *MapLiteral or *Parameter, nil if absent
}

// RelDirection is the arrow direction of a relationship pattern.
type RelDirection int

const (
	// RelNone is `-[...]-` (undirected).
	RelNone RelDirection = iota
	// RelRight is `-[...]->`.
	RelRight
	// RelLeft is `<-[...]-`.
	RelLeft
)

// RelationshipPattern is the relationship between two node patterns, including
// optional variable-length specification (`*`, `*2`, `*1..3`, `*..3`, `*2..`).
type RelationshipPattern struct {
	Pos
	Direction  RelDirection
	Variable   string   // "" if anonymous
	Types      []string // alternative rel types joined by '|'
	VarLength  bool     // true if a `*` was present
	MinHops    *int     // nil means unbounded/default
	MaxHops    *int     // nil means unbounded
	Properties Expr     // *MapLiteral or *Parameter, nil if absent
}

// Expr is any expression node.
type Expr interface {
	expr()
}

// BinaryExpr is an arithmetic or logical binary operation. Op is the operator
// token type (e.g. PLUS, AND, OR, CARET).
type BinaryExpr struct {
	Op    TokenType
	Left  Expr
	Right Expr
}

// Comparison is a chained comparison such as `a < b <= c`. Ops has one entry
// per operator and Operands has len(Ops)+1 entries.
type Comparison struct {
	Operands []Expr
	Ops      []TokenType // each of EQ, NE, LT, GT, LE, GE
}

// UnaryExpr is a prefix `+` or `-`.
type UnaryExpr struct {
	Op      TokenType // PLUS or MINUS
	Operand Expr
}

// Not is logical negation.
type Not struct {
	Operand Expr
}

// StringOpKind distinguishes the string predicates.
type StringOpKind int

const (
	StartsWith StringOpKind = iota
	EndsWith
	Contains
)

// StringOp is `<expr> STARTS WITH|ENDS WITH|CONTAINS <expr>`.
type StringOp struct {
	Kind  StringOpKind
	Left  Expr
	Right Expr
}

// In is `<expr> IN <expr>`.
type In struct {
	Left  Expr
	Right Expr
}

// IsNull is `<expr> IS NULL` or `<expr> IS NOT NULL` when Negated is true.
type IsNull struct {
	Operand Expr
	Negated bool
}

// PropertyAccess is `<expr>.<name>`.
type PropertyAccess struct {
	Operand Expr
	Name    string
}

// Index is `<expr>[<index>]`.
type Index struct {
	Operand Expr
	Index   Expr
}

// Slice is `<expr>[<low>..<high>]`; Low or High may be nil.
type Slice struct {
	Operand Expr
	Low     Expr
	High    Expr
}

// LabelsCheck is `<expr>:Label[:Label...]` used as a boolean predicate.
type LabelsCheck struct {
	Operand Expr
	Labels  []string
}

// FunctionCall is `name(args...)`, with Distinct for `count(DISTINCT x)` and
// Star for `count(*)`.
type FunctionCall struct {
	Name     string
	Distinct bool
	Star     bool
	Args     []Expr
}

// CaseExpr covers both the simple form (`CASE x WHEN ...`) and the generic
// form (`CASE WHEN cond ...`). For the simple form Subject is non-nil.
type CaseExpr struct {
	Subject Expr // nil for the generic form
	Whens   []CaseWhen
	Else    Expr // nil if absent
}

// CaseWhen is one `WHEN <when> THEN <then>` branch.
type CaseWhen struct {
	When Expr
	Then Expr
}

// ListLiteral is `[e1, e2, ...]`.
type ListLiteral struct {
	Elements []Expr
}

// MapLiteral is `{ k1: v1, k2: v2 }`.
type MapLiteral struct {
	Keys   []string
	Values []Expr
}

// Variable is a reference to a bound name.
type Variable struct {
	Pos
	Name string
}

// Parameter is `$name` or `$1`.
type Parameter struct {
	Name string
}

// LiteralKind classifies a Literal's value.
type LiteralKind int

const (
	IntLit LiteralKind = iota
	FloatLit
	StringLit
	BoolLit
	NullLit
)

// Literal is a constant value. The relevant typed field is set per Kind:
// Int/Float/Str/Bool. NullLit uses no value field.
type Literal struct {
	Kind  LiteralKind
	Int   int64
	Float float64
	Str   string
	Bool  bool
}

func (*BinaryExpr) expr()     {}
func (*Comparison) expr()     {}
func (*UnaryExpr) expr()      {}
func (*Not) expr()            {}
func (*StringOp) expr()       {}
func (*In) expr()             {}
func (*IsNull) expr()         {}
func (*PropertyAccess) expr() {}
func (*Index) expr()          {}
func (*Slice) expr()          {}
func (*LabelsCheck) expr()    {}
func (*FunctionCall) expr()   {}
func (*CaseExpr) expr()       {}
func (*ListLiteral) expr()    {}
func (*MapLiteral) expr()     {}
func (*Variable) expr()       {}
func (*Parameter) expr()      {}
func (*Literal) expr()        {}
