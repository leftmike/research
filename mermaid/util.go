package main

import (
	"strconv"
	"unicode"
)

func itoa(n int) string { return strconv.Itoa(n) }

func isAlnum(r rune) bool      { return unicode.IsLetter(r) || unicode.IsDigit(r) }
func isIDChar(r rune) bool     { return isAlnum(r) || r == '_' }
func isASCIIAlnum(r rune) bool { return r <= 127 && isAlnum(r) }
func isLinkChar(r rune) bool {
	switch r {
	case '-', '.', '=', '<', '>':
		return true
	}
	return false
}

func skipSpaces(chars []rune, i int) int {
	for i < len(chars) && (chars[i] == ' ' || chars[i] == '\t') {
		i++
	}
	return i
}

// lastIndexAnyRune returns the last index in chars of any rune in set, or -1.
func lastIndexAnyRune(chars []rune, set string) int {
	for i := len(chars) - 1; i >= 0; i-- {
		for _, s := range set {
			if chars[i] == s {
				return i
			}
		}
	}
	return -1
}

// cutPrefixAny strips a single leading byte that appears in set.
func cutPrefixAny(s, set string) (string, bool) {
	if s == "" {
		return s, false
	}
	for i := 0; i < len(set); i++ {
		if s[0] == set[i] {
			return s[1:], true
		}
	}
	return s, false
}

// runesHasPrefix reports whether chars[i:] starts with the runes of p.
func runesHasPrefix(chars []rune, i int, p []rune) bool {
	if i+len(p) > len(chars) {
		return false
	}
	for k, r := range p {
		if chars[i+k] != r {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func containsWhitespace(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
