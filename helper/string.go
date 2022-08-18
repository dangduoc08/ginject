package helper

import (
	"regexp"
	"strings"
)

func RemoveSpace(s string) string {
	noSpcReg := regexp.MustCompile(`\s`)
	return noSpcReg.ReplaceAllString(s, "")
}

func AddSlash(s string) string {
	// begin
	if string(s[0]) != "/" {
		s = "/" + s
	}

	// end
	if strings.LastIndex(s, "/") != len(s)-1 {
		s = s + "/"
	}

	return s
}
