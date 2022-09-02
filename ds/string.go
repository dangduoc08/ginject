package ds

import (
	"regexp"
	"strings"
)

const (
	COLON                = ":"
	UNDERSCORE           = "_"
	SLASH                = "/"
	BACKSLASH            = "\\"
	WILDCARD             = "*"
	DOLLAR_SIGN          = "$"
	OPEN_CURLY_BRACKET   = "{"
	CLOSE_CURLY_BRACKET  = "}"
	OPEN_SQUARE_BRACKET  = "["
	CLOSE_SQUARE_BRACKET = "]"
	DOT                  = "."
)

func RemoveSpace(str string) string {
	noSpcReg := regexp.MustCompile(`\s`)
	return noSpcReg.ReplaceAllString(str, "")
}

func AddAtBegin(str, subStr string) string {
	if str[:len(subStr)] != subStr {
		return subStr + str
	}

	return str
}

func RemoveAtBegin(str, subStr string) string {
	subStrLength := len(subStr)
	if str[:subStrLength] == subStr {
		return str[subStrLength:]
	}

	return str
}

func AddAtEnd(str, subStr string) string {
	lastMatchedIndex := strings.LastIndex(str, subStr)
	isAtEnd := lastMatchedIndex+len(subStr) == len(str)
	if !isAtEnd {
		return str + subStr
	}

	return str
}

func RemoveAtEnd(str, subStr string) string {
	lastMatchedIndex := strings.LastIndex(str, subStr)
	isAtEnd := lastMatchedIndex+len(subStr) == len(str)
	if isAtEnd {
		return str[:lastMatchedIndex]
	}

	return str
}
