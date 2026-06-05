package log

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"
	"sync"

	"github.com/dangduoc08/ginject/internal/color"
)

type PrettyHandlerOptions struct {
	TimeFormat string
	slog.HandlerOptions
}

type levelColor struct {
	formattedLabel   string
	formattedTrailer string
}

const (
	ansiReset   = "\x1b[0m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
	ansiOrange  = "\033[38;5;208m"
	ansiTimePfx = "\033[48;5;236m "
	ansiTimeSfx = " \033[0m"
	ansiMsgPfx  = ansiCyan + " ["
	ansiMsgSfx  = "]" + ansiReset + " "
	ansiStrPfx  = ansiGreen + "\""
	ansiStrSfx  = "\"" + ansiReset
	ansiNull    = ansiMagenta + "null" + ansiReset
)

type PrettyHandler struct {
	levelColors [5]levelColor
	writer      io.Writer
	timeFormat  string
	buf         bytes.Buffer
	slog.TextHandler
	mu sync.Mutex
}

func (h *PrettyHandler) colorOf(l slog.Level) (levelColor, bool) {
	switch l {
	case DebugLevel:
		return h.levelColors[0], true
	case InfoLevel:
		return h.levelColors[1], true
	case WarnLevel:
		return h.levelColors[2], true
	case ErrorLevel:
		return h.levelColors[3], true
	case FatalLevel:
		return h.levelColors[4], true
	default:
		return levelColor{}, false
	}
}

func (h *PrettyHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buf.Reset()
	h.buf.WriteByte('\n')

	if colorize, ok := h.colorOf(record.Level); ok {
		h.buf.WriteString(colorize.formattedLabel)
		h.buf.WriteString(ansiTimePfx)
		h.buf.Write(record.Time.AppendFormat(h.buf.AvailableBuffer(), h.timeFormat))
		h.buf.WriteString(ansiTimeSfx)
		h.buf.WriteString(colorize.formattedTrailer)
	}

	if record.Message != "" {
		h.buf.WriteString(ansiMsgPfx)
		h.buf.WriteString(record.Message)
		h.buf.WriteString(ansiMsgSfx)
	}

	numAttrs := record.NumAttrs()
	i := 0
	record.Attrs(func(attr slog.Attr) bool {
		h.buf.WriteByte('\n')

		if numAttrs == 1 || i == numAttrs-1 {
			h.buf.WriteString("  └── ")
		} else {
			h.buf.WriteString("  ├── ")
		}

		h.buf.WriteString(ansiRed)
		h.buf.WriteString(attr.Key)
		h.buf.WriteString(ansiReset)
		h.buf.WriteByte(' ')

		switch attr.Value.Kind() {
		case slog.KindString:
			h.buf.WriteString(ansiStrPfx)
			h.buf.WriteString(attr.Value.String())
			h.buf.WriteString(ansiStrSfx)
		case slog.KindInt64:
			h.buf.WriteString(ansiOrange)
			h.buf.Write(strconv.AppendInt(h.buf.AvailableBuffer(), attr.Value.Int64(), 10))
			h.buf.WriteString(ansiReset)
		case slog.KindUint64:
			h.buf.WriteString(ansiOrange)
			h.buf.Write(strconv.AppendUint(h.buf.AvailableBuffer(), attr.Value.Uint64(), 10))
			h.buf.WriteString(ansiReset)
		case slog.KindFloat64:
			h.buf.WriteString(ansiOrange)
			h.buf.Write(strconv.AppendFloat(h.buf.AvailableBuffer(), attr.Value.Float64(), 'g', -1, 64))
			h.buf.WriteString(ansiReset)
		case slog.KindBool:
			h.buf.WriteString(ansiBlue)
			if attr.Value.Bool() {
				h.buf.WriteString("true")
			} else {
				h.buf.WriteString("false")
			}
			h.buf.WriteString(ansiReset)
		default:
			raw := attr.Value.String()
			if raw == "<nil>" {
				h.buf.WriteString(ansiNull)
			} else {
				h.buf.WriteString(ansiGreen)
				h.buf.WriteString(raw)
				h.buf.WriteString(ansiReset)
			}
		}

		i++
		return true
	})

	h.buf.WriteByte('\n')
	_, err := h.writer.Write(h.buf.Bytes())
	return err
}

func NewPrettyHandler(out io.Writer, opts *PrettyHandlerOptions) *PrettyHandler {
	buildLabel := func(label string, bg func(string, ...any) string) string {
		space := " "
		if label == labelInfo || label == labelWarn {
			space = "  "
		}
		return color.FmtBold("%s", bg(" "+label+space))
	}

	h := &PrettyHandler{
		TextHandler: *slog.NewTextHandler(out, &opts.HandlerOptions),
		writer:      out,
		timeFormat:  opts.TimeFormat,
	}
	h.levelColors[0] = levelColor{buildLabel(labelDebug, color.FmtBGBlue), color.FmtBGBlue(" ")}
	h.levelColors[1] = levelColor{buildLabel(labelInfo, color.FmtBGGreen), color.FmtBGGreen(" ")}
	h.levelColors[2] = levelColor{buildLabel(labelWarn, color.FmtBGYellow), color.FmtBGYellow(" ")}
	h.levelColors[3] = levelColor{buildLabel(labelError, color.FmtBGRed), color.FmtBGRed(" ")}
	h.levelColors[4] = levelColor{buildLabel(labelFatal, color.FmtBGRed), color.FmtBGRed(" ")}
	return h
}
