package cypher

import "strings"

// TokenType identifies the lexical category of a Token.
type TokenType int

const (
	EOF TokenType = iota
	ERROR

	// Literals and names.
	IDENT     // identifier (also backtick-quoted)
	INTEGER   // 123, 0x1f, 0o17
	FLOAT     // 1.5, .5, 1e10
	STRING    // 'abc' or "abc"
	PARAMETER // $name or $1

	// Punctuation and operators.
	LPAREN   // (
	RPAREN   // )
	LBRACKET // [
	RBRACKET // ]
	LBRACE   // {
	RBRACE   // }
	COMMA    // ,
	DOT      // .
	DOTDOT   // ..
	COLON    // :
	SEMI     // ;
	PIPE     // |
	PLUS     // +
	MINUS    // -
	STAR     // *
	SLASH    // /
	PERCENT  // %
	CARET    // ^
	EQ       // =
	NE       // <>
	LT       // <
	GT       // >
	LE       // <=
	GE       // >=

	// Keywords. Keep these last so keyword lookup can range over them.
	keywordBeg
	MATCH
	OPTIONAL
	WHERE
	WITH
	UNWIND
	RETURN
	AS
	DISTINCT
	ORDER
	BY
	SKIP
	LIMIT
	AND
	OR
	XOR
	NOT
	IN
	IS
	NULL
	TRUE
	FALSE
	STARTS
	ENDS
	CONTAINS
	CASE
	WHEN
	THEN
	ELSE
	END
	ASC
	DESC
	UNION // reserved for future write/union support
	keywordEnd
)

var tokenNames = map[TokenType]string{
	EOF:       "EOF",
	ERROR:     "ERROR",
	IDENT:     "IDENT",
	INTEGER:   "INTEGER",
	FLOAT:     "FLOAT",
	STRING:    "STRING",
	PARAMETER: "PARAMETER",
	LPAREN:    "(",
	RPAREN:    ")",
	LBRACKET:  "[",
	RBRACKET:  "]",
	LBRACE:    "{",
	RBRACE:    "}",
	COMMA:     ",",
	DOT:       ".",
	DOTDOT:    "..",
	COLON:     ":",
	SEMI:      ";",
	PIPE:      "|",
	PLUS:      "+",
	MINUS:     "-",
	STAR:      "*",
	SLASH:     "/",
	PERCENT:   "%",
	CARET:     "^",
	EQ:        "=",
	NE:        "<>",
	LT:        "<",
	GT:        ">",
	LE:        "<=",
	GE:        ">=",
	MATCH:     "MATCH",
	OPTIONAL:  "OPTIONAL",
	WHERE:     "WHERE",
	WITH:      "WITH",
	UNWIND:    "UNWIND",
	RETURN:    "RETURN",
	AS:        "AS",
	DISTINCT:  "DISTINCT",
	ORDER:     "ORDER",
	BY:        "BY",
	SKIP:      "SKIP",
	LIMIT:     "LIMIT",
	AND:       "AND",
	OR:        "OR",
	XOR:       "XOR",
	NOT:       "NOT",
	IN:        "IN",
	IS:        "IS",
	NULL:      "NULL",
	TRUE:      "TRUE",
	FALSE:     "FALSE",
	STARTS:    "STARTS",
	ENDS:      "ENDS",
	CONTAINS:  "CONTAINS",
	CASE:      "CASE",
	WHEN:      "WHEN",
	THEN:      "THEN",
	ELSE:      "ELSE",
	END:       "END",
	ASC:       "ASC",
	DESC:      "DESC",
	UNION:     "UNION",
}

// String returns a human-readable name for the token type, useful in errors.
func (t TokenType) String() string {
	if s, ok := tokenNames[t]; ok {
		return s
	}
	return "UNKNOWN"
}

// keyword aliases that map to the same token type as their canonical spelling.
var keywordAliases = map[string]TokenType{
	"ASCENDING":  ASC,
	"DESCENDING": DESC,
}

// keywords maps the upper-cased keyword text to its token type. Cypher
// keywords are case-insensitive; identifiers are not.
var keywords = func() map[string]TokenType {
	m := make(map[string]TokenType)
	for tt := keywordBeg + 1; tt < keywordEnd; tt++ {
		m[tokenNames[tt]] = tt
	}
	for alias, tt := range keywordAliases {
		m[alias] = tt
	}
	return m
}()

// lookupKeyword returns the keyword token type for s and whether s is a
// keyword. The comparison is case-insensitive.
func lookupKeyword(s string) (TokenType, bool) {
	tt, ok := keywords[strings.ToUpper(s)]
	return tt, ok
}

// Token is a single lexical token with its source position.
type Token struct {
	Type TokenType
	Text string // literal/identifier text, or error message for ERROR
	Line int    // 1-based
	Col  int    // 1-based, in runes
}
