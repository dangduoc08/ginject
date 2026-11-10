package color

import (
	"fmt"
	"strings"
)

const (
	Reset   = "\x1b[0m"
	Red     = "\x1b[31m"
	Green   = "\x1b[32m"
	Yellow  = "\x1b[33m"
	Blue    = "\x1b[34m"
	Magenta = "\x1b[35m"
	Cyan    = "\x1b[36m"
	Gray    = "\x1b[90m"
	Orange  = "\x1b[38;5;208m"
	BGDim   = "\x1b[48;5;236m"
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
	return wrap(Blue, resetFg, format, a...)
}

func FmtGreen(format string, a ...any) string {
	return wrap(Green, resetFg, format, a...)
}

func FmtCyan(format string, a ...any) string {
	return wrap(Cyan, resetFg, format, a...)
}

func FmtYellow(format string, a ...any) string {
	return wrap(Yellow, resetFg, format, a...)
}

func FmtRed(format string, a ...any) string {
	return wrap(Red, resetFg, format, a...)
}

func FmtMagenta(format string, a ...any) string {
	return wrap(Magenta, resetFg, format, a...)
}

func FmtPurple(format string, a ...any) string {
	return wrap("\x1b[38;5;129m", resetFg, format, a...)
}

func FmtOrange(format string, a ...any) string {
	return wrap(Orange, resetFg, format, a...)
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
	return wrap(BGDim, resetBg, format, a...)
}

func wrap(openCode, resetCode, format string, a ...any) string {
	if len(a) == 0 && strings.IndexByte(format, '%') < 0 {
		return openCode + format + resetCode
	}
	return openCode + fmt.Sprintf(format, a...) + resetCode
}
