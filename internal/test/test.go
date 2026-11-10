package test

import "fmt"

const (
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiGreen   = "\x1b[32m"
	ansiRed     = "\x1b[31m"
	resetWeight = "\x1b[22m"
	resetFg     = "\x1b[39m"
)

const diffFormat = ansiBold + "%s" + resetWeight + "\n" +
	ansiDim + "expected:" + resetWeight + " " + ansiGreen + "%#v" + resetFg + "\n" +
	ansiDim + "actual:" + resetWeight + "   " + ansiRed + "%#v" + resetFg

func DiffMessage(actual, expected any, desc string) string {
	return fmt.Sprintf(diffFormat, desc, expected, actual)
}
