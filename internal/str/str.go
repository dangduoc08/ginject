package str

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func RemoveSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func Segment(s string, sep byte, start int) (string, int) {
	if len(s) == 0 || start < 0 || start > len(s)-1 {
		return "", -1
	}

	i := strings.IndexByte(s[start+1:], sep)
	if i < 0 {
		return s[start:], i
	}

	next := i + start + 1
	return s[start+1 : next], next
}

func Enclose(s string, sep byte) string {
	if len(s) > 0 && s[0] == sep && isEnclosedClean(s, sep) {
		if s[len(s)-1] == sep {
			return s
		}
		return s + string(sep)
	}
	return encloseSlow(s, sep)
}

func isEnclosedClean(s string, sep byte) bool {
	prev := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isASCIISpace(c) || (c == prev && (c == sep || c == '*')) {
			return false
		}
		prev = c
	}
	return true
}

func encloseSlow(s string, sep byte) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte(sep)
	prev := sep
	hadContent := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if isASCIISpace(c) {
			continue
		}
		hadContent = true
		if (c == sep && prev == sep) || (c == '*' && prev == '*') {
			continue
		}
		b.WriteByte(c)
		prev = c
	}

	if !hadContent {
		return ""
	}

	if prev != sep {
		b.WriteByte(sep)
	}
	return b.String()
}

func isASCIISpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\v' || char == '\f' || char == '\r'
}

func RemoveDup(s, pattern string) string {
	patRune, patSize := utf8.DecodeRuneInString(pattern)
	if patRune == utf8.RuneError || patSize != len(pattern) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	var prev rune
	for i, r := range s {
		if i > 0 && r == patRune && prev == patRune {
			prev = r
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}
