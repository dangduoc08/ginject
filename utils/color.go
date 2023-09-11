package utils

import "fmt"

func FmtWhite(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 97, fmt.Sprintf(format, a...))
}

func FmtBlue(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 34, fmt.Sprintf(format, a...))
}

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

func FmtMagenta(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 35, fmt.Sprintf(format, a...))
}

func FmtPurple(format string, a ...any) string {
	return fmt.Sprintf("\033[38;5;129m%s\033[39m", fmt.Sprintf(format, a...))
}

func FmtOrange(format string, a ...any) string {
	return fmt.Sprintf("\033[38;5;208m%s\033[0m", fmt.Sprintf(format, a...))
}

func FmtDim(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 2, fmt.Sprintf(format, a...))
}

func FmtBold(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 1, fmt.Sprintf(format, a...))
}

func FmtItalic(format string, a ...any) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", 3, fmt.Sprintf(format, a...))
}

func FmtBGBlue(format string, a ...any) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", 44, fmt.Sprintf(format, a...))
}

func FmtBGGreen(format string, a ...any) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", 42, fmt.Sprintf(format, a...))
}

func FmtBGRed(format string, a ...any) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", 41, fmt.Sprintf(format, a...))
}

func FmtBGYellow(format string, a ...any) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", 43, fmt.Sprintf(format, a...))
}

func FmtBGGrey(format string, a ...any) string {
	return fmt.Sprintf("\033[47m%s\033[49m", fmt.Sprintf(format, a...))
}

func FmtBGDim(format string, a ...any) string {
	return fmt.Sprintf("\033[48;5;236m%s\033[0m", fmt.Sprintf(format, a...))
}
