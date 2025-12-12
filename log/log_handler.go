package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/dangduoc08/ginject/utils"
)

type PrettyHandlerOptions struct {
	TimeFormat string
	slog.HandlerOptions
}

type PrettyHandler struct {
	levelsMapColors map[string]struct {
		txt func(string, ...any) string
		bg  func(string, ...any) string
	}
	writer     io.Writer
	timeFormat string
	slog.TextHandler
	mu sync.Mutex
}

func (h *PrettyHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.writer.Write([]byte("\n"))
	level := levelLabel[record.Level]
	space := " "

	if colorize, ok := h.levelsMapColors[level]; ok {
		switch level {
		case labelInfo, labelWarn:
			space = "  "
		}
		level = utils.FmtBold("%s", colorize.bg(" %v%v", level, space))
		h.writer.Write([]byte(level))

		time := utils.FmtBGDim(" %v ", record.Time.Format(h.timeFormat))
		h.writer.Write([]byte(time))

		h.writer.Write([]byte(colorize.bg(" ")))
	}

	if record.Message != "" {
		msg := utils.FmtCyan(" [%v]", record.Message) + " "
		h.writer.Write([]byte(msg))
	}

	i := 0
	record.Attrs(func(attr slog.Attr) bool {
		h.writer.Write([]byte("\n"))

		key := utils.FmtRed("%v", attr.Key)
		value := attr.Value.String()

		switch attr.Value.Kind() {
		case slog.KindFloat64, slog.KindInt64, slog.KindUint64:
			value = utils.FmtOrange("%s", value)
		case slog.KindBool:
			value = utils.FmtBlue("%s", value)
		case slog.KindString:
			value = utils.FmtGreen("\"%s\"", value)
		default:
			if value == "<nil>" {
				value = utils.FmtMagenta("null")
			} else {
				value = utils.FmtGreen("%v", value)
			}
		}

		pair := fmt.Sprintf("  ├── %v %v", key, value)

		if record.NumAttrs() == 1 || i == record.NumAttrs()-1 {
			pair = fmt.Sprintf("  └── %v %v", key, value)
		}

		h.writer.Write([]byte(pair))

		i++
		return true
	})

	h.writer.Write([]byte("\n"))
	return nil
}

func NewPrettyHandler(out io.Writer, opts *PrettyHandlerOptions) *PrettyHandler {
	h := &PrettyHandler{
		TextHandler: *slog.NewTextHandler(out, &opts.HandlerOptions),
		writer:      out,
		timeFormat:  opts.TimeFormat,
		levelsMapColors: map[string]struct {
			txt func(string, ...any) string
			bg  func(string, ...any) string
		}{
			labelDebug: {
				txt: utils.FmtBlue,
				bg:  utils.FmtBGBlue,
			},
			labelInfo: {
				txt: utils.FmtGreen,
				bg:  utils.FmtBGGreen,
			},
			labelWarn: {
				txt: utils.FmtYellow,
				bg:  utils.FmtBGYellow,
			},
			labelError: {
				txt: utils.FmtRed,
				bg:  utils.FmtBGRed,
			},
			labelFatal: {
				txt: utils.FmtRed,
				bg:  utils.FmtBGRed,
			},
		},
	}

	return h
}
