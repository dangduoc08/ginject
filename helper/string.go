package helper

import (
	"regexp"
	"strings"
)

const (
	COLON      = ":"
	UNDERSCORE = "_"
	SLASH      = "/"
	WILDCART   = "*"
	EMPTY      = ""
)

func RemoveSpace(s string) string {
	noSpcReg := regexp.MustCompile(`\s`)
	return noSpcReg.ReplaceAllString(s, "")
}

func AddFirstSlash(s string) string {
	if string(s[0]) != SLASH {
		s = SLASH + s
	}

	return s
}

func AddLastSlash(s string) string {
	if strings.LastIndex(s, SLASH) != len(s)-1 {
		s = s + SLASH
	}

	return s
}

func RemoveLastSlash(s string) string {
	l := len(s)
	if string(s[l-1]) == SLASH {
		s = s[0 : l-1]
	}

	return s
}

func RemoveFirstColon(s string) string {
	l := len(s)
	if string(s[0]) == COLON {
		s = s[1:l]
	}

	return s
}
