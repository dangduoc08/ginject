package log

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/test"
)

func newTestHandler(level slog.Level) (*prettyHandler, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	h := newPrettyHandler(buf, &prettyHandlerOptions{
		TimeFormat:     time.DateTime,
		HandlerOptions: slog.HandlerOptions{Level: level},
	})
	return h, buf
}

func TestLoadLogOptions_NilInput(t *testing.T) {
	opts := loadLogOptions(nil)
	if opts == nil {
		t.Error(test.DiffMessage(nil, "*LogOptions", "nil input should return non-nil opts"))
	}
}

func TestLoadLogOptions_DefaultLogFormat(t *testing.T) {
	opts := loadLogOptions(nil)
	if opts.LogFormat != PrettyFormat {
		t.Error(test.DiffMessage(opts.LogFormat, PrettyFormat, "default LogFormat should be PrettyFormat"))
	}
}

func TestLoadLogOptions_DefaultTimeFormat(t *testing.T) {
	opts := loadLogOptions(nil)
	if opts.TimeFormat != time.DateTime {
		t.Error(test.DiffMessage(opts.TimeFormat, time.DateTime, "default TimeFormat"))
	}
}

func TestLoadLogOptions_PreservesCustomLevel(t *testing.T) {
	opts := loadLogOptions(&LogOptions{Level: ErrorLevel})
	if opts.Level != ErrorLevel {
		t.Error(test.DiffMessage(opts.Level, ErrorLevel, "custom level should be preserved"))
	}
}

func TestLoadLogOptions_PreservesCustomTimeFormat(t *testing.T) {
	opts := loadLogOptions(&LogOptions{TimeFormat: "2006-01-02"})
	if opts.TimeFormat != "2006-01-02" {
		t.Error(test.DiffMessage(opts.TimeFormat, "2006-01-02", "custom TimeFormat"))
	}
}

func TestLoadLogOptions_PreservesCustomLogFormat(t *testing.T) {
	opts := loadLogOptions(&LogOptions{LogFormat: TextFormat})
	if opts.LogFormat != TextFormat {
		t.Error(test.DiffMessage(opts.LogFormat, TextFormat, "custom LogFormat"))
	}
}

func TestLoadLogOptions_LevelBetweenDebugAndInfoIsKept(t *testing.T) {
	opts := loadLogOptions(&LogOptions{Level: -1})
	if opts.Level != -1 {
		t.Error(test.DiffMessage(opts.Level, slog.Level(-1), "a custom level between Debug and Info must not be rewritten"))
	}
}

func TestNewLog_Singleton(t *testing.T) {
	a := NewLog(nil)
	b := NewLog(&LogOptions{LogFormat: JSONFormat})
	if a != b {
		t.Error(test.DiffMessage(b, a, "NewLog must return the same singleton"))
	}
}

func newTestLogInstance(level slog.Level) (*logInstance, *bytes.Buffer) {
	h, buf := newTestHandler(level)
	return &logInstance{slog: slog.New(h)}, buf
}

func TestLogInstance_Debug(t *testing.T) {
	instance, buf := newTestLogInstance(slog.LevelDebug)
	instance.Debug("debug msg")
	out := buf.String()
	if !strings.Contains(out, "DEBUG") || !strings.Contains(out, "debug msg") {
		t.Error(test.DiffMessage(out, "contains DEBUG and debug msg", "Debug should log at debug level"))
	}
}

func TestLogInstance_Info(t *testing.T) {
	instance, buf := newTestLogInstance(slog.LevelDebug)
	instance.Info("info msg")
	out := buf.String()
	if !strings.Contains(out, "INFO") || !strings.Contains(out, "info msg") {
		t.Error(test.DiffMessage(out, "contains INFO and info msg", "Info should log at info level"))
	}
}

func TestLogInstance_Warn(t *testing.T) {
	instance, buf := newTestLogInstance(slog.LevelDebug)
	instance.Warn("warn msg")
	out := buf.String()
	if !strings.Contains(out, "WARN") || !strings.Contains(out, "warn msg") {
		t.Error(test.DiffMessage(out, "contains WARN and warn msg", "Warn should log at warn level"))
	}
}

func TestLogInstance_Error(t *testing.T) {
	instance, buf := newTestLogInstance(slog.LevelDebug)
	instance.Error("error msg")
	out := buf.String()
	if !strings.Contains(out, "ERROR") || !strings.Contains(out, "error msg") {
		t.Error(test.DiffMessage(out, "contains ERROR and error msg", "Error should log at error level"))
	}
}

func TestLogInstance_Debug_ForwardsArgs(t *testing.T) {
	instance, buf := newTestLogInstance(slog.LevelDebug)
	instance.Debug("msg", "key", "value")
	out := buf.String()
	if !strings.Contains(out, "key") || !strings.Contains(out, "value") {
		t.Error(test.DiffMessage(out, "contains key and value", "Debug should forward key/value args to the handler"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelInfo(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "INFO") {
		t.Error(test.DiffMessage(buf.String(), "contains INFO", "INFO level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelDebug(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelDebug, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "DEBUG") {
		t.Error(test.DiffMessage(buf.String(), "contains DEBUG", "DEBUG level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelWarn(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelWarn, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "WARN") {
		t.Error(test.DiffMessage(buf.String(), "contains WARN", "WARN level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelError(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelError, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "ERROR") {
		t.Error(test.DiffMessage(buf.String(), "contains ERROR", "ERROR level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelFatal(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), FatalLevel, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "FATAL") {
		t.Error(test.DiffMessage(buf.String(), "contains FATAL", "FATAL level label"))
	}
}

func TestPrettyHandler_Handle_ContainsMessage(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "hello world") {
		t.Error(test.DiffMessage(buf.String(), "contains hello world", "message"))
	}
}

func TestPrettyHandler_Handle_EmptyMessageSkipped(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "", 0)
	_ = h.Handle(context.Background(), r)
	if strings.Contains(buf.String(), "[]") {
		t.Error(test.DiffMessage(buf.String(), "no []", "empty message must not produce [] block"))
	}
}

func TestPrettyHandler_Handle_SingleAttr_UsesLastBranch(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k", "v"))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "└──") {
		t.Error(test.DiffMessage(buf.String(), "contains └──", "single attr should use last-branch symbol"))
	}
}

func TestPrettyHandler_Handle_MultipleAttrs_MiddleAndLastBranch(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k1", "v1"), slog.String("k2", "v2"))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "├──") {
		t.Error(test.DiffMessage(buf.String(), "contains ├──", "first attr should use middle-branch symbol"))
	}
	if !strings.Contains(buf.String(), "└──") {
		t.Error(test.DiffMessage(buf.String(), "contains └──", "last attr should use last-branch symbol"))
	}
}

func TestPrettyHandler_Handle_AttrKeyPresent(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("mykey", "myvalue"))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "mykey") {
		t.Error(test.DiffMessage(buf.String(), "contains mykey", "attr key"))
	}
}

func TestPrettyHandler_Handle_StringAttrValueNotQuoted(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k", "hello"))
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Error(test.DiffMessage(out, "contains hello", "string attr value"))
	}
	if strings.Contains(out, `"hello"`) {
		t.Error(test.DiffMessage(out, `does not contain "hello"`, "string attr value should not be quoted"))
	}
}

func TestPrettyHandler_Handle_IntAttrValuePresent(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Int("count", 42))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "42") {
		t.Error(test.DiffMessage(buf.String(), "contains 42", "int attr value"))
	}
}

func TestPrettyHandler_Handle_BoolAttrValuePresent(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Bool("ok", true))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "true") {
		t.Error(test.DiffMessage(buf.String(), "contains true", "bool attr value"))
	}
}

func TestPrettyHandler_Handle_NilAttrShowsNull(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("ptr", nil))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "null") {
		t.Error(test.DiffMessage(buf.String(), "contains null", "nil attr value should display as null"))
	}
}

func TestPrettyHandler_Handle_StartsAndEndsWithNewline(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	if !strings.HasPrefix(out, "\n") {
		t.Error(test.DiffMessage(out, "starts with \\n", "output must start with newline"))
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error(test.DiffMessage(out, "ends with \\n", "output must end with newline"))
	}
}

func TestPrettyHandler_Handle_SliceAttr_ElementsRenderedWithoutIndexKey(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("tags", []string{"admin", "beta"}))
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	for _, want := range []string{"admin", "beta"} {
		if !strings.Contains(out, want) {
			t.Error(test.DiffMessage(out, "contains "+want, "slice attr should render each element's value directly"))
		}
	}
	for _, notWant := range []string{color.Red + "0" + color.Reset, color.Red + "1" + color.Reset} {
		if strings.Contains(out, notWant) {
			t.Error(test.DiffMessage(out, "no index keys", "slice attr should not render its element index as a key"))
		}
	}
}

func TestPrettyHandler_Handle_SliceOfStructsAttr_FieldsRenderedDirectlyUnderParent(t *testing.T) {
	type entry struct {
		Name     string
		Duration string
	}
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("stage", []entry{
		{Name: "M1", Duration: "10ms"},
		{Name: "M2", Duration: "20ms"},
	}))
	_ = h.Handle(context.Background(), r)
	out := buf.String()

	for _, want := range []string{"M1", "10ms", "M2", "20ms"} {
		if !strings.Contains(out, want) {
			t.Error(test.DiffMessage(out, "contains "+want, "slice-of-structs elements should be rendered"))
		}
	}
	for _, notWant := range []string{color.Red + "0" + color.Reset, color.Red + "1" + color.Reset} {
		if strings.Contains(out, notWant) {
			t.Error(test.DiffMessage(out, "no index keys", "slice-of-structs elements should not be wrapped in an index-keyed node"))
		}
	}
	nameCount := strings.Count(out, color.Red+"Name"+color.Reset)
	if nameCount != 2 {
		t.Error(test.DiffMessage(nameCount, 2, "each struct element's own fields should be spliced in directly, one 'Name' key per element"))
	}
}

func TestPrettyHandler_Handle_StructAttr_FieldNameKeyedChildren(t *testing.T) {
	type addr struct {
		City string
	}
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("addr", addr{City: "Hanoi"}))
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	if !strings.Contains(out, "City") || !strings.Contains(out, "Hanoi") {
		t.Error(test.DiffMessage(out, "contains City Hanoi", "struct attr should render field-name-keyed children"))
	}
}

func TestPrettyHandler_Handle_MapAttr_KeyKeyedChildrenSorted(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("meta", map[string]int{"score": 42, "age": 30}))
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	ageIdx := strings.Index(out, "age")
	scoreIdx := strings.Index(out, "score")
	if ageIdx == -1 || scoreIdx == -1 || ageIdx > scoreIdx {
		t.Error(test.DiffMessage(out, "age before score", "map attr children should be sorted by key"))
	}
}

func TestPrettyHandler_Handle_NestedContainer_UsesContinuationBar(t *testing.T) {
	type wrapper struct {
		Tags []string
	}
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("w", wrapper{Tags: []string{"a", "b"}}))
	r.AddAttrs(slog.String("done", "yes"))
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	if !strings.Contains(out, "│") {
		t.Error(test.DiffMessage(out, "contains │", "a non-last top-level container should indent children under a continuation bar"))
	}
}

func TestPrettyHandler_Handle_StringerStruct_PrintedInlineNotExpanded(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("at", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)))
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	if strings.Contains(out, "wall") || strings.Contains(out, "ext") {
		t.Error(test.DiffMessage(out, "no internal time.Time fields", "a Stringer struct should be printed inline via String(), not expanded into its fields"))
	}
	if !strings.Contains(out, "2024") {
		t.Error(test.DiffMessage(out, "contains 2024", "a Stringer struct should still show its String() representation"))
	}
}

func valueColumn(t *testing.T, line, value string) int {
	t.Helper()
	line = ansiSeq.ReplaceAllString(line, "")
	i := strings.Index(line, value)
	if i == -1 {
		t.Fatal(test.DiffMessage(line, "contains "+value, "value present in the line"))
	}
	return len([]rune(line[:i]))
}

var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

func TestPrettyHandler_Handle_NestedValues_AlignAcrossDepths(t *testing.T) {
	type inner struct {
		VeryLongFieldName string
	}
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("a", inner{VeryLongFieldName: "deep"}))
	r.AddAttrs(slog.String("bb", "shallow"))
	_ = h.Handle(context.Background(), r)

	var deepCol, shallowCol int
	for line := range strings.SplitSeq(buf.String(), "\n") {
		switch {
		case strings.Contains(line, "VeryLongFieldName"):
			deepCol = valueColumn(t, line, "deep")
		case strings.Contains(line, "\x1b[31mbb"):
			shallowCol = valueColumn(t, line, "shallow")
		}
	}
	if deepCol == 0 || shallowCol == 0 {
		t.Fatal(test.DiffMessage(buf.String(), "both lines present", "deep and shallow lines found"))
	}
	if deepCol != shallowCol {
		t.Error(test.DiffMessage(shallowCol, deepCol, "a shallow top-level attr's value should align with a deeper nested field's value when the nested key is the longest in the record"))
	}
}

func TestPrettyHandler_Handle_ColoredValue_UsesItsOwnColorNotStringerGreen(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("dur", Colored{Text: "10ms (2%)", Color: ColorRed}))
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	if !strings.Contains(out, string(ColorRed)+"10ms (2%)"+color.Reset) {
		t.Error(test.DiffMessage(out, "contains red-colored \"10ms (2%)\"", "a Colored value should render with its own Color, not the generic Stringer green"))
	}
}

func TestPrettyHandler_Handle_ConcurrentCalls_NoDataRace(t *testing.T) {
	h, _ := newTestHandler(slog.LevelDebug)

	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
			r.AddAttrs(
				slog.Int("i", i),
				slog.Any("meta", map[string]int{"a": i, "b": i * 2}),
			)
			_ = h.Handle(context.Background(), r)
		}(i)
	}
	wg.Wait()
}

func renderPlain(h slog.Handler, msg string, attrs ...slog.Attr) string {
	var out bytes.Buffer
	r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)
	r.AddAttrs(attrs...)
	_ = h.Handle(context.Background(), r)
	if ph, ok := h.(*prettyHandler); ok {
		if b, ok := ph.out.writer.(*bytes.Buffer); ok {
			out.Write(b.Bytes())
		}
	}
	return ansiSeq.ReplaceAllString(out.String(), "")
}

type customError struct{ code int }

func (e *customError) Error() string { return "custom " + strconv.Itoa(e.code) }

func TestPrettyHandler_Handle_ErrorPrintsItsMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"errors.New", errors.New("boom"), "boom"},
		{"wrapped", fmt.Errorf("ctx: %w", errors.New("inner")), "ctx: inner"},
		{"custom pointer", &customError{code: 7}, "custom 7"},
	}
	for _, c := range cases {
		h, _ := newTestHandler(slog.LevelDebug)
		out := renderPlain(h, "msg", slog.Any("error", c.err))
		if !strings.Contains(out, c.want) {
			t.Error(test.DiffMessage(out, "contains "+c.want, c.name))
		}
	}
}

func TestPrettyHandler_Handle_NestedErrorAndStringerPrintInline(t *testing.T) {
	h, _ := newTestHandler(slog.LevelDebug)
	out := renderPlain(h, "msg", slog.Any("nested", map[string]any{"e": errors.New("deep"), "d": 2 * time.Second}))
	for _, want := range []string{"deep", "2s"} {
		if !strings.Contains(out, want) {
			t.Error(test.DiffMessage(out, "contains "+want, "nested error and Stringer values print their text"))
		}
	}
}

func TestPrettyHandler_Handle_NilErrorPointerIsNull(t *testing.T) {
	var err *customError
	h, _ := newTestHandler(slog.LevelDebug)
	out := renderPlain(h, "msg", slog.Any("error", err))
	if !strings.Contains(out, "null") {
		t.Error(test.DiffMessage(out, "contains null", "a nil error pointer must print null without calling Error()"))
	}
}

func TestPrettyHandler_Handle_EscapesControlBytes(t *testing.T) {
	h, _ := newTestHandler(slog.LevelDebug)
	out := renderPlain(h, "GET /a\n\x1b[41m FATAL",
		slog.String("ua", "x\n  └── admin true\x1b[2J"),
		slog.String("k\tey", "v"),
		slog.Any("err", errors.New("line1\nline2")),
		slog.Any("colored", Colored{Text: "a\rb", Color: ColorRed}),
	)
	for _, want := range []string{`GET /a\n\x1b[41m FATAL`, `x\n  └── admin true\x1b[2J`, `k\tey`, `line1\nline2`, `a\rb`} {
		if !strings.Contains(out, want) {
			t.Error(test.DiffMessage(out, "contains "+want, "control bytes are escaped"))
		}
	}
	if strings.Contains(out, "\x1b") || strings.Contains(out, "\r") {
		t.Error(test.DiffMessage(out, "no raw control bytes besides the framework's own colours", "escaping"))
	}
	if lines := strings.Count(out, "\n"); lines != 6 {
		t.Error(test.DiffMessage(lines, 6, "one line per attr plus the header and the framing newlines"))
	}
}

func TestWriteEscaped_FastPathEqualsSlowPath(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	alphabet := []byte("ab \n\r\t\x00\x1b\x7f\xc3\xa9")
	for range 100000 {
		raw := make([]byte, r.IntN(12))
		for i := range raw {
			raw[i] = alphabet[r.IntN(len(alphabet))]
		}
		var got bytes.Buffer
		writeEscaped(&got, string(raw))
		if want := escapeReference(raw); got.String() != want {
			t.Fatal(test.DiffMessage(got.String(), want, fmt.Sprintf("%q", raw)))
		}
	}
}

func escapeReference(raw []byte) string {
	var b strings.Builder
	for _, c := range raw {
		switch {
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\r':
			b.WriteString(`\r`)
		case c == '\t':
			b.WriteString(`\t`)
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

type cyclicNode struct {
	Name string
	Next *cyclicNode
}

func TestPrettyHandler_Handle_CyclesStopAtMaxDepth(t *testing.T) {
	selfMap := map[string]any{}
	selfMap["self"] = selfMap
	selfSlice := []any{nil}
	selfSlice[0] = selfSlice
	selfStruct := &cyclicNode{Name: "a"}
	selfStruct.Next = selfStruct

	for name, v := range map[string]any{"map": selfMap, "slice": selfSlice, "struct": selfStruct} {
		done := make(chan string, 1)
		go func() {
			h, _ := newTestHandler(slog.LevelDebug)
			done <- renderPlain(h, "msg", slog.Any("v", v))
		}()
		select {
		case out := <-done:
			if !strings.Contains(out, maxDepthPlaceholder) {
				t.Error(test.DiffMessage(out, "contains "+maxDepthPlaceholder, name))
			}
		case <-time.After(2 * time.Second):
			t.Fatal(test.DiffMessage("still running", "finished", name+": a cyclic value must not hang"))
		}
	}
}

func TestPrettyHandler_Handle_WideValueStopsAtMaxNodes(t *testing.T) {
	wide := make(map[string]int, 5000)
	for i := range 5000 {
		wide["k"+strconv.Itoa(i)] = i
	}
	h, _ := newTestHandler(slog.LevelDebug)
	out := renderPlain(h, "msg", slog.Any("wide", wide))
	if n := strings.Count(out, "── k"); n != maxNodes-1 {
		t.Error(test.DiffMessage(n, maxNodes-1, "rendered children"))
	}
	if !strings.Contains(out, "… (4001 more)") {
		t.Error(test.DiffMessage(out, "contains … (4001 more)", "the skipped count"))
	}
}

func TestPrettyHandler_Handle_WideSliceOfMapsStopsAtMaxNodes(t *testing.T) {
	rows := make([]map[string]int, 3000)
	for i := range rows {
		rows[i] = map[string]int{"a": i}
	}
	h, _ := newTestHandler(slog.LevelDebug)
	out := renderPlain(h, "msg", slog.Any("rows", rows))
	if n := strings.Count(out, "── a"); n != maxNodes-1 {
		t.Error(test.DiffMessage(n, maxNodes-1, "rendered children"))
	}
	if !strings.Contains(out, "… (2001 more)") {
		t.Error(test.DiffMessage(out, "contains … (2001 more)", "the skipped count"))
	}
}

func TestPrettyHandler_Enabled(t *testing.T) {
	cases := []struct {
		level   slog.Leveler
		record  slog.Level
		enabled bool
	}{
		{nil, slog.LevelInfo, true},
		{nil, slog.LevelDebug, false},
		{slog.LevelWarn, slog.LevelInfo, false},
		{slog.LevelWarn, FatalLevel, true},
		{slog.Level(-1), slog.Level(-1), true},
	}
	for _, c := range cases {
		h := newPrettyHandler(io.Discard, &prettyHandlerOptions{HandlerOptions: slog.HandlerOptions{Level: c.level}})
		if got := h.Enabled(context.Background(), c.record); got != c.enabled {
			t.Error(test.DiffMessage(got, c.enabled, fmt.Sprintf("level %v record %v", c.level, c.record)))
		}
	}
}

func TestPrettyHandler_WithAttrsAndWithGroupKeepPrettyFormat(t *testing.T) {
	var out bytes.Buffer
	base := newPrettyHandler(&out, &prettyHandlerOptions{TimeFormat: time.DateTime})
	logger := slog.New(base).With("req", "r1").WithGroup("http").With("method", "GET")
	logger.Info("hello", "code", 200)
	logger.WithGroup("").Info("again")
	slog.New(base).Info("base")

	plain := ansiSeq.ReplaceAllString(out.String(), "")
	for _, want := range []string{"├── req", "├── http.method", "└── http.code", "INFO "} {
		if !strings.Contains(plain, want) {
			t.Error(test.DiffMessage(plain, "contains "+want, "With/WithGroup keep the pretty tree and prefix grouped keys"))
		}
	}
	if strings.Contains(plain, "level=") {
		t.Error(test.DiffMessage(plain, "no text-handler output", "With must not fall back to slog.TextHandler"))
	}
	if records := strings.Split(plain, "[base]"); len(records) != 2 || strings.Contains(records[1], "req") {
		t.Error(test.DiffMessage(plain, "base record without req", "With must not change the handler it was called on"))
	}
}

func TestPrettyHandler_AddSource(t *testing.T) {
	var out bytes.Buffer
	h := newPrettyHandler(&out, &prettyHandlerOptions{TimeFormat: time.DateTime, HandlerOptions: slog.HandlerOptions{AddSource: true}})
	slog.New(h).Info("hello")
	if !strings.Contains(out.String(), "log_test.go:") {
		t.Error(test.DiffMessage(out.String(), "contains log_test.go:<line>", "AddSource prints the caller's file and line"))
	}
}

func TestPrettyHandler_Handle_ResolvesLogValuer(t *testing.T) {
	h, _ := newTestHandler(slog.LevelDebug)
	out := renderPlain(h, "msg", slog.Any("v", fnLogValuerStub{secret: "hidden"}))
	if !strings.Contains(out, "resolved") || strings.Contains(out, "hidden") {
		t.Error(test.DiffMessage(out, "contains resolved, not hidden", "a slog.LogValuer is resolved before printing"))
	}
}

func TestPrettyHandler_Handle_ConcurrentRecordsStayWhole(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(newPrettyHandler(&out, &prettyHandlerOptions{TimeFormat: time.DateTime}))
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Go(func() {
			for range 200 {
				logger.Info("m", "g", g, "user", map[string]any{"id": g})
			}
		})
	}
	wg.Wait()
	plain := ansiSeq.ReplaceAllString(out.String(), "")
	if n := strings.Count(plain, "[m]"); n != 3200 {
		t.Error(test.DiffMessage(n, 3200, "every record written"))
	}
	for record := range strings.SplitSeq(plain, "\n\n") {
		if record != "" && strings.Count(record, "[m]") != 1 {
			t.Fatal(test.DiffMessage(record, "one whole record", "records must not interleave"))
		}
	}
}

func TestPrettyHandler_Handle_OversizedRecordThenSmallRecord(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	_ = h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, strings.Repeat("x", 2*maxPooledBufferSize), 0))
	buf.Reset()
	_ = h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "small", 0))
	if plain := ansiSeq.ReplaceAllString(buf.String(), ""); !strings.Contains(plain, "[small]") || strings.Contains(plain, "xxx") {
		t.Error(test.DiffMessage(plain, "only the small record", "a record larger than the pooled buffer limit must not leak into later records"))
	}
}

func TestReplaceLevelLabel(t *testing.T) {
	cases := []struct {
		groups []string
		attr   slog.Attr
		want   any
		desc   string
	}{
		{nil, slog.Any(slog.LevelKey, FatalLevel), labelFatal, "the record level gets its label"},
		{nil, slog.Any(slog.LevelKey, slog.Level(2)), "INFO+2", "an unnamed level uses slog's name"},
		{nil, slog.String(slog.LevelKey, "high"), "high", "a user attribute named level is kept, not asserted to slog.Level"},
		{[]string{"g"}, slog.Any(slog.LevelKey, FatalLevel), FatalLevel, "inside a group the key is the user's, so it is kept as is"},
	}
	for _, c := range cases {
		if got := replaceLevelLabel(c.groups, c.attr).Value.Any(); got != c.want {
			t.Error(test.DiffMessage(got, c.want, c.desc))
		}
	}
}

func TestJSONHandler_UserAttributeNamedLevelDoesNotPanic(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{ReplaceAttr: replaceLevelLabel}))
	logger.Info("x", "level", "high", slog.Group("g", slog.String("level", "low")))
	if !strings.Contains(out.String(), `"level":"INFO"`) || !strings.Contains(out.String(), `"level":"high"`) {
		t.Error(test.DiffMessage(out.String(), `"level":"INFO" and "level":"high"`, "record level labelled, user attribute kept"))
	}
}

func TestPrettyHandler_Handle_GroupAttrsAreFlattenedWithPrefix(t *testing.T) {
	h, _ := newTestHandler(slog.LevelDebug)
	out := renderPlain(h, "msg",
		slog.Group("req", slog.String("method", "GET"), slog.Group("user", slog.Int("id", 7))),
		slog.Group("", slog.String("inline", "yes")),
		slog.Group("empty"),
	)
	for _, want := range []string{"req.method", "req.user.id", "── inline"} {
		if !strings.Contains(out, want) {
			t.Error(test.DiffMessage(out, "contains "+want, "group keys are prefixed, and an empty-key group is inlined"))
		}
	}
	if strings.Contains(out, "method=GET") || strings.Contains(out, "empty") {
		t.Error(test.DiffMessage(out, "no Attr.String() output and no empty group", "groups render as their attributes"))
	}
}

func TestPrettyHandler_Handle_PointerToColoredKeepsItsColor(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("dur", &Colored{Text: "hi", Color: ColorRed}))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), string(ColorRed)+"hi"+color.Reset) {
		t.Error(test.DiffMessage(buf.String(), "red hi", "a *Colored renders like a Colored"))
	}
}

func TestPrettyHandler_Handle_EscapedKeysStayAligned(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("a\nb", "v1"), slog.String("abcde", "v2"), slog.String("x\x01", "v3"))
	_ = h.Handle(context.Background(), r)
	var columns []int
	for line := range strings.SplitSeq(buf.String(), "\n") {
		for _, v := range []string{"v1", "v2", "v3"} {
			if strings.Contains(line, v) {
				columns = append(columns, valueColumn(t, line, v))
			}
		}
	}
	if len(columns) != 3 || columns[0] != columns[1] || columns[1] != columns[2] {
		t.Error(test.DiffMessage(columns, "three equal columns", "the width of an escaped key counts its escaped form"))
	}
}

func TestPrettyHandler_Handle_ManyAttrsBeyondTheStackArray(t *testing.T) {
	h, _ := newTestHandler(slog.LevelDebug)
	attrs := make([]slog.Attr, 20)
	for i := range attrs {
		attrs[i] = slog.Int("k"+strconv.Itoa(i), i)
	}
	out := renderPlain(h, "msg", attrs...)
	for i := range attrs {
		if !strings.Contains(out, "k"+strconv.Itoa(i)+" ") {
			t.Error(test.DiffMessage(out, "contains k"+strconv.Itoa(i), "records with more attrs than the stack array keep every attr"))
		}
	}
}

func FuzzPrettyHandler_ClientTextCannotBreakTheLayout(f *testing.F) {
	f.Add("GET /a\n[FATAL] fake", "ua", "curl\x1b[2J")
	f.Add("msg", "k\tey", "v\r\n")
	f.Fuzz(func(t *testing.T, msg, key, value string) {
		var escaped bytes.Buffer
		writeEscaped(&escaped, msg+key+value)
		if hasControlByte(escaped.String()) {
			t.Fatal(test.DiffMessage(escaped.String(), "no control bytes", "writeEscaped"))
		}

		h, buf := newTestHandler(slog.LevelDebug)
		r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)
		r.AddAttrs(slog.String(key, value), slog.Any("err", errors.New(value)))
		_ = h.Handle(context.Background(), r)
		if lines := strings.Count(buf.String(), "\n"); lines != 4 {
			t.Fatal(test.DiffMessage(lines, 4, "one header and two attr lines, whatever the client sent"))
		}
		if stripped := strings.ReplaceAll(ansiSeq.ReplaceAllString(buf.String(), ""), "\n", ""); hasControlByte(stripped) {
			t.Fatal(test.DiffMessage(stripped, "no raw control byte", "only the framework's own colour codes and line breaks"))
		}
		clean, cleanBuf := newTestHandler(slog.LevelDebug)
		cleanMessage := "m"
		if msg == "" {
			cleanMessage = ""
		}
		cleanRecord := slog.NewRecord(time.Now(), slog.LevelInfo, cleanMessage, 0)
		cleanRecord.AddAttrs(slog.String("k", "v"), slog.Any("err", errors.New("v")))
		_ = clean.Handle(context.Background(), cleanRecord)
		if got, want := strings.Count(buf.String(), "\x1b"), strings.Count(cleanBuf.String(), "\x1b"); got != want {
			t.Fatal(test.DiffMessage(got, want, "client text adds no escape sequence, not even one shaped like a colour code"))
		}
	})
}
