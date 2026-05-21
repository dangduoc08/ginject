package utils

import "testing"

func TestFmtColors(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string, ...any) string
		want string
	}{
		{"FmtWhite", FmtWhite, "\x1b[97mhello world\x1b[0m"},
		{"FmtBlue", FmtBlue, "\x1b[34mhello world\x1b[0m"},
		{"FmtGreen", FmtGreen, "\x1b[32mhello world\x1b[0m"},
		{"FmtCyan", FmtCyan, "\x1b[36mhello world\x1b[0m"},
		{"FmtYellow", FmtYellow, "\x1b[33mhello world\x1b[0m"},
		{"FmtRed", FmtRed, "\x1b[31mhello world\x1b[0m"},
		{"FmtMagenta", FmtMagenta, "\x1b[35mhello world\x1b[0m"},
		{"FmtPurple", FmtPurple, "\033[38;5;129mhello world\033[39m"},
		{"FmtOrange", FmtOrange, "\033[38;5;208mhello world\033[0m"},
		{"FmtDim", FmtDim, "\x1b[2mhello world\x1b[0m"},
		{"FmtBold", FmtBold, "\x1b[1mhello world\x1b[0m"},
		{"FmtItalic", FmtItalic, "\x1b[3mhello world\x1b[0m"},
		{"FmtBGBlue", FmtBGBlue, "\033[44mhello world\033[0m"},
		{"FmtBGGreen", FmtBGGreen, "\033[42mhello world\033[0m"},
		{"FmtBGRed", FmtBGRed, "\033[41mhello world\033[0m"},
		{"FmtBGYellow", FmtBGYellow, "\033[43mhello world\033[0m"},
		{"FmtBGGrey", FmtBGGrey, "\033[47mhello world\033[49m"},
		{"FmtBGDim", FmtBGDim, "\033[48;5;236mhello world\033[0m"},
	}

	for _, c := range cases {
		got := c.fn("hello %s", "world")
		if got != c.want {
			t.Errorf("%s = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFmtColorsFormatArgs(t *testing.T) {
	got := FmtGreen("count: %d, label: %s", 42, "ok")
	want := "\x1b[32mcount: 42, label: ok\x1b[0m"
	if got != want {
		t.Errorf("FmtGreen with multiple args = %q, want %q", got, want)
	}
}

func TestFmtColorsNoArgs(t *testing.T) {
	got := FmtRed("plain string")
	want := "\x1b[31mplain string\x1b[0m"
	if got != want {
		t.Errorf("FmtRed no args = %q, want %q", got, want)
	}
}
