package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/dangduoc08/ginject/internal/color"
)

const (
	branchWidth         = 4
	maxPooledBufferSize = 64 << 10
)

type prettyHandlerOptions struct {
	TimeFormat string
	slog.HandlerOptions
}

type levelColor struct {
	formattedLabel   string
	formattedTrailer string
}

const (
	ansiTimePfx = color.BGDim + " "
	ansiTimeSfx = " " + color.Reset
	ansiMsgPfx  = color.Cyan + " ["
	ansiMsgSfx  = "]" + color.Reset + " "
	ansiNull    = color.Magenta + "null" + color.Reset
)

type errorText string

var (
	coloredType   = reflect.TypeFor[Colored]()
	errorTextType = reflect.TypeFor[errorText]()
	bufferPool    = sync.Pool{New: func() any { return new(bytes.Buffer) }}
)

type lockedWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

func (w *lockedWriter) write(p []byte) error {
	w.mu.Lock()
	_, err := w.writer.Write(p)
	w.mu.Unlock()
	return err
}

type prettyHandler struct {
	levelColors [5]levelColor
	out         *lockedWriter
	timeFormat  string
	level       slog.Leveler
	addSource   bool
	attrs       []slog.Attr
	groupPrefix string
}

func (h *prettyHandler) colorOf(level slog.Level) (levelColor, bool) {
	switch level {
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

func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.level != nil {
		minLevel = h.level.Level()
	}
	return level >= minLevel
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	clone := *h
	clone.attrs = slices.Clip(h.attrs)
	for _, attr := range attrs {
		clone.attrs = appendFlattened(clone.attrs, h.groupPrefix, attr)
	}
	return &clone
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groupPrefix = h.groupPrefix + name + "."
	return &clone
}

func appendFlattened(attrs []slog.Attr, prefix string, attr slog.Attr) []slog.Attr {
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() != slog.KindGroup {
		attr.Key = prefix + attr.Key
		return append(attrs, attr)
	}
	if attr.Key != "" {
		prefix += attr.Key + "."
	}
	for _, sub := range attr.Value.Group() {
		attrs = appendFlattened(attrs, prefix, sub)
	}
	return attrs
}

func (h *prettyHandler) Handle(_ context.Context, record slog.Record) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	h.format(buf, record)
	err := h.out.write(buf.Bytes())
	if buf.Cap() <= maxPooledBufferSize {
		bufferPool.Put(buf)
	}
	return err
}

func (h *prettyHandler) format(buf *bytes.Buffer, record slog.Record) {
	buf.WriteByte('\n')

	if colorize, ok := h.colorOf(record.Level); ok {
		buf.WriteString(colorize.formattedLabel)
		buf.WriteString(ansiTimePfx)
		buf.Write(record.Time.AppendFormat(buf.AvailableBuffer(), h.timeFormat))
		buf.WriteString(ansiTimeSfx)
		buf.WriteString(colorize.formattedTrailer)
	}

	if record.Message != "" {
		buf.WriteString(ansiMsgPfx)
		writeEscaped(buf, record.Message)
		buf.WriteString(ansiMsgSfx)
	}

	attrCount := len(h.attrs) + record.NumAttrs()
	if h.addSource {
		attrCount++
	}
	var stack [8]slog.Attr
	attrs := stack[:0]
	if attrCount > len(stack) {
		attrs = make([]slog.Attr, 0, attrCount)
	}
	if h.addSource && record.PC != 0 {
		source := record.Source()
		attrs = append(attrs, slog.String(slog.SourceKey, source.File+":"+strconv.Itoa(source.Line)))
	}
	attrs = append(attrs, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = appendFlattened(attrs, h.groupPrefix, attr)
		return true
	})

	layout := tree{remaining: maxNodes}
	var nodes []node
	for i, attr := range attrs {
		if rv := attrReflectValue(attr); rv.IsValid() {
			if nodes == nil {
				nodes = make([]node, len(attrs))
			}
			nodes[i] = buildNode(&layout, "  ", kv{attr.Key, rv}, 0)
		} else if w := utf8.RuneCountInString("  ") + branchWidth + displayWidth(attr.Key); w > layout.width {
			layout.width = w
		}
	}

	for i, attr := range attrs {
		buf.WriteByte('\n')

		last := len(attrs) == 1 || i == len(attrs)-1
		if last {
			buf.WriteString("  └── ")
		} else {
			buf.WriteString("  ├── ")
		}

		writeKey(buf, attr.Key, layout.width-2-branchWidth-displayWidth(attr.Key))

		switch attr.Value.Kind() {
		case slog.KindString:
			buf.WriteString(color.Green)
			writeEscaped(buf, attr.Value.String())
			buf.WriteString(color.Reset)
		case slog.KindInt64:
			buf.WriteString(color.Orange)
			buf.Write(strconv.AppendInt(buf.AvailableBuffer(), attr.Value.Int64(), 10))
			buf.WriteString(color.Reset)
		case slog.KindUint64:
			buf.WriteString(color.Orange)
			buf.Write(strconv.AppendUint(buf.AvailableBuffer(), attr.Value.Uint64(), 10))
			buf.WriteString(color.Reset)
		case slog.KindFloat64:
			buf.WriteString(color.Orange)
			buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), attr.Value.Float64(), 'g', -1, 64))
			buf.WriteString(color.Reset)
		case slog.KindBool:
			buf.WriteString(color.Blue)
			if attr.Value.Bool() {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
			buf.WriteString(color.Reset)
		case slog.KindDuration, slog.KindTime:
			buf.WriteString(color.Green)
			buf.WriteString(attr.Value.String())
			buf.WriteString(color.Reset)
		default:
			var n node
			if nodes != nil {
				n = nodes[i]
			}
			rv := dereference(n.val)
			if !rv.IsValid() {
				buf.WriteString(ansiNull)
				break
			}

			if isBranchable(rv) {
				writeChildren(buf, indent("  ", last), n.children, layout.width)
			} else {
				writeScalar(buf, rv)
			}
		}
	}

	buf.WriteByte('\n')
}

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

func attrReflectValue(attr slog.Attr) reflect.Value {
	switch attr.Value.Kind() {
	case slog.KindString, slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool, slog.KindDuration, slog.KindTime:
		return reflect.Value{}
	default:
		v := attr.Value.Any()
		if v == nil {
			return reflect.Value{}
		}
		return reflect.ValueOf(v)
	}
}

type node struct {
	kv
	children []node
}

type tree struct {
	width     int
	remaining int
}

func buildNode(layout *tree, prefix string, item kv, depth int) node {
	layout.remaining--
	if w := utf8.RuneCountInString(prefix) + branchWidth + displayWidth(item.key); w > layout.width {
		layout.width = w
	}

	n := node{kv: item}
	if text, ok := textOf(item.val); ok {
		n.val = text
		return n
	}

	rv := dereference(item.val)
	if !rv.IsValid() || !isBranchable(rv) {
		return n
	}
	if depth >= maxDepth {
		n.val = reflect.ValueOf(maxDepthPlaceholder)
		return n
	}

	kids, more := collectChildren(rv, depth, max(layout.remaining, 0))
	childPrefix := indent(prefix, false)
	n.children = make([]node, 0, len(kids)+1)
	for i, c := range kids {
		if layout.remaining <= 0 {
			more += len(kids) - i
			break
		}
		n.children = append(n.children, buildNode(layout, childPrefix, c, depth+1))
	}
	if more > 0 {
		n.children = append(n.children, node{kv: kv{"", reflect.ValueOf("… (" + strconv.Itoa(more) + " more)")}})
	}
	return n
}

func collectChildren(rv reflect.Value, depth, limit int) ([]kv, int) {
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		var out []kv
		for i := range n {
			if len(out) >= limit {
				return out, n - i
			}
			elem := dereference(rv.Index(i))
			if !elem.IsValid() || !isBranchable(elem) {
				out = append(out, kv{"", rv.Index(i)})
				continue
			}
			if depth >= maxDepth {
				out = append(out, kv{"", reflect.ValueOf(maxDepthPlaceholder)})
				continue
			}
			flat, more := collectChildren(elem, depth+1, limit-len(out))
			out = append(out, flat...)
			if more > 0 {
				return out, more + n - i - 1
			}
		}
		return out, 0
	case reflect.Map:
		total := rv.Len()
		out := make([]kv, 0, min(total, limit))
		iter := rv.MapRange()
		for len(out) < limit && iter.Next() {
			out = append(out, kv{mapKeyString(iter.Key()), iter.Value()})
		}
		slices.SortFunc(out, func(a, b kv) int {
			return strings.Compare(a.key, b.key)
		})
		return out, total - len(out)
	case reflect.Struct:
		fields := taggedFields(rv)
		if len(fields) > limit {
			return fields[:limit], len(fields) - limit
		}
		return fields, 0
	}
	return nil, 0
}

func textOf(rv reflect.Value) (reflect.Value, bool) {
	if base := dereference(rv); base.IsValid() && base.Type() == coloredType {
		return reflect.Value{}, false
	}
	for rv.IsValid() {
		kind := rv.Kind()
		if (kind == reflect.Pointer || kind == reflect.Interface) && rv.IsNil() {
			return reflect.Value{}, false
		}
		valueType := rv.Type()
		if !rv.CanInterface() {
			return reflect.Value{}, false
		}
		if valueType.Implements(errorType) {
			return reflect.ValueOf(errorText(rv.Interface().(error).Error())), true
		}
		if valueType.Implements(stringerType) {
			return reflect.ValueOf(rv.Interface().(fmt.Stringer).String()), true
		}
		if kind != reflect.Pointer && kind != reflect.Interface {
			return reflect.Value{}, false
		}
		rv = rv.Elem()
	}
	return reflect.Value{}, false
}

func writeKey(buf *bytes.Buffer, key string, pad int) {
	buf.WriteString(color.Red)
	writeEscaped(buf, key)
	buf.WriteString(color.Reset)
	for range pad + 1 {
		buf.WriteByte(' ')
	}
}

func writeChildren(buf *bytes.Buffer, prefix string, children []node, width int) {
	for i, c := range children {
		buf.WriteByte('\n')
		writeNode(buf, prefix, c, i == len(children)-1, width)
	}
}

func writeNode(buf *bytes.Buffer, prefix string, item node, last bool, width int) {
	if last {
		buf.WriteString(prefix)
		buf.WriteString("└── ")
	} else {
		buf.WriteString(prefix)
		buf.WriteString("├── ")
	}

	if item.key != "" {
		writeKey(buf, item.key, width-utf8.RuneCountInString(prefix)-branchWidth-displayWidth(item.key))
	}

	rv := dereference(item.val)
	if !rv.IsValid() {
		buf.WriteString(ansiNull)
		return
	}

	if isBranchable(rv) {
		writeChildren(buf, indent(prefix, last), item.children, width)
		return
	}

	writeScalar(buf, rv)
}

func writeScalar(buf *bytes.Buffer, rv reflect.Value) {
	if rv.Type() == errorTextType {
		buf.WriteString(color.Red)
		writeEscaped(buf, rv.String())
		buf.WriteString(color.Reset)
		return
	}

	switch rv.Kind() {
	case reflect.String:
		buf.WriteString(color.Green)
		writeEscaped(buf, rv.String())
		buf.WriteString(color.Reset)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		buf.WriteString(color.Orange)
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), rv.Int(), 10))
		buf.WriteString(color.Reset)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		buf.WriteString(color.Orange)
		buf.Write(strconv.AppendUint(buf.AvailableBuffer(), rv.Uint(), 10))
		buf.WriteString(color.Reset)
	case reflect.Float32, reflect.Float64:
		buf.WriteString(color.Orange)
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), rv.Float(), 'g', -1, 64))
		buf.WriteString(color.Reset)
	case reflect.Bool:
		buf.WriteString(color.Blue)
		if rv.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		buf.WriteString(color.Reset)
	default:
		if c, ok := rv.Interface().(Colored); ok {
			buf.WriteString(string(c.Color))
			writeEscaped(buf, c.Text)
			buf.WriteString(color.Reset)
			return
		}
		buf.WriteString(color.Green)
		fmt.Fprint(buf, rv.Interface())
		buf.WriteString(color.Reset)
	}
}

func writeEscaped(buf *bytes.Buffer, s string) {
	if !hasControlByte(s) {
		buf.WriteString(s)
		return
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch {
		case b == '\n':
			buf.WriteString(`\n`)
		case b == '\r':
			buf.WriteString(`\r`)
		case b == '\t':
			buf.WriteString(`\t`)
		case isControlByte(b):
			buf.WriteString(`\x`)
			buf.WriteByte(hexDigits[b>>4])
			buf.WriteByte(hexDigits[b&0x0f])
		default:
			buf.WriteByte(b)
		}
	}
}

func displayWidth(key string) int {
	width := utf8.RuneCountInString(key)
	if !hasControlByte(key) {
		return width
	}
	for i := 0; i < len(key); i++ {
		switch b := key[i]; {
		case b == '\n' || b == '\r' || b == '\t':
			width++
		case isControlByte(b):
			width += 3
		}
	}
	return width
}

const hexDigits = "0123456789abcdef"

func hasControlByte(s string) bool {
	for i := 0; i < len(s); i++ {
		if isControlByte(s[i]) {
			return true
		}
	}
	return false
}

func isControlByte(b byte) bool {
	return b < 0x20 || b == 0x7f
}

func newPrettyHandler(out io.Writer, opts *prettyHandlerOptions) *prettyHandler {
	buildLabel := func(label string, bg func(string, ...any) string) string {
		space := " "
		if label == labelInfo || label == labelWarn {
			space = "  "
		}
		return color.FmtBold("%s", bg(" "+label+space))
	}

	h := &prettyHandler{
		out:        &lockedWriter{writer: out},
		timeFormat: opts.TimeFormat,
		level:      opts.Level,
		addSource:  opts.AddSource,
	}
	h.levelColors[0] = levelColor{buildLabel(labelDebug, color.FmtBGBlue), color.FmtBGBlue(" ")}
	h.levelColors[1] = levelColor{buildLabel(labelInfo, color.FmtBGGreen), color.FmtBGGreen(" ")}
	h.levelColors[2] = levelColor{buildLabel(labelWarn, color.FmtBGYellow), color.FmtBGYellow(" ")}
	h.levelColors[3] = levelColor{buildLabel(labelError, color.FmtBGRed), color.FmtBGRed(" ")}
	h.levelColors[4] = levelColor{buildLabel(labelFatal, color.FmtBGRed), color.FmtBGRed(" ")}
	return h
}
