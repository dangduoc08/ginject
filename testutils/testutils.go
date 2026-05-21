package testutils

import "fmt"

func DiffMessage(actual, expected any, desc string) string {
	s := ""
	if desc != "" {
		s = fmt.Sprintf("\x1b[32m\n%v\x1b[0m", desc)
	}
	s += fmt.Sprintf("\x1b[2m\nExpected: \x1b[31m%v\x1b[0m\x1b[0m", expected)
	s += fmt.Sprintf("\x1b[2m\nActual: \x1b[31m%v\x1b[0m\x1b[0m", actual)
	return s
}
