package main

import (
	"strconv"
	"strings"
)

// charWidth returns the terminal column count of a rune: 0 for control and
// combining marks, 2 for East Asian wide / fullwidth / emoji, else 1.
func charWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if r < 0x20 || (r >= 0x7f && r < 0xa0) { // control
		return 0
	}
	if isCombining(r) {
		return 0
	}
	if isWide(r) {
		return 2
	}
	return 1
}

func isCombining(r rune) bool {
	switch {
	case r >= 0x0300 && r <= 0x036f, // combining diacritical marks
		r >= 0x0483 && r <= 0x0489,
		r >= 0x0591 && r <= 0x05bd,
		r >= 0x0610 && r <= 0x061a,
		r >= 0x064b && r <= 0x065f,
		r >= 0x1ab0 && r <= 0x1aff,
		r >= 0x1dc0 && r <= 0x1dff,
		r >= 0x200b && r <= 0x200f, // zero-width space/joiners/marks
		r >= 0x20d0 && r <= 0x20ff,
		r >= 0xfe00 && r <= 0xfe0f, // variation selectors
		r >= 0xfe20 && r <= 0xfe2f:
		return true
	}
	return false
}

func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115f, // Hangul Jamo
		r == 0x2329 || r == 0x232a,
		r >= 0x2e80 && r <= 0x303e, // CJK radicals, Kangxi, symbols
		r >= 0x3041 && r <= 0x33ff, // Hiragana..CJK compatibility
		r >= 0x3400 && r <= 0x4dbf, // CJK ext A
		r >= 0x4e00 && r <= 0x9fff, // CJK unified
		r >= 0xa000 && r <= 0xa4cf, // Yi
		r >= 0xac00 && r <= 0xd7a3, // Hangul syllables
		r >= 0xf900 && r <= 0xfaff, // CJK compatibility ideographs
		r >= 0xfe10 && r <= 0xfe19,
		r >= 0xfe30 && r <= 0xfe6f,
		r >= 0xff00 && r <= 0xff60, // fullwidth forms
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x1f300 && r <= 0x1faff, // emoji and symbols
		r >= 0x20000 && r <= 0x3fffd: // CJK ext B+
		return true
	}
	return false
}

// strWidth is the total display width of s.
func strWidth(s string) int {
	w := 0
	for _, r := range s {
		w += charWidth(r)
	}
	return w
}

// clean_label: strip HTML markup, quotes, and markdown, then decode entities.
func cleanLabel(raw string) string {
	stripped := stripHTMLTags(strings.TrimSpace(raw))
	trimmed := strings.TrimSpace(stripped)
	unquoted := trimmed
	if u, ok := stripPair(trimmed, '"'); ok {
		unquoted = u
	} else if u, ok := stripPair(trimmed, '\''); ok {
		unquoted = u
	}
	unquoted = strings.TrimSpace(unquoted)
	text := unquoted
	if md, ok := stripPair(unquoted, '`'); ok {
		text = stripMarkdown(strings.TrimSpace(md))
	}
	return decodeHTMLEntities(text)
}

// stripPair removes a matching leading/trailing c, reporting whether both were present.
func stripPair(s string, c byte) (string, bool) {
	if len(s) >= 2 && s[0] == c && s[len(s)-1] == c {
		return s[1 : len(s)-1], true
	}
	return s, false
}

// decodeHTMLEntities decodes named and numeric entities in a single left-to-right
// pass, so already-decoded output is never re-scanned.
func decodeHTMLEntities(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s
	}
	chars := []rune(s)
	var out strings.Builder
	i := 0
	for i < len(chars) {
		if chars[i] != '&' {
			out.WriteRune(chars[i])
			i++
			continue
		}
		hi := i + 1 + entityLookahd
		if hi > len(chars) {
			hi = len(chars)
		}
		semi := -1
		for j := i + 1; j < hi; j++ {
			if chars[j] == ';' {
				semi = j
				break
			}
		}
		decoded := rune(-1)
		if semi >= 0 {
			if c, ok := decodeEntityBody(string(chars[i+1 : semi])); ok {
				decoded = c
			}
		}
		if decoded >= 0 {
			out.WriteRune(decoded)
			i = semi + 1
		} else {
			out.WriteByte('&')
			i++
		}
	}
	return out.String()
}

func decodeEntityBody(body string) (rune, bool) {
	switch body {
	case "lt":
		return '<', true
	case "gt":
		return '>', true
	case "amp":
		return '&', true
	case "quot":
		return '"', true
	case "apos":
		return '\'', true
	}
	num, ok := strings.CutPrefix(body, "#")
	if !ok {
		return 0, false
	}
	var code int64
	var err error
	if hex, ok := cutPrefixAny(num, "xX"); ok {
		code, err = strconv.ParseInt(hex, 16, 64)
	} else {
		code, err = strconv.ParseInt(num, 10, 64)
	}
	if err != nil {
		return 0, false
	}
	r := rune(code)
	// Reject control chars: NUL collides with the cont sentinel and ESC would
	// inject ANSI into scrollback.
	if r == 0 || (r < 0x20) || (r >= 0x7f && r < 0xa0) || r > 0x10ffff {
		return 0, false
	}
	return r, true
}

func stripMarkdown(s string) string {
	noCode := strings.ReplaceAll(s, "`", "")
	noStrong := strings.ReplaceAll(strings.ReplaceAll(noCode, "**", ""), "__", "")
	chars := []rune(noStrong)
	var out strings.Builder
	for i, c := range chars {
		if c == '*' || c == '_' {
			innerWord := i > 0 && isAlnum(chars[i-1]) && i+1 < len(chars) && isAlnum(chars[i+1])
			if !innerWord {
				continue
			}
		}
		out.WriteRune(c)
	}
	return strings.TrimSpace(out.String())
}

var htmlFormatTags = map[string]bool{
	"b": true, "strong": true, "i": true, "em": true, "u": true, "s": true,
	"strike": true, "del": true, "ins": true, "mark": true, "small": true,
	"big": true, "sub": true, "sup": true, "code": true, "kbd": true,
	"samp": true, "var": true, "tt": true, "span": true, "font": true,
	"q": true, "abbr": true, "cite": true, "pre": true,
}

func stripHTMLTags(s string) string {
	chars := []rune(s)
	var out strings.Builder
	i := 0
	for i < len(chars) {
		if chars[i] == '<' {
			if name, end, ok := htmlTagAt(chars, i); ok {
				lower := strings.ToLower(name)
				if lower == "br" {
					out.WriteByte(' ')
					i = end
					continue
				}
				if htmlFormatTags[lower] {
					i = end
					continue
				}
			}
		}
		out.WriteRune(chars[i])
		i++
	}
	return out.String()
}

func htmlTagAt(chars []rune, start int) (string, int, bool) {
	i := start + 1
	if i < len(chars) && chars[i] == '/' {
		i++
	}
	nameStart := i
	for i < len(chars) && isASCIIAlnum(chars[i]) {
		i++
	}
	if i == nameStart {
		return "", 0, false
	}
	name := string(chars[nameStart:i])
	for i < len(chars) && chars[i] != '>' {
		if chars[i] == '<' {
			return "", 0, false
		}
		i++
	}
	if i < len(chars) && chars[i] == '>' {
		return name, i + 1, true
	}
	return "", 0, false
}

// wrapLabel wraps label to at most maxLn lines of width columns, breaking long
// words at identifier boundaries where possible and truncating overflow with an
// ellipsis.
func wrapLabel(label string, width, maxLn int) []string {
	if width < 1 {
		width = 1
	}
	cw := func(c rune) int {
		if w := charWidth(c); w > 0 {
			return w
		}
		return 1
	}
	var lines []string
	var cur []rune
	curW := 0
	for _, word := range strings.Fields(label) {
		ww := strWidth(word)
		wr := []rune(word)
		switch {
		case ww > width:
			if len(cur) > 0 {
				lines = append(lines, string(cur))
				cur, curW = nil, 0
			}
			var chunk []rune
			chunkW := 0
			for _, ch := range wr {
				c := cw(ch)
				if chunkW+c > width && len(chunk) > 0 {
					var carry []rune
					if p := lastIndexAnyRune(chunk, labelBreakChars); p >= 0 {
						carry = append(carry, chunk[p+1:]...)
						chunk = chunk[:p+1]
					}
					lines = append(lines, string(chunk))
					chunkW = 0
					for _, cc := range carry {
						chunkW += cw(cc)
					}
					chunk = carry
				}
				chunk = append(chunk, ch)
				chunkW += c
			}
			cur, curW = chunk, chunkW
		case len(cur) == 0:
			cur = append(cur, wr...)
			curW = ww
		case curW+1+ww <= width:
			cur = append(cur, ' ')
			cur = append(cur, wr...)
			curW += 1 + ww
		default:
			lines = append(lines, string(cur))
			cur = append(cur[:0:0], wr...)
			curW = ww
		}
	}
	if len(cur) > 0 {
		lines = append(lines, string(cur))
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	if len(lines) > maxLn {
		lines = lines[:maxLn]
		target := width - 1
		if target < 1 {
			target = 1
		}
		var s []rune
		sw := 0
		for _, ch := range []rune(lines[len(lines)-1]) {
			c := cw(ch)
			if sw+c > target {
				break
			}
			s = append(s, ch)
			sw += c
		}
		s = append(s, '…')
		lines[len(lines)-1] = string(s)
	}
	return lines
}

// fitLabel truncates label to fit inner columns, appending an ellipsis.
func fitLabel(label string, inner int) string {
	if strWidth(label) <= inner {
		return label
	}
	var out strings.Builder
	used := 0
	for _, c := range label {
		cw := charWidth(c)
		if used+cw+1 > inner {
			break
		}
		out.WriteRune(c)
		used += cw
	}
	out.WriteRune('…')
	return out.String()
}
