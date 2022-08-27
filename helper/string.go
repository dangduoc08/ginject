package helper

import (
	"regexp"
	"strings"
)

const (
	COLON               = ":"
	UNDERSCORE          = "_"
	SLASH               = "/"
	WILDCARD            = "*"
	DOLLAR_SIGN         = "$"
	OPEN_CURLY_BRACKET  = "{"
	CLOSE_CURLY_BRACKET = "}"
	DOT                 = "."
)

func RemoveSpace(s string) string {
	noSpcReg := regexp.MustCompile(`\s`)
	return noSpcReg.ReplaceAllString(s, "")
}

func AddAtBegin(s, sub string) string {
	if s[:len(sub)] != sub {
		return sub + s
	}

	return s
}

func RemoveAtBegin(s, sub string) string {
	lSub := len(sub)
	if s[:lSub] == sub {
		return s[lSub:]
	}

	return s
}

func AddAtEnd(s, sub string) string {
	lastMatchedIdx := strings.LastIndex(s, sub)
	isAtEnd := lastMatchedIdx+len(sub) == len(s)
	if !isAtEnd {
		return s + sub
	}

	return s
}

func RemoveAtEnd(s, sub string) string {
	lastMatchedIdx := strings.LastIndex(s, sub)
	isAtEnd := lastMatchedIdx+len(sub) == len(s)
	if isAtEnd {
		return s[:lastMatchedIdx]
	}

	return s
}
