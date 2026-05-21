package utils

import (
	cryptoRand "crypto/rand"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func StrRemoveSpace(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func StrAddBegin(str, subStr string) string {
	if str == "" {
		return str
	}

	if !strings.HasPrefix(str, subStr) {
		return subStr + str
	}

	return str
}

func StrRemoveBegin(str, subStr string) string {
	return strings.TrimPrefix(str, subStr)
}

func StrAddEnd(str, subStr string) string {
	if str == "" {
		return str
	}

	if !strings.HasSuffix(str, subStr) {
		return str + subStr
	}

	return str
}

func StrRemoveEnd(str, subStr string) string {
	return strings.TrimSuffix(str, subStr)
}

func StrWithCharset(length int, charset string) string {
	b := make([]byte, length)
	randBytes := make([]byte, length)
	cryptoRand.Read(randBytes)
	n := len(charset)
	for i, rb := range randBytes {
		b[i] = charset[int(rb)%n]
	}
	return string(b)
}

func StrRandom(length int) string {
	charset := "`~1!2@3#4$5%6^7&8*9(0)-_=+qQwWeErRtTyYuUiIoOpP[{]}\\|aAsSdDfFgGhHjJkKlL;:'zZxXcCvVbBnNmM,<.>/?"
	return StrWithCharset(length, charset)
}

func StrSegment(str string, sep byte, start int) (string, int) {
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

func StrRemoveDup(str, pattern string) string {
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

func StrIsLower(str string) []bool {
	res := make([]bool, 0, len(str))
	for _, r := range str {
		res = append(res, unicode.ToLower(r) == r)
	}
	return res
}

func StrIsUpper(str string) []bool {
	res := make([]bool, 0, len(str))
	for _, r := range str {
		res = append(res, unicode.ToUpper(r) == r)
	}
	return res
}

func StrUUID() (string, error) {
	uuid := make([]byte, 16)
	_, err := cryptoRand.Read(uuid)
	if err != nil {
		return "", err
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func ErrorMessage(actual, expected any, desc string) string {
	if desc != "" {
		desc = FmtGreen("\n%v", desc)
	}

	return desc +
		FmtDim("\nExpected: %v", FmtRed("%v", expected)) +
		FmtDim("\nActual: %v", FmtRed("%v", actual))
}
