package cypher

import "testing"

type tok struct {
	tt   TokenType
	text string
}

// lexAll runs the lexer to EOF and returns the token (type, text) pairs,
// excluding the trailing EOF. If an ERROR token is produced it is the last
// element returned.
func lexAll(src string) []tok {
	l := NewLexer(src)
	var toks []tok
	for {
		t := l.Next()
		if t.Type == EOF {
			break
		}
		toks = append(toks, tok{t.Type, t.Text})
		if t.Type == ERROR {
			break
		}
	}
	return toks
}

func TestLexTokens(t *testing.T) {
	cases := []struct {
		src  string
		want []tok
	}{
		{src: "", want: nil},
		{src: "   \t\n  ", want: nil},
		{src: "// a line comment\n", want: nil},
		{src: "/* block\ncomment */", want: nil},

		{src: "MATCH match Match", want: []tok{
			{MATCH, "MATCH"}, {MATCH, "match"}, {MATCH, "Match"},
		}},
		{src: "RETURN n", want: []tok{{RETURN, "RETURN"}, {IDENT, "n"}}},
		{src: "ascending DESCENDING", want: []tok{{ASC, "ascending"}, {DESC, "DESCENDING"}}},
		{src: "foo _bar baz123", want: []tok{
			{IDENT, "foo"}, {IDENT, "_bar"}, {IDENT, "baz123"},
		}},
		{src: "`odd name` `a``b`", want: []tok{{IDENT, "odd name"}, {IDENT, "a`b"}}},

		{src: "0 123 0x1F 0o17", want: []tok{
			{INTEGER, "0"}, {INTEGER, "123"}, {INTEGER, "0x1F"}, {INTEGER, "0o17"},
		}},
		{src: "1.5 .5 1e10 2.5E-3", want: []tok{
			{FLOAT, "1.5"}, {FLOAT, ".5"}, {FLOAT, "1e10"}, {FLOAT, "2.5E-3"},
		}},

		{src: `'hello' "world"`, want: []tok{{STRING, "hello"}, {STRING, "world"}}},
		{src: `'a\nb\t\\\''`, want: []tok{{STRING, "a\nb\t\\'"}}},
		{src: `'A'`, want: []tok{{STRING, "A"}}},

		{src: "$name $1", want: []tok{{PARAMETER, "name"}, {PARAMETER, "1"}}},

		{src: "( ) [ ] { } , . .. : ; |", want: []tok{
			{LPAREN, "("}, {RPAREN, ")"}, {LBRACKET, "["}, {RBRACKET, "]"},
			{LBRACE, "{"}, {RBRACE, "}"}, {COMMA, ","}, {DOT, "."}, {DOTDOT, ".."},
			{COLON, ":"}, {SEMI, ";"}, {PIPE, "|"},
		}},
		{src: "+ - * / % ^ = <> < > <= >=", want: []tok{
			{PLUS, "+"}, {MINUS, "-"}, {STAR, "*"}, {SLASH, "/"}, {PERCENT, "%"},
			{CARET, "^"}, {EQ, "="}, {NE, "<>"}, {LT, "<"}, {GT, ">"},
			{LE, "<="}, {GE, ">="},
		}},

		{src: "a.b", want: []tok{{IDENT, "a"}, {DOT, "."}, {IDENT, "b"}}},
		{src: "[1..5]", want: []tok{
			{LBRACKET, "["}, {INTEGER, "1"}, {DOTDOT, ".."}, {INTEGER, "5"},
			{RBRACKET, "]"},
		}},

		{src: "(a)-[:KNOWS]->(b)", want: []tok{
			{LPAREN, "("}, {IDENT, "a"}, {RPAREN, ")"}, {MINUS, "-"},
			{LBRACKET, "["}, {COLON, ":"}, {IDENT, "KNOWS"}, {RBRACKET, "]"},
			{MINUS, "-"}, {GT, ">"}, {LPAREN, "("}, {IDENT, "b"}, {RPAREN, ")"},
		}},
	}

	for _, c := range cases {
		got := lexAll(c.src)
		if len(got) != len(c.want) {
			t.Errorf("lex(%q): got %d tokens %v, want %d %v",
				c.src, len(got), got, len(c.want), c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("lex(%q) token %d: got %v, want %v", c.src, i, got[i], c.want[i])
			}
		}
	}
}

func TestLexErrors(t *testing.T) {
	cases := []string{
		"'unterminated",
		`"no close`,
		"`no backtick close",
		"0x",
		"1e",
		"$",
		"#",
		`'\q'`,
	}
	for _, src := range cases {
		got := lexAll(src)
		if len(got) == 0 || got[len(got)-1].tt != ERROR {
			t.Errorf("lex(%q): expected an ERROR token, got %v", src, got)
		}
	}
}
