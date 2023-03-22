package utils

import "fmt"

func FmtGreen(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 32, fmt.Sprintf(format, a...))
}

func FmtCyan(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 36, fmt.Sprintf(format, a...))
}

func FmtYellow(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 33, fmt.Sprintf(format, a...))
}

func FmtRed(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 31, fmt.Sprintf(format, a...))
}

func FmtDim(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 2, fmt.Sprintf(format, a...))
}
