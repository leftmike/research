package cypher

import (
	"fmt"
	"strconv"
)

// ParseError describes a syntax error with its source position.
type ParseError struct {
	Msg  string
	Line int
	Col  int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("cypher: %d:%d: %s", e.Line, e.Col, e.Msg)
}

// Parse parses src as a read-only openCypher query and returns its AST.
func Parse(src string) (q *Query, err error) {
	p := &Parser{lex: NewLexer(src)}
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(*ParseError); ok {
				q, err = nil, pe
				return
			}
			panic(r)
		}
	}()
	p.advance()
	p.advance()
	q = p.parseQuery()
	return q, nil
}

// Parser is a hand-written recursive-descent parser with two tokens of
// lookahead (cur and nxt). Errors are reported by panicking with a
// *ParseError that Parse recovers.
type Parser struct {
	lex *Lexer
	cur Token
	nxt Token
}

func (p *Parser) advance() {
	p.cur = p.nxt
	p.nxt = p.lex.Next()
	if p.nxt.Type == ERROR {
		p.failAt(p.nxt, "%s", p.nxt.Text)
	}
}

func (p *Parser) failAt(t Token, format string, args ...any) {
	panic(&ParseError{Msg: fmt.Sprintf(format, args...), Line: t.Line, Col: t.Col})
}

func (p *Parser) fail(format string, args ...any) {
	p.failAt(p.cur, format, args...)
}

func (p *Parser) is(tt TokenType) bool { return p.cur.Type == tt }

func (p *Parser) accept(tt TokenType) bool {
	if p.cur.Type == tt {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) expect(tt TokenType) Token {
	if p.cur.Type != tt {
		p.fail("expected %s, found %s", tt, describe(p.cur))
	}
	t := p.cur
	p.advance()
	return t
}

func (p *Parser) pos() Pos { return Pos{Line: p.cur.Line, Col: p.cur.Col} }

func describe(t Token) string {
	switch t.Type {
	case EOF:
		return "end of input"
	case IDENT, INTEGER, FLOAT, PARAMETER:
		return fmt.Sprintf("%s %q", t.Type, t.Text)
	case STRING:
		return fmt.Sprintf("string %q", t.Text)
	default:
		return t.Type.String()
	}
}

func isKeyword(tt TokenType) bool { return tt > keywordBeg && tt < keywordEnd }

// symbolicName accepts an identifier or any keyword used as a name (label,
// relationship type, property/map key, function name) and returns its text.
func (p *Parser) symbolicName() string {
	if p.cur.Type == IDENT || isKeyword(p.cur.Type) {
		s := p.cur.Text
		p.advance()
		return s
	}
	p.fail("expected a name, found %s", describe(p.cur))
	return ""
}

// --- Query and clauses ---

func (p *Parser) parseQuery() *Query {
	q := &Query{}
	for {
		switch {
		case p.is(MATCH) || p.is(OPTIONAL):
			q.Clauses = append(q.Clauses, p.parseMatch())
		case p.is(UNWIND):
			q.Clauses = append(q.Clauses, p.parseUnwind())
		case p.is(WITH):
			q.Clauses = append(q.Clauses, p.parseWith())
		case p.is(RETURN):
			q.Clauses = append(q.Clauses, p.parseReturn())
		default:
			goto done
		}
	}
done:
	if len(q.Clauses) == 0 {
		p.fail("expected a query clause, found %s", describe(p.cur))
	}
	p.accept(SEMI)
	if !p.is(EOF) {
		p.fail("unexpected %s after query", describe(p.cur))
	}
	return q
}

func (p *Parser) parseMatch() Clause {
	m := &Match{}
	m.Optional = p.accept(OPTIONAL)
	p.expect(MATCH)
	m.Pattern = p.parsePattern()
	if p.accept(WHERE) {
		m.Where = p.parseExpr()
	}
	return m
}

func (p *Parser) parseUnwind() Clause {
	p.expect(UNWIND)
	e := p.parseExpr()
	p.expect(AS)
	v := p.expect(IDENT).Text
	return &Unwind{Expr: e, Variable: v}
}

func (p *Parser) parseWith() Clause {
	p.expect(WITH)
	w := &With{Projection: p.parseProjection()}
	if p.accept(WHERE) {
		w.Where = p.parseExpr()
	}
	return w
}

func (p *Parser) parseReturn() Clause {
	p.expect(RETURN)
	return &Return{Projection: p.parseProjection()}
}

func (p *Parser) parseProjection() Projection {
	var proj Projection
	proj.Distinct = p.accept(DISTINCT)
	if p.is(STAR) {
		proj.Star = true
		p.advance()
	} else {
		proj.Items = append(proj.Items, p.parseProjectionItem())
	}
	for p.accept(COMMA) {
		proj.Items = append(proj.Items, p.parseProjectionItem())
	}
	if p.accept(ORDER) {
		p.expect(BY)
		proj.OrderBy = p.parseSortItems()
	}
	if p.accept(SKIP) {
		proj.Skip = p.parseExpr()
	}
	if p.accept(LIMIT) {
		proj.Limit = p.parseExpr()
	}
	return proj
}

func (p *Parser) parseProjectionItem() ProjectionItem {
	item := ProjectionItem{Expr: p.parseExpr()}
	if p.accept(AS) {
		item.Alias = p.symbolicName()
	}
	return item
}

func (p *Parser) parseSortItems() []SortItem {
	items := []SortItem{p.parseSortItem()}
	for p.accept(COMMA) {
		items = append(items, p.parseSortItem())
	}
	return items
}

func (p *Parser) parseSortItem() SortItem {
	si := SortItem{Expr: p.parseExpr(), Dir: Ascending}
	if p.accept(ASC) {
		si.Dir = Ascending
	} else if p.accept(DESC) {
		si.Dir = Descending
	}
	return si
}

// --- Patterns ---

func (p *Parser) parsePattern() []PatternPart {
	parts := []PatternPart{p.parsePatternPart()}
	for p.accept(COMMA) {
		parts = append(parts, p.parsePatternPart())
	}
	return parts
}

func (p *Parser) parsePatternPart() PatternPart {
	var part PatternPart
	if p.is(IDENT) && p.nxt.Type == EQ {
		part.Variable = p.cur.Text
		p.advance() // variable
		p.advance() // =
	}
	part.Element = p.parsePatternElement()
	return part
}

func (p *Parser) parsePatternElement() PatternElement {
	var el PatternElement
	el.Nodes = append(el.Nodes, p.parseNodePattern())
	for p.is(MINUS) || p.is(LT) {
		el.Rels = append(el.Rels, p.parseRelationshipPattern())
		el.Nodes = append(el.Nodes, p.parseNodePattern())
	}
	return el
}

func (p *Parser) parseNodePattern() NodePattern {
	n := NodePattern{Pos: p.pos()}
	p.expect(LPAREN)
	if p.is(IDENT) {
		n.Variable = p.cur.Text
		p.advance()
	}
	for p.accept(COLON) {
		n.Labels = append(n.Labels, p.symbolicName())
	}
	if p.is(LBRACE) || p.is(PARAMETER) {
		n.Properties = p.parseProperties()
	}
	p.expect(RPAREN)
	return n
}

func (p *Parser) parseRelationshipPattern() RelationshipPattern {
	r := RelationshipPattern{Pos: p.pos()}
	leftArrow := p.accept(LT)
	p.expect(MINUS)
	if p.is(LBRACKET) {
		p.parseRelationshipDetail(&r)
	}
	p.expect(MINUS)
	rightArrow := p.accept(GT)

	switch {
	case leftArrow && rightArrow:
		p.fail("relationship cannot point in both directions")
	case leftArrow:
		r.Direction = RelLeft
	case rightArrow:
		r.Direction = RelRight
	default:
		r.Direction = RelNone
	}
	return r
}

func (p *Parser) parseRelationshipDetail(r *RelationshipPattern) {
	p.expect(LBRACKET)
	if p.is(IDENT) {
		r.Variable = p.cur.Text
		p.advance()
	}
	if p.accept(COLON) {
		r.Types = append(r.Types, p.symbolicName())
		for p.accept(PIPE) {
			p.accept(COLON) // optional ':' before alternative type
			r.Types = append(r.Types, p.symbolicName())
		}
	}
	if p.accept(STAR) {
		r.VarLength = true
		if p.is(INTEGER) {
			n := p.parseIntToken()
			r.MinHops = &n
		}
		if p.accept(DOTDOT) {
			if p.is(INTEGER) {
				n := p.parseIntToken()
				r.MaxHops = &n
			}
		} else if r.MinHops != nil {
			// `*2` means exactly 2 hops.
			n := *r.MinHops
			r.MaxHops = &n
		}
	}
	if p.is(LBRACE) || p.is(PARAMETER) {
		r.Properties = p.parseProperties()
	}
	p.expect(RBRACKET)
}

// parseProperties parses a node/relationship property specification, which is
// either a map literal or a parameter.
func (p *Parser) parseProperties() Expr {
	if p.is(PARAMETER) {
		return p.parseParameter()
	}
	return p.parseMapLiteral()
}

func (p *Parser) parseIntToken() int {
	t := p.expect(INTEGER)
	v, err := strconv.ParseInt(t.Text, 0, 64)
	if err != nil {
		p.failAt(t, "invalid integer %q", t.Text)
	}
	return int(v)
}

// --- Expressions (precedence climbing, low to high) ---

func (p *Parser) parseExpr() Expr { return p.parseOr() }

func (p *Parser) parseOr() Expr {
	left := p.parseXor()
	for p.accept(OR) {
		left = &BinaryExpr{Op: OR, Left: left, Right: p.parseXor()}
	}
	return left
}

func (p *Parser) parseXor() Expr {
	left := p.parseAnd()
	for p.accept(XOR) {
		left = &BinaryExpr{Op: XOR, Left: left, Right: p.parseAnd()}
	}
	return left
}

func (p *Parser) parseAnd() Expr {
	left := p.parseNot()
	for p.accept(AND) {
		left = &BinaryExpr{Op: AND, Left: left, Right: p.parseNot()}
	}
	return left
}

func (p *Parser) parseNot() Expr {
	if p.accept(NOT) {
		return &Not{Operand: p.parseNot()}
	}
	return p.parseComparison()
}

func isComparisonOp(tt TokenType) bool {
	switch tt {
	case EQ, NE, LT, GT, LE, GE:
		return true
	}
	return false
}

func (p *Parser) parseComparison() Expr {
	first := p.parseStringListNull()
	if !isComparisonOp(p.cur.Type) {
		return first
	}
	cmp := &Comparison{Operands: []Expr{first}}
	for isComparisonOp(p.cur.Type) {
		op := p.cur.Type
		p.advance()
		cmp.Ops = append(cmp.Ops, op)
		cmp.Operands = append(cmp.Operands, p.parseStringListNull())
	}
	return cmp
}

func (p *Parser) parseStringListNull() Expr {
	left := p.parseAddSub()
	for {
		switch {
		case p.is(STARTS) && p.nxt.Type == WITH:
			p.advance()
			p.advance()
			left = &StringOp{Kind: StartsWith, Left: left, Right: p.parseAddSub()}
		case p.is(ENDS) && p.nxt.Type == WITH:
			p.advance()
			p.advance()
			left = &StringOp{Kind: EndsWith, Left: left, Right: p.parseAddSub()}
		case p.is(CONTAINS):
			p.advance()
			left = &StringOp{Kind: Contains, Left: left, Right: p.parseAddSub()}
		case p.is(IN):
			p.advance()
			left = &In{Left: left, Right: p.parseAddSub()}
		case p.is(IS):
			p.advance()
			neg := p.accept(NOT)
			p.expect(NULL)
			left = &IsNull{Operand: left, Negated: neg}
		default:
			return left
		}
	}
}

func (p *Parser) parseAddSub() Expr {
	left := p.parseMulDiv()
	for p.is(PLUS) || p.is(MINUS) {
		op := p.cur.Type
		p.advance()
		left = &BinaryExpr{Op: op, Left: left, Right: p.parseMulDiv()}
	}
	return left
}

func (p *Parser) parseMulDiv() Expr {
	left := p.parsePow()
	for p.is(STAR) || p.is(SLASH) || p.is(PERCENT) {
		op := p.cur.Type
		p.advance()
		left = &BinaryExpr{Op: op, Left: left, Right: p.parsePow()}
	}
	return left
}

func (p *Parser) parsePow() Expr {
	left := p.parseUnary()
	if p.accept(CARET) {
		// Right associative.
		return &BinaryExpr{Op: CARET, Left: left, Right: p.parsePow()}
	}
	return left
}

func (p *Parser) parseUnary() Expr {
	if p.is(PLUS) || p.is(MINUS) {
		op := p.cur.Type
		p.advance()
		return &UnaryExpr{Op: op, Operand: p.parseUnary()}
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() Expr {
	e := p.parseAtom()
	for {
		switch {
		case p.accept(DOT):
			e = &PropertyAccess{Operand: e, Name: p.symbolicName()}
		case p.is(LBRACKET):
			e = p.parseIndexOrSlice(e)
		case p.is(COLON):
			lc := &LabelsCheck{Operand: e}
			for p.accept(COLON) {
				lc.Labels = append(lc.Labels, p.symbolicName())
			}
			e = lc
		default:
			return e
		}
	}
}

func (p *Parser) parseIndexOrSlice(operand Expr) Expr {
	p.expect(LBRACKET)
	if p.accept(DOTDOT) { // [..high]
		s := &Slice{Operand: operand}
		if !p.is(RBRACKET) {
			s.High = p.parseExpr()
		}
		p.expect(RBRACKET)
		return s
	}
	idx := p.parseExpr()
	if p.accept(DOTDOT) { // [low..high] or [low..]
		s := &Slice{Operand: operand, Low: idx}
		if !p.is(RBRACKET) {
			s.High = p.parseExpr()
		}
		p.expect(RBRACKET)
		return s
	}
	p.expect(RBRACKET)
	return &Index{Operand: operand, Index: idx}
}

func (p *Parser) parseAtom() Expr {
	t := p.cur
	switch t.Type {
	case INTEGER:
		v, err := strconv.ParseInt(t.Text, 0, 64)
		if err != nil {
			p.fail("invalid integer %q", t.Text)
		}
		p.advance()
		return &Literal{Kind: IntLit, Int: v}
	case FLOAT:
		v, err := strconv.ParseFloat(t.Text, 64)
		if err != nil {
			p.fail("invalid float %q", t.Text)
		}
		p.advance()
		return &Literal{Kind: FloatLit, Float: v}
	case STRING:
		p.advance()
		return &Literal{Kind: StringLit, Str: t.Text}
	case TRUE:
		p.advance()
		return &Literal{Kind: BoolLit, Bool: true}
	case FALSE:
		p.advance()
		return &Literal{Kind: BoolLit, Bool: false}
	case NULL:
		p.advance()
		return &Literal{Kind: NullLit}
	case PARAMETER:
		return p.parseParameter()
	case LPAREN:
		p.advance()
		e := p.parseExpr()
		p.expect(RPAREN)
		return e
	case LBRACKET:
		return p.parseListLiteral()
	case LBRACE:
		return p.parseMapLiteral()
	case CASE:
		return p.parseCase()
	case IDENT:
		name := t.Text
		pos := p.pos()
		p.advance()
		if p.is(LPAREN) {
			return p.parseFunctionCall(name)
		}
		return &Variable{Pos: pos, Name: name}
	}
	p.fail("unexpected %s in expression", describe(t))
	return nil
}

func (p *Parser) parseParameter() Expr {
	t := p.expect(PARAMETER)
	return &Parameter{Name: t.Text}
}

func (p *Parser) parseFunctionCall(name string) Expr {
	p.expect(LPAREN)
	fc := &FunctionCall{Name: name}
	fc.Distinct = p.accept(DISTINCT)
	if p.accept(STAR) {
		fc.Star = true
	} else if !p.is(RPAREN) {
		fc.Args = append(fc.Args, p.parseExpr())
		for p.accept(COMMA) {
			fc.Args = append(fc.Args, p.parseExpr())
		}
	}
	p.expect(RPAREN)
	return fc
}

func (p *Parser) parseListLiteral() Expr {
	p.expect(LBRACKET)
	list := &ListLiteral{}
	if !p.is(RBRACKET) {
		list.Elements = append(list.Elements, p.parseExpr())
		for p.accept(COMMA) {
			list.Elements = append(list.Elements, p.parseExpr())
		}
	}
	p.expect(RBRACKET)
	return list
}

func (p *Parser) parseMapLiteral() *MapLiteral {
	p.expect(LBRACE)
	m := &MapLiteral{}
	if !p.is(RBRACE) {
		for {
			m.Keys = append(m.Keys, p.symbolicName())
			p.expect(COLON)
			m.Values = append(m.Values, p.parseExpr())
			if !p.accept(COMMA) {
				break
			}
		}
	}
	p.expect(RBRACE)
	return m
}

func (p *Parser) parseCase() Expr {
	p.expect(CASE)
	c := &CaseExpr{}
	if !p.is(WHEN) {
		c.Subject = p.parseExpr()
	}
	for p.accept(WHEN) {
		w := CaseWhen{When: p.parseExpr()}
		p.expect(THEN)
		w.Then = p.parseExpr()
		c.Whens = append(c.Whens, w)
	}
	if len(c.Whens) == 0 {
		p.fail("CASE requires at least one WHEN")
	}
	if p.accept(ELSE) {
		c.Else = p.parseExpr()
	}
	p.expect(END)
	return c
}
