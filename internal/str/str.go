package str

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func RemoveSpace(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func AddBegin(str, subStr string) string {
	if str == "" {
		return str
	}

	if !strings.HasPrefix(str, subStr) {
		return subStr + str
	}

	return str
}

func AddEnd(str, subStr string) string {
	if str == "" {
		return str
	}

	if !strings.HasSuffix(str, subStr) {
		return str + subStr
	}

	return str
}

func Segment(str string, sep byte, start int) (string, int) {
	if len(str) == 0 || start < 0 || start > len(str)-1 {
		return "", -1
	}

	i := strings.IndexByte(str[start+1:], sep)
	if i < 0 {
		return str[start:], i
	}

	next := i + start + 1
	return str[start+1 : next], next
}

func Enclose(str string, sep byte) string {
	if str == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(str) + 2)
	b.WriteByte(sep)
	prev := sep
	hadContent := false

	for i := 0; i < len(str); i++ {
		c := str[i]
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

func isASCIISpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

func RemoveDup(str, pattern string) string {
	patRune, patSize := utf8.DecodeRuneInString(pattern)
	if patRune == utf8.RuneError || patSize != len(pattern) {
		return str
	}
	var b strings.Builder
	b.Grow(len(str))
	var prev rune
	for i, r := range str {
		if i > 0 && r == patRune && prev == patRune {
			prev = r
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

func IsLower(str string) []bool {
	res := make([]bool, 0, len(str))
	for _, r := range str {
		res = append(res, unicode.ToLower(r) == r)
	}
	return res
}

func IsUpper(str string) []bool {
	res := make([]bool, 0, len(str))
	for _, r := range str {
		res = append(res, unicode.ToUpper(r) == r)
	}
	return res
}
