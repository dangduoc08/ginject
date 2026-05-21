package utils

import "fmt"

func FmtWhite(format string, a ...any) string {
	return "\x1b[97m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtBlue(format string, a ...any) string {
	return "\x1b[34m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtGreen(format string, a ...any) string {
	return "\x1b[32m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtCyan(format string, a ...any) string {
	return "\x1b[36m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtYellow(format string, a ...any) string {
	return "\x1b[33m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtRed(format string, a ...any) string {
	return "\x1b[31m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtMagenta(format string, a ...any) string {
	return "\x1b[35m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtPurple(format string, a ...any) string {
	return "\033[38;5;129m" + fmt.Sprintf(format, a...) + "\033[39m"
}

func FmtOrange(format string, a ...any) string {
	return "\033[38;5;208m" + fmt.Sprintf(format, a...) + "\033[0m"
}

func FmtDim(format string, a ...any) string {
	return "\x1b[2m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtBold(format string, a ...any) string {
	return "\x1b[1m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtItalic(format string, a ...any) string {
	return "\x1b[3m" + fmt.Sprintf(format, a...) + "\x1b[0m"
}

func FmtBGBlue(format string, a ...any) string {
	return "\033[44m" + fmt.Sprintf(format, a...) + "\033[0m"
}

func FmtBGGreen(format string, a ...any) string {
	return "\033[42m" + fmt.Sprintf(format, a...) + "\033[0m"
}

func FmtBGRed(format string, a ...any) string {
	return "\033[41m" + fmt.Sprintf(format, a...) + "\033[0m"
}

func FmtBGYellow(format string, a ...any) string {
	return "\033[43m" + fmt.Sprintf(format, a...) + "\033[0m"
}

func FmtBGGrey(format string, a ...any) string {
	return "\033[47m" + fmt.Sprintf(format, a...) + "\033[49m"
}

func FmtBGDim(format string, a ...any) string {
	return "\033[48;5;236m" + fmt.Sprintf(format, a...) + "\033[0m"
}
