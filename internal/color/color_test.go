package color

import (
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

type styleCase struct {
	name      string
	fn        func(string, ...any) string
	openCode  string
	resetCode string
}

var styleCases = []styleCase{
	{"FmtWhite", FmtWhite, "\x1b[97m", "\x1b[39m"},
	{"FmtBlue", FmtBlue, "\x1b[34m", "\x1b[39m"},
	{"FmtGreen", FmtGreen, "\x1b[32m", "\x1b[39m"},
	{"FmtCyan", FmtCyan, "\x1b[36m", "\x1b[39m"},
	{"FmtYellow", FmtYellow, "\x1b[33m", "\x1b[39m"},
	{"FmtRed", FmtRed, "\x1b[31m", "\x1b[39m"},
	{"FmtMagenta", FmtMagenta, "\x1b[35m", "\x1b[39m"},
	{"FmtPurple", FmtPurple, "\x1b[38;5;129m", "\x1b[39m"},
	{"FmtOrange", FmtOrange, "\x1b[38;5;208m", "\x1b[39m"},
	{"FmtDim", FmtDim, "\x1b[2m", "\x1b[22m"},
	{"FmtBold", FmtBold, "\x1b[1m", "\x1b[22m"},
	{"FmtItalic", FmtItalic, "\x1b[3m", "\x1b[23m"},
	{"FmtBGBlue", FmtBGBlue, "\x1b[44m", "\x1b[49m"},
	{"FmtBGGreen", FmtBGGreen, "\x1b[42m", "\x1b[49m"},
	{"FmtBGRed", FmtBGRed, "\x1b[41m", "\x1b[49m"},
	{"FmtBGYellow", FmtBGYellow, "\x1b[43m", "\x1b[49m"},
	{"FmtBGGrey", FmtBGGrey, "\x1b[47m", "\x1b[49m"},
	{"FmtBGDim", FmtBGDim, "\x1b[48;5;236m", "\x1b[49m"},
}

func TestFmtColors(t *testing.T) {
	for _, c := range styleCases {
		got := c.fn("hello %s", "world")
		want := c.openCode + "hello world" + c.resetCode
		if got != want {
			t.Error(test.DiffMessage(got, want, c.name))
		}
	}
}

func TestFmtColorsNoArgs(t *testing.T) {
	for _, c := range styleCases {
		got := c.fn("plain")
		want := c.openCode + "plain" + c.resetCode
		if got != want {
			t.Error(test.DiffMessage(got, want, c.name+" no args"))
		}
	}
}

func TestFmtColorsEmpty(t *testing.T) {
	for _, c := range styleCases {
		got := c.fn("")
		want := c.openCode + c.resetCode
		if got != want {
			t.Error(test.DiffMessage(got, want, c.name+" empty"))
		}
	}
}

func TestFmtColorsPercentNoArgs(t *testing.T) {
	got := FmtRed("100%%")
	want := "\x1b[31m100%\x1b[39m"
	if got != want {
		t.Error(test.DiffMessage(got, want, "FmtRed percent no args"))
	}
}

func TestFmtColorsPercentNoArgsAllStyles(t *testing.T) {
	for _, c := range styleCases {
		got := c.fn("cpu 100%%")
		want := c.openCode + "cpu 100%" + c.resetCode
		if got != want {
			t.Error(test.DiffMessage(got, want, c.name+" percent no args"))
		}
	}
}

func TestFmtColorsFormatArgs(t *testing.T) {
	got := FmtGreen("count: %d, label: %s, ok: %t", 42, "ok", true)
	want := "\x1b[32mcount: 42, label: ok, ok: true\x1b[39m"
	if got != want {
		t.Error(test.DiffMessage(got, want, "FmtGreen with multiple args"))
	}
}

func TestFmtColorsArgWithPercent(t *testing.T) {
	got := FmtBGRed("%s", "50%")
	want := "\x1b[41m50%\x1b[49m"
	if got != want {
		t.Error(test.DiffMessage(got, want, "FmtBGRed arg with percent"))
	}
}

func TestFmtColorsNesting(t *testing.T) {
	got := FmtBold("%s", "a "+FmtRed("b")+" c")
	want := "\x1b[1ma \x1b[31mb\x1b[39m c\x1b[22m"
	if got != want {
		t.Error(test.DiffMessage(got, want, "FmtBold nesting FmtRed"))
	}
	if strings.Contains(got, "\x1b[0m") {
		t.Error(test.DiffMessage(got, "no \x1b[0m", "FmtBold nesting FmtRed must not contain a full reset"))
	}
}

func TestFmtColorsNestingForegroundInBackground(t *testing.T) {
	got := FmtBGBlue("%s", "x"+FmtWhite("y")+"z")
	want := "\x1b[44mx\x1b[97my\x1b[39mz\x1b[49m"
	if got != want {
		t.Error(test.DiffMessage(got, want, "FmtBGBlue nesting FmtWhite"))
	}
}

func TestFmtColorsNeverFullReset(t *testing.T) {
	for _, c := range styleCases {
		outputs := []string{
			c.fn("hello %s", "world"),
			c.fn("plain"),
			c.fn(""),
			c.fn("100%%"),
		}
		for _, got := range outputs {
			if strings.Contains(got, "\x1b[0m") {
				t.Error(test.DiffMessage(got, "no \x1b[0m", c.name+" must not contain a full reset"))
			}
		}
	}
}
