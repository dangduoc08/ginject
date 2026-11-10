package color

import (
	"fmt"
	"strings"
)

const (
	resetFg     = "\x1b[39m"
	resetBg     = "\x1b[49m"
	resetWeight = "\x1b[22m"
	resetItalic = "\x1b[23m"
)

func FmtWhite(format string, a ...any) string {
	return wrap("\x1b[97m", resetFg, format, a...)
}

func FmtBlue(format string, a ...any) string {
	return wrap("\x1b[34m", resetFg, format, a...)
}

func FmtGreen(format string, a ...any) string {
	return wrap("\x1b[32m", resetFg, format, a...)
}

func FmtCyan(format string, a ...any) string {
	return wrap("\x1b[36m", resetFg, format, a...)
}

func FmtYellow(format string, a ...any) string {
	return wrap("\x1b[33m", resetFg, format, a...)
}

func FmtRed(format string, a ...any) string {
	return wrap("\x1b[31m", resetFg, format, a...)
}

func FmtMagenta(format string, a ...any) string {
	return wrap("\x1b[35m", resetFg, format, a...)
}

func FmtPurple(format string, a ...any) string {
	return wrap("\x1b[38;5;129m", resetFg, format, a...)
}

func FmtOrange(format string, a ...any) string {
	return wrap("\x1b[38;5;208m", resetFg, format, a...)
}

func FmtDim(format string, a ...any) string {
	return wrap("\x1b[2m", resetWeight, format, a...)
}

func FmtBold(format string, a ...any) string {
	return wrap("\x1b[1m", resetWeight, format, a...)
}

func FmtItalic(format string, a ...any) string {
	return wrap("\x1b[3m", resetItalic, format, a...)
}

func FmtBGBlue(format string, a ...any) string {
	return wrap("\x1b[44m", resetBg, format, a...)
}

func FmtBGGreen(format string, a ...any) string {
	return wrap("\x1b[42m", resetBg, format, a...)
}

func FmtBGRed(format string, a ...any) string {
	return wrap("\x1b[41m", resetBg, format, a...)
}

func FmtBGYellow(format string, a ...any) string {
	return wrap("\x1b[43m", resetBg, format, a...)
}

func FmtBGGrey(format string, a ...any) string {
	return wrap("\x1b[47m", resetBg, format, a...)
}

func FmtBGDim(format string, a ...any) string {
	return wrap("\x1b[48;5;236m", resetBg, format, a...)
}

func wrap(openCode, resetCode, format string, a ...any) string {
	if len(a) == 0 && strings.IndexByte(format, '%') < 0 {
		return openCode + format + resetCode
	}
	return openCode + fmt.Sprintf(format, a...) + resetCode
}
