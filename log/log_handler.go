package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/dangduoc08/ginject/internal/color"
)

const branchWidth = 4

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
	ansiGray    = "\x1b[90m"
	ansiYellow  = "\x1b[33m"
	ansiOrange  = "\033[38;5;208m"
	ansiTimePfx = "\033[48;5;236m "
	ansiTimeSfx = " \033[0m"
	ansiMsgPfx  = ansiCyan + " ["
	ansiMsgSfx  = "]" + ansiReset + " "
	ansiStrPfx  = ansiGreen
	ansiStrSfx  = ansiReset
	ansiNull    = ansiMagenta + "null" + ansiReset
)

type Color string

const (
	ColorGray   Color = ansiGray
	ColorGreen  Color = ansiGreen
	ColorBlue   Color = ansiBlue
	ColorYellow Color = ansiYellow
	ColorOrange Color = ansiOrange
	ColorRed    Color = ansiRed
)

type Colored struct {
	Text  string
	Color Color
}

func (c Colored) String() string {
	return c.Text
}

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

	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	width := 0
	var nodes []node
	for i, attr := range attrs {
		if rv := attrReflectValue(attr); rv.IsValid() {
			if nodes == nil {
				nodes = make([]node, len(attrs))
			}
			nodes[i] = buildNode("  ", kv{attr.Key, rv}, &width)
		} else if w := utf8.RuneCountInString("  ") + branchWidth + utf8.RuneCountInString(attr.Key); w > width {
			width = w
		}
	}

	for i, attr := range attrs {
		h.buf.WriteByte('\n')

		last := len(attrs) == 1 || i == len(attrs)-1
		if last {
			h.buf.WriteString("  └── ")
		} else {
			h.buf.WriteString("  ├── ")
		}

		h.writeKey(attr.Key, width-2-branchWidth-utf8.RuneCountInString(attr.Key))

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
		case slog.KindDuration, slog.KindTime:
			h.buf.WriteString(ansiGreen)
			h.buf.WriteString(attr.Value.String())
			h.buf.WriteString(ansiReset)
		default:
			var n node
			if nodes != nil {
				n = nodes[i]
			}
			if !n.val.IsValid() {
				h.buf.WriteString(ansiNull)
				break
			}

			rv := dereference(n.val)
			if isBranchable(rv) {
				h.writeChildren(indent("  ", last), n.children, width)
			} else {
				h.writeScalar(rv)
			}
		}
	}

	h.buf.WriteByte('\n')
	_, err := h.writer.Write(h.buf.Bytes())
	return err
}

func dereference(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return reflect.Value{}
		}
		rv = rv.Elem()
	}
	return rv
}

var (
	stringerType = reflect.TypeFor[fmt.Stringer]()
	errorType    = reflect.TypeFor[error]()
)

func isBranchable(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		t := rv.Type()
		return !t.Implements(stringerType) && !t.Implements(errorType)
	default:
		return false
	}
}

func indent(prefix string, parentWasLast bool) string {
	if parentWasLast {
		return prefix + "    "
	}
	return prefix + "│   "
}

type kv struct {
	key string
	val reflect.Value
}

func mapKeyString(k reflect.Value) string {
	if k.Kind() == reflect.String {
		return k.String()
	}
	return fmt.Sprint(k.Interface())
}

func collectChildren(rv reflect.Value) []kv {
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		var out []kv
		for i := range n {
			elem := dereference(rv.Index(i))
			if elem.IsValid() && isBranchable(elem) {
				out = append(out, collectChildren(elem)...)
			} else {
				out = append(out, kv{"", rv.Index(i)})
			}
		}
		return out
	case reflect.Map:
		keys := rv.MapKeys()
		out := make([]kv, len(keys))
		for i, k := range keys {
			out[i] = kv{mapKeyString(k), rv.MapIndex(k)}
		}
		sort.Slice(out, func(a, b int) bool {
			return out[a].key < out[b].key
		})
		return out
	case reflect.Struct:
		t := rv.Type()
		var out []kv
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			out = append(out, kv{f.Name, rv.Field(i)})
		}
		return out
	}
	return nil
}

func attrReflectValue(attr slog.Attr) reflect.Value {
	switch attr.Value.Kind() {
	case slog.KindString, slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool, slog.KindDuration, slog.KindTime:
		return reflect.Value{}
	default:
		v := attr.Value.Any()
		if v == nil {
			return reflect.Value{}
		}
		return dereference(reflect.ValueOf(v))
	}
}

type node struct {
	kv
	children []node
}

func buildNode(prefix string, item kv, width *int) node {
	w := utf8.RuneCountInString(prefix) + branchWidth + utf8.RuneCountInString(item.key)
	if w > *width {
		*width = w
	}

	n := node{kv: item}
	rv := dereference(item.val)
	if rv.IsValid() && isBranchable(rv) {
		childPrefix := indent(prefix, false)
		kids := collectChildren(rv)
		n.children = make([]node, len(kids))
		for i, c := range kids {
			n.children[i] = buildNode(childPrefix, c, width)
		}
	}
	return n
}

func (h *PrettyHandler) writeKey(key string, pad int) {
	h.buf.WriteString(ansiRed)
	h.buf.WriteString(key)
	h.buf.WriteString(ansiReset)
	for range pad + 1 {
		h.buf.WriteByte(' ')
	}
}

func (h *PrettyHandler) writeChildren(prefix string, children []node, width int) {
	for i, c := range children {
		h.buf.WriteByte('\n')
		h.writeNode(prefix, c, i == len(children)-1, width)
	}
}

func (h *PrettyHandler) writeNode(prefix string, n node, last bool, width int) {
	if last {
		h.buf.WriteString(prefix)
		h.buf.WriteString("└── ")
	} else {
		h.buf.WriteString(prefix)
		h.buf.WriteString("├── ")
	}

	if n.key != "" {
		h.writeKey(n.key, width-utf8.RuneCountInString(prefix)-branchWidth-utf8.RuneCountInString(n.key))
	}

	rv := dereference(n.val)
	if !rv.IsValid() {
		h.buf.WriteString(ansiNull)
		return
	}

	if isBranchable(rv) {
		h.writeChildren(indent(prefix, last), n.children, width)
		return
	}

	h.writeScalar(rv)
}

func (h *PrettyHandler) writeScalar(rv reflect.Value) {
	switch rv.Kind() {
	case reflect.String:
		h.buf.WriteString(ansiStrPfx)
		h.buf.WriteString(rv.String())
		h.buf.WriteString(ansiStrSfx)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		h.buf.WriteString(ansiOrange)
		h.buf.Write(strconv.AppendInt(h.buf.AvailableBuffer(), rv.Int(), 10))
		h.buf.WriteString(ansiReset)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		h.buf.WriteString(ansiOrange)
		h.buf.Write(strconv.AppendUint(h.buf.AvailableBuffer(), rv.Uint(), 10))
		h.buf.WriteString(ansiReset)
	case reflect.Float32, reflect.Float64:
		h.buf.WriteString(ansiOrange)
		h.buf.Write(strconv.AppendFloat(h.buf.AvailableBuffer(), rv.Float(), 'g', -1, 64))
		h.buf.WriteString(ansiReset)
	case reflect.Bool:
		h.buf.WriteString(ansiBlue)
		if rv.Bool() {
			h.buf.WriteString("true")
		} else {
			h.buf.WriteString("false")
		}
		h.buf.WriteString(ansiReset)
	default:
		if c, ok := rv.Interface().(Colored); ok {
			h.buf.WriteString(string(c.Color))
			h.buf.WriteString(c.Text)
			h.buf.WriteString(ansiReset)
			return
		}
		h.buf.WriteString(ansiGreen)
		fmt.Fprint(&h.buf, rv.Interface())
		h.buf.WriteString(ansiReset)
	}
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
