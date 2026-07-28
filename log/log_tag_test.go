package log

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
)

type tagPerson struct {
	Age    int    `log:"age"`
	name   string `log:"name"`
	School string
	policy string
}

type tagSkipped struct {
	Keep    string `log:"keep"`
	Skipped string `log:"-"`
}

type tagAddress struct {
	City    string `log:"city"`
	country string `log:"country"`
}

type tagPersonWithAddress struct {
	Name string     `log:"name"`
	Addr tagAddress `log:"addr"`
}

type capturingLogger struct {
	mu       sync.Mutex
	lastArgs []any
}

func (c *capturingLogger) capture(args []any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastArgs = args
}

func (c *capturingLogger) args() []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastArgs
}

func (c *capturingLogger) Debug(_ string, args ...any) { c.capture(args) }
func (c *capturingLogger) Info(_ string, args ...any)  { c.capture(args) }
func (c *capturingLogger) Warn(_ string, args ...any)  { c.capture(args) }
func (c *capturingLogger) Error(_ string, args ...any) { c.capture(args) }
func (c *capturingLogger) Fatal(_ string, args ...any) { c.capture(args) }

func TestWrapLogger_TaggedUnexportedFieldIncluded(t *testing.T) {
	p := tagPerson{Age: 10, name: "Joh", School: "MIT", policy: "ok"}
	cap := &capturingLogger{}
	l := WrapLogger(cap, nil)
	l.Info("test", "person", p)

	person, ok := cap.args()[1].(map[string]any)
	if !ok {
		t.Fatalf("expected person to be a map[string]any, got %T", cap.args()[1])
	}

	if person["name"] != "Joh" {
		t.Error(test.DiffMessage(person["name"], "Joh", "unexported field with a log tag should be logged under its tag name"))
	}
	if person["age"] != 10 {
		t.Error(test.DiffMessage(person["age"], 10, "tagged exported field should be logged under its tag name"))
	}
	if person["School"] != "MIT" {
		t.Error(test.DiffMessage(person["School"], "MIT", "untagged exported field should be logged under its Go field name"))
	}
	if _, exists := person["policy"]; exists {
		t.Error(test.DiffMessage(true, false, "untagged unexported field must not be logged"))
	}
	if len(person) != 3 {
		t.Error(test.DiffMessage(len(person), 3, "only age/name/School should be present"))
	}
}

func TestWrapLogger_ExplicitSkipTag(t *testing.T) {
	v := tagSkipped{Keep: "yes", Skipped: "no"}
	cap := &capturingLogger{}
	l := WrapLogger(cap, nil)
	l.Info("test", "v", v)

	obj := cap.args()[1].(map[string]any)
	if obj["keep"] != "yes" {
		t.Error(test.DiffMessage(obj["keep"], "yes", "field tagged log:\"keep\" should be present"))
	}
	if _, exists := obj["Skipped"]; exists {
		t.Error(test.DiffMessage(true, false, "field tagged log:\"-\" must be skipped even though exported"))
	}
}

func TestWrapLogger_NestedTaggedStruct(t *testing.T) {
	v := tagPersonWithAddress{
		Name: "Duoc",
		Addr: tagAddress{City: "Hanoi", country: "VN"},
	}
	cap := &capturingLogger{}
	l := WrapLogger(cap, nil)
	l.Info("test", "v", v)

	obj := cap.args()[1].(map[string]any)
	addr, ok := obj["addr"].(map[string]any)
	if !ok {
		t.Fatalf("expected addr to be a map[string]any, got %T", obj["addr"])
	}
	if addr["city"] != "Hanoi" {
		t.Error(test.DiffMessage(addr["city"], "Hanoi", "nested exported+tagged field should be present"))
	}
	if addr["country"] != "VN" {
		t.Error(test.DiffMessage(addr["country"], "VN", "nested unexported+tagged field should be present at any depth"))
	}
}

func TestWrapLogger_StringerStructNotExpanded(t *testing.T) {
	when := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	cap := &capturingLogger{}
	l := WrapLogger(cap, nil)
	l.Info("test", "at", when)

	if _, ok := cap.args()[1].(time.Time); !ok {
		t.Error(test.DiffMessage(cap.args()[1], "a time.Time value, unexpanded", "a Stringer struct like time.Time must not be expanded into raw fields"))
	}
}

func TestWrapLogger_LogValuerNotOverridden(t *testing.T) {
	lv := fnLogValuerStub{secret: "hidden"}
	cap := &capturingLogger{}
	l := WrapLogger(cap, nil)
	l.Info("test", "v", lv)

	if _, ok := cap.args()[1].(fnLogValuerStub); !ok {
		t.Error(test.DiffMessage(cap.args()[1], "an unexpanded fnLogValuerStub", "a type implementing slog.LogValuer should pass through untouched, not get expanded into fields"))
	}
}

type fnLogValuerStub struct {
	secret string
}

func (fnLogValuerStub) LogValue() slog.Value {
	return slog.StringValue("resolved")
}

func TestWrapLogger_CustomLoggerAutomaticallyBenefits(t *testing.T) {
	p := tagPerson{Age: 10, name: "Joh", School: "MIT", policy: "ok"}
	cap := &capturingLogger{}

	wrapped := WrapLogger(cap, nil)
	wrapped.Info("test", "person", p)

	person := cap.args()[1].(map[string]any)
	if person["name"] != "Joh" {
		t.Error(test.DiffMessage(person["name"], "Joh", "a plain custom Logger implementation should see the tagged fields without any changes on its part"))
	}
}

func TestPrettyHandler_Handle_TaggedUnexportedFieldIncluded(t *testing.T) {
	p := tagPerson{Age: 10, name: "Joh", School: "MIT", policy: "ok"}
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Any("person", p))
	_ = h.Handle(context.Background(), r)

	out := buf.String()
	if !strings.Contains(out, "name") || !strings.Contains(out, "Joh") {
		t.Error(test.DiffMessage(out, "contains name Joh", "pretty handler should also expose the tagged unexported field standalone (no WrapLogger involved)"))
	}
	if strings.Contains(out, "policy") {
		t.Error(test.DiffMessage(out, "no policy", "untagged unexported field must not leak into pretty output"))
	}
}

func TestWrapLogger_ConcurrentCallsNoDataRace(t *testing.T) {
	l := WrapLogger(&capturingLogger{}, nil)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := tagPerson{Age: 1, name: "x", School: "y", policy: "z"}
			l.Info("test", "person", p)
		}()
	}
	wg.Wait()
}
