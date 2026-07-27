package log

import (
	"bytes"
	"context"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
)

func newTestHandler(level slog.Level) (*PrettyHandler, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	h := NewPrettyHandler(buf, &PrettyHandlerOptions{
		TimeFormat:     time.DateTime,
		HandlerOptions: slog.HandlerOptions{Level: level},
	})
	return h, buf
}

// loadLogOptions tests

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

func TestLoadLogOptions_LevelMinusOneDefaultsToInfo(t *testing.T) {
	opts := loadLogOptions(&LogOptions{Level: -1})
	if opts.Level != InfoLevel {
		t.Error(test.DiffMessage(opts.Level, InfoLevel, "Level=-1 sentinel should default to InfoLevel"))
	}
}

// NewLog singleton test

func TestNewLog_Singleton(t *testing.T) {
	a := NewLog(nil)
	b := NewLog(&LogOptions{LogFormat: JSONFormat})
	if a != b {
		t.Error(test.DiffMessage(b, a, "NewLog must return the same singleton"))
	}
}

// PrettyHandler.Handle tests

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
	for _, notWant := range []string{ansiRed + "0" + ansiReset, ansiRed + "1" + ansiReset} {
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
	for _, notWant := range []string{ansiRed + "0" + ansiReset, ansiRed + "1" + ansiReset} {
		if strings.Contains(out, notWant) {
			t.Error(test.DiffMessage(out, "no index keys", "slice-of-structs elements should not be wrapped in an index-keyed node"))
		}
	}
	nameCount := strings.Count(out, ansiRed+"Name"+ansiReset)
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
		t.Fatalf("value %q not found in line %q", value, line)
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
	for _, line := range strings.Split(buf.String(), "\n") {
		switch {
		case strings.Contains(line, "VeryLongFieldName"):
			deepCol = valueColumn(t, line, "deep")
		case strings.Contains(line, "\x1b[31mbb"):
			shallowCol = valueColumn(t, line, "shallow")
		}
	}
	if deepCol == 0 || shallowCol == 0 {
		t.Fatalf("did not find both lines: deepCol=%d shallowCol=%d\n%s", deepCol, shallowCol, buf.String())
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
	if !strings.Contains(out, string(ColorRed)+"10ms (2%)"+ansiReset) {
		t.Error(test.DiffMessage(out, "contains red-colored \"10ms (2%)\"", "a Colored value should render with its own Color, not the generic Stringer green"))
	}
}
