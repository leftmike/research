package cypher

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer scans Cypher source text into a stream of Tokens.
type Lexer struct {
	src  string
	pos  int // byte offset of the next rune to read
	line int // 1-based line of pos
	col  int // 1-based column (in runes) of pos
}

// NewLexer returns a Lexer over src.
func NewLexer(src string) *Lexer {
	return &Lexer{src: src, pos: 0, line: 1, col: 1}
}

const eofRune = rune(-1)

// peek returns the rune at the current position without consuming it.
func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return eofRune
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos:])
	return r
}

// peek2 returns the rune one past the current position without consuming.
func (l *Lexer) peek2() rune {
	if l.pos >= len(l.src) {
		return eofRune
	}
	_, w := utf8.DecodeRuneInString(l.src[l.pos:])
	if l.pos+w >= len(l.src) {
		return eofRune
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos+w:])
	return r
}

// advance consumes and returns the current rune, tracking line/column.
func (l *Lexer) advance() rune {
	if l.pos >= len(l.src) {
		return eofRune
	}
	r, w := utf8.DecodeRuneInString(l.src[l.pos:])
	l.pos += w
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

func (l *Lexer) tok(tt TokenType, text string, line, col int) Token {
	return Token{Type: tt, Text: text, Line: line, Col: col}
}

func (l *Lexer) errorf(line, col int, format string, args ...any) Token {
	return Token{Type: ERROR, Text: fmt.Sprintf(format, args...), Line: line, Col: col}
}

// Next returns the next token, skipping whitespace and comments.
func (l *Lexer) Next() Token {
	l.skipTrivia()

	line, col := l.line, l.col
	r := l.peek()
	switch {
	case r == eofRune:
		return l.tok(EOF, "", line, col)
	case isIdentStart(r):
		return l.lexIdent(line, col)
	case r == '`':
		return l.lexBacktick(line, col)
	case unicode.IsDigit(r) || (r == '.' && unicode.IsDigit(l.peek2())):
		return l.lexNumber(line, col)
	case r == '\'' || r == '"':
		return l.lexString(line, col)
	case r == '$':
		return l.lexParameter(line, col)
	}
	return l.lexOperator(line, col)
}

// skipTrivia consumes whitespace, line comments and block comments.
func (l *Lexer) skipTrivia() {
	for {
		r := l.peek()
		switch {
		case r == ' ' || r == '\t' || r == '\r' || r == '\n':
			l.advance()
		case r == '/' && l.peek2() == '/':
			for l.peek() != '\n' && l.peek() != eofRune {
				l.advance()
			}
		case r == '/' && l.peek2() == '*':
			l.advance() // /
			l.advance() // *
			for {
				if l.peek() == eofRune {
					return
				}
				if l.peek() == '*' && l.peek2() == '/' {
					l.advance()
					l.advance()
					break
				}
				l.advance()
			}
		default:
			return
		}
	}
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (l *Lexer) lexIdent(line, col int) Token {
	start := l.pos
	for isIdentPart(l.peek()) {
		l.advance()
	}
	text := l.src[start:l.pos]
	if tt, ok := lookupKeyword(text); ok {
		return l.tok(tt, text, line, col)
	}
	return l.tok(IDENT, text, line, col)
}

// lexBacktick scans a backtick-quoted identifier. A doubled backtick is an
// escaped backtick within the identifier.
func (l *Lexer) lexBacktick(line, col int) Token {
	l.advance() // opening `
	var sb strings.Builder
	for {
		r := l.peek()
		switch r {
		case eofRune:
			return l.errorf(line, col, "unterminated backtick identifier")
		case '`':
			l.advance()
			if l.peek() == '`' { // escaped backtick
				l.advance()
				sb.WriteRune('`')
				continue
			}
			return l.tok(IDENT, sb.String(), line, col)
		default:
			sb.WriteRune(r)
			l.advance()
		}
	}
}

func (l *Lexer) lexNumber(line, col int) Token {
	start := l.pos
	isFloat := false

	if l.peek() == '0' && (l.peek2() == 'x' || l.peek2() == 'X') {
		l.advance() // 0
		l.advance() // x
		if !isHexDigit(l.peek()) {
			return l.errorf(line, col, "malformed hexadecimal literal")
		}
		for isHexDigit(l.peek()) {
			l.advance()
		}
		return l.tok(INTEGER, l.src[start:l.pos], line, col)
	}
	if l.peek() == '0' && (l.peek2() == 'o' || l.peek2() == 'O') {
		l.advance() // 0
		l.advance() // o
		if !isOctalDigit(l.peek()) {
			return l.errorf(line, col, "malformed octal literal")
		}
		for isOctalDigit(l.peek()) {
			l.advance()
		}
		return l.tok(INTEGER, l.src[start:l.pos], line, col)
	}

	for unicode.IsDigit(l.peek()) {
		l.advance()
	}
	if l.peek() == '.' && l.peek2() != '.' { // not the .. range operator
		isFloat = true
		l.advance()
		for unicode.IsDigit(l.peek()) {
			l.advance()
		}
	}
	if l.peek() == 'e' || l.peek() == 'E' {
		isFloat = true
		l.advance()
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}
		if !unicode.IsDigit(l.peek()) {
			return l.errorf(line, col, "malformed number: missing exponent digits")
		}
		for unicode.IsDigit(l.peek()) {
			l.advance()
		}
	}
	if isFloat {
		return l.tok(FLOAT, l.src[start:l.pos], line, col)
	}
	return l.tok(INTEGER, l.src[start:l.pos], line, col)
}

func isHexDigit(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isOctalDigit(r rune) bool {
	return r >= '0' && r <= '7'
}

// lexString scans a single- or double-quoted string with backslash escapes.
// The returned token Text holds the decoded (unescaped) string value.
func (l *Lexer) lexString(line, col int) Token {
	quote := l.advance()
	var sb strings.Builder
	for {
		r := l.peek()
		switch r {
		case eofRune, '\n':
			return l.errorf(line, col, "unterminated string literal")
		case quote:
			l.advance()
			return l.tok(STRING, sb.String(), line, col)
		case '\\':
			l.advance()
			esc := l.peek()
			switch esc {
			case eofRune:
				return l.errorf(line, col, "unterminated string literal")
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case 'b':
				sb.WriteRune('\b')
			case 'f':
				sb.WriteRune('\f')
			case '0':
				sb.WriteRune(0)
			case '\\', '\'', '"', '`':
				sb.WriteRune(esc)
			case 'u':
				l.advance() // consume u; read 4 hex digits below
				v, ok := l.readHex(4)
				if !ok {
					return l.errorf(line, col, "malformed \\u escape in string")
				}
				sb.WriteRune(rune(v))
				continue
			case 'U':
				l.advance() // consume U; read 8 hex digits below
				v, ok := l.readHex(8)
				if !ok {
					return l.errorf(line, col, "malformed \\U escape in string")
				}
				sb.WriteRune(rune(v))
				continue
			default:
				return l.errorf(line, col, "invalid escape sequence \\%c", esc)
			}
			l.advance()
		default:
			sb.WriteRune(r)
			l.advance()
		}
	}
}

// readHex reads exactly n hex digits and returns their value.
func (l *Lexer) readHex(n int) (int, bool) {
	v := 0
	for i := 0; i < n; i++ {
		r := l.peek()
		if !isHexDigit(r) {
			return 0, false
		}
		v = v*16 + hexVal(r)
		l.advance()
	}
	return v, true
}

func hexVal(r rune) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10
	default:
		return int(r-'A') + 10
	}
}

// lexParameter scans $name or $123.
func (l *Lexer) lexParameter(line, col int) Token {
	l.advance() // $
	start := l.pos
	if isIdentStart(l.peek()) {
		for isIdentPart(l.peek()) {
			l.advance()
		}
	} else if unicode.IsDigit(l.peek()) {
		for unicode.IsDigit(l.peek()) {
			l.advance()
		}
	} else {
		return l.errorf(line, col, "expected name or number after '$'")
	}
	return l.tok(PARAMETER, l.src[start:l.pos], line, col)
}

// lexOperator scans punctuation and operators, preferring the longest match.
func (l *Lexer) lexOperator(line, col int) Token {
	r := l.advance()
	switch r {
	case '(':
		return l.tok(LPAREN, "(", line, col)
	case ')':
		return l.tok(RPAREN, ")", line, col)
	case '[':
		return l.tok(LBRACKET, "[", line, col)
	case ']':
		return l.tok(RBRACKET, "]", line, col)
	case '{':
		return l.tok(LBRACE, "{", line, col)
	case '}':
		return l.tok(RBRACE, "}", line, col)
	case ',':
		return l.tok(COMMA, ",", line, col)
	case '.':
		if l.peek() == '.' {
			l.advance()
			return l.tok(DOTDOT, "..", line, col)
		}
		return l.tok(DOT, ".", line, col)
	case ':':
		return l.tok(COLON, ":", line, col)
	case ';':
		return l.tok(SEMI, ";", line, col)
	case '|':
		return l.tok(PIPE, "|", line, col)
	case '+':
		return l.tok(PLUS, "+", line, col)
	case '-':
		return l.tok(MINUS, "-", line, col)
	case '*':
		return l.tok(STAR, "*", line, col)
	case '/':
		return l.tok(SLASH, "/", line, col)
	case '%':
		return l.tok(PERCENT, "%", line, col)
	case '^':
		return l.tok(CARET, "^", line, col)
	case '=':
		return l.tok(EQ, "=", line, col)
	case '<':
		if l.peek() == '>' {
			l.advance()
			return l.tok(NE, "<>", line, col)
		}
		if l.peek() == '=' {
			l.advance()
			return l.tok(LE, "<=", line, col)
		}
		return l.tok(LT, "<", line, col)
	case '>':
		if l.peek() == '=' {
			l.advance()
			return l.tok(GE, ">=", line, col)
		}
		return l.tok(GT, ">", line, col)
	}
	return l.errorf(line, col, "unexpected character %q", r)
}
