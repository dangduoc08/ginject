package log

import "github.com/dangduoc08/ginject/internal/color"

type Color string

const (
	ColorGray   Color = color.Gray
	ColorGreen  Color = color.Green
	ColorBlue   Color = color.Blue
	ColorYellow Color = color.Yellow
	ColorOrange Color = color.Orange
	ColorRed    Color = color.Red
)

type Colored struct {
	Text  string
	Color Color
}

func (c Colored) String() string {
	return c.Text
}
