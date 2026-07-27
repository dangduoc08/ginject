package wsevent_test

import (
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/wsevent"
)

func TestWSEvent_MatchExactPattern(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.created", wsevent.WSEventItem{Handler: "exact-handler"})

	value, pattern, ok := r.Match("chat.created")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "exact topic should match its own pattern"))
	}
	if value.Handler != "exact-handler" {
		t.Error(test.DiffMessage(value.Handler, "exact-handler", "unexpected value for exact match"))
	}
	if pattern != "chat.created" {
		t.Error(test.DiffMessage(pattern, "chat.created", "unexpected matched pattern"))
	}
}

func TestWSEvent_MatchWildcardPattern(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.to.*", wsevent.WSEventItem{Handler: "wildcard-handler"})

	value, pattern, ok := r.Match("chat.to.user2")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "topic should match a trailing wildcard pattern"))
	}
	if value.Handler != "wildcard-handler" {
		t.Error(test.DiffMessage(value.Handler, "wildcard-handler", "unexpected value for wildcard match"))
	}
	if pattern != "chat.to.*" {
		t.Error(test.DiffMessage(pattern, "chat.to.*", "unexpected matched pattern"))
	}
}

// A trailing "*" is greedy: it matches one or more remaining segments, not
// just exactly one.
func TestWSEvent_MatchWildcardMultipleSegments(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.to.*", wsevent.WSEventItem{Handler: "wildcard-handler"})

	value, pattern, ok := r.Match("chat.to.user2.thread.5")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "topic should match a trailing wildcard pattern across multiple segments"))
	}
	if value.Handler != "wildcard-handler" || pattern != "chat.to.*" {
		t.Error(test.DiffMessage(value.Handler, "wildcard-handler", "unexpected value/pattern for multi-segment wildcard match"))
	}
}

// Two trailing-wildcard patterns registered at different depths ("chat.*"
// and "chat.to.*") can both be reachable from the same topic; the most
// specific (longest/deepest) registered prefix must win.
func TestWSEvent_SuffixWildcard_MostSpecificPrefixWins(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.*", wsevent.WSEventItem{Handler: "shallow-handler"})
	r.Add("chat.to.*", wsevent.WSEventItem{Handler: "deep-handler"})

	value, pattern, ok := r.Match("chat.to.user2")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if value.Handler != "deep-handler" || pattern != "chat.to.*" {
		t.Error(test.DiffMessage(value.Handler, "deep-handler", "the deeper/more specific suffix-wildcard prefix should win"))
	}
}

func TestWSEvent_MatchWildcardMidPattern(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.to.*.in.group", wsevent.WSEventItem{Handler: "mid-wildcard-handler"})

	_, pattern, ok := r.Match("chat.to.user2.in.group")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "topic should match a wildcard segment in the middle of a pattern"))
	}
	if pattern != "chat.to.*.in.group" {
		t.Error(test.DiffMessage(pattern, "chat.to.*.in.group", "unexpected matched pattern"))
	}
}

func TestWSEvent_NoMatchReturnsFalse(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.to.*", wsevent.WSEventItem{Handler: "wildcard-handler"})

	value, pattern, ok := r.Match("random.topic")
	if ok {
		t.Fatal(test.DiffMessage(ok, false, "unrelated topic should not match"))
	}
	if value.Handler != nil {
		t.Error(test.DiffMessage(value.Handler, nil, "value should be zero value when no match"))
	}
	if pattern != "" {
		t.Error(test.DiffMessage(pattern, "", "pattern should be empty when no match"))
	}
}

func TestWSEvent_ExactPreferredOverWildcard(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.to.*", wsevent.WSEventItem{Handler: "wildcard-handler"})
	r.Add("chat.to.user2", wsevent.WSEventItem{Handler: "exact-handler"})

	value, pattern, ok := r.Match("chat.to.user2")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if value.Handler != "exact-handler" || pattern != "chat.to.user2" {
		t.Error(test.DiffMessage(value.Handler, "exact-handler", "exact pattern should win over an overlapping wildcard pattern"))
	}
}

func TestWSEvent_AddOverwritesPreviousValue(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.to.*", wsevent.WSEventItem{Handler: "first"})
	r.Add("chat.to.*", wsevent.WSEventItem{Handler: "second"})

	value, _, ok := r.Match("chat.to.user2")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if value.Handler != "second" {
		t.Error(test.DiffMessage(value.Handler, "second", "re-adding the same pattern should overwrite the value"))
	}
}

// WSEvent is not safe for concurrent Add — callers must finish adding
// (e.g. during app boot) before any concurrent Match (e.g. across WS
// connection goroutines) begins. This test covers exactly that contract:
// sequential Add, then concurrent Match.
func TestWSEvent_ConcurrentMatchAfterAdd_NoDataRace(t *testing.T) {
	r := wsevent.NewWSEvent()
	for i := 0; i < 8; i++ {
		r.Add("topic.*", wsevent.WSEventItem{Handler: i})
	}

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			r.Match("topic.anything")
		}()
	}

	wg.Wait()
}

func TestWSEvent_AddMiddlewaresBeforeInjectableHandler_MatchesAndKeepsBoth(t *testing.T) {
	r := wsevent.NewWSEvent()
	mw1 := func(*ctx.WSContext) {}
	mw2 := func(*ctx.WSContext) {}
	handler := func() {}

	r.AddMiddlewares("chat.created", mw1)
	r.AddMiddlewares("chat.created", mw2)
	r.AddInjectableHandler("chat.created", handler)

	value, pattern, ok := r.Match("chat.created")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "pattern should become matchable once AddInjectableHandler registers its handler"))
	}
	if pattern != "chat.created" {
		t.Error(test.DiffMessage(pattern, "chat.created", "unexpected matched pattern"))
	}
	if len(value.Middlewares) != 2 {
		t.Fatal(test.DiffMessage(len(value.Middlewares), 2, "middlewares added before the handler should be preserved"))
	}
}

func TestWSEvent_AddMiddlewaresAfterInjectableHandler_KeepsHandler(t *testing.T) {
	r := wsevent.NewWSEvent()
	handler := func() {}
	mw := func(*ctx.WSContext) {}

	r.AddInjectableHandler("chat.created", handler)
	r.AddMiddlewares("chat.created", mw)

	value, _, ok := r.Match("chat.created")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if len(value.Middlewares) != 1 {
		t.Error(test.DiffMessage(len(value.Middlewares), 1, "middlewares added after the handler should still be recorded"))
	}
	if value.Handler == nil {
		t.Error(test.DiffMessage(value.Handler, handler, "handler should be preserved after adding more middlewares"))
	}
}

func TestWSEvent_AddMiddlewaresAlone_PatternStaysUnmatchable(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.AddMiddlewares("chat.created", func(*ctx.WSContext) {})

	_, _, ok := r.Match("chat.created")
	if ok {
		t.Error(test.DiffMessage(ok, false, "a pattern with only middlewares and no registered handler should not be matchable"))
	}
}

func TestWSEvent_AddInjectableHandlerPanicsOnNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when handler is nil")
		}
	}()
	wsevent.NewWSEvent().AddInjectableHandler("chat.created", nil)
}

func TestWSEvent_AddInjectableHandlerPanicsOnNonFunc(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when handler is not a func")
		}
	}()
	wsevent.NewWSEvent().AddInjectableHandler("chat.created", "not a func")
}

func TestWSEvent_MatchGlobalPattern(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("*", wsevent.WSEventItem{Handler: "global-handler"})

	value, pattern, ok := r.Match("anything.at.all")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "any topic should match a global pattern"))
	}
	if value.Handler != "global-handler" || pattern != "*" {
		t.Error(test.DiffMessage([]any{value.Handler, pattern}, []any{"global-handler", "*"}, "unexpected value/pattern for global match"))
	}
}

func TestWSEvent_ExactPreferredOverGlobal(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("*", wsevent.WSEventItem{Handler: "global-handler"})
	r.Add("chat.created", wsevent.WSEventItem{Handler: "exact-handler"})

	value, pattern, ok := r.Match("chat.created")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if value.Handler != "exact-handler" || pattern != "chat.created" {
		t.Error(test.DiffMessage(value.Handler, "exact-handler", "exact pattern should win over an overlapping global pattern"))
	}
}

func TestWSEvent_SuffixWildcardPreferredOverComplex(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("chat.*.urgent", wsevent.WSEventItem{Handler: "complex-handler"})
	r.Add("chat.*", wsevent.WSEventItem{Handler: "suffix-handler"})

	value, pattern, ok := r.Match("chat.urgent")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if value.Handler != "suffix-handler" || pattern != "chat.*" {
		t.Error(test.DiffMessage(value.Handler, "suffix-handler", "a suffix-wildcard pattern should win over an overlapping complex pattern"))
	}
}

func TestWSEvent_ComplexPreferredOverGlobal(t *testing.T) {
	r := wsevent.NewWSEvent()
	r.Add("*", wsevent.WSEventItem{Handler: "global-handler"})
	r.Add("chat.*.urgent", wsevent.WSEventItem{Handler: "complex-handler"})

	value, pattern, ok := r.Match("chat.room1.urgent")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if value.Handler != "complex-handler" || pattern != "chat.*.urgent" {
		t.Error(test.DiffMessage(value.Handler, "complex-handler", "a complex pattern should win over an overlapping global pattern"))
	}
}

func TestWSEvent_RepeatedAddOnSamePatternDoesNotDuplicateMatches(t *testing.T) {
	r := wsevent.NewWSEvent()
	mw := func(*ctx.WSContext) {}
	r.AddMiddlewares("chat.*.urgent", mw)
	r.AddMiddlewares("chat.*.urgent", mw)
	r.AddInjectableHandler("chat.*.urgent", func() {})
	r.AddInjectableHandler("chat.*.urgent", func() {})

	value, pattern, ok := r.Match("chat.room1.urgent")
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "should match"))
	}
	if pattern != "chat.*.urgent" {
		t.Error(test.DiffMessage(pattern, "chat.*.urgent", "unexpected matched pattern"))
	}
	if len(value.Middlewares) != 2 {
		t.Error(test.DiffMessage(len(value.Middlewares), 2, "middlewares should accumulate, not duplicate pattern registration"))
	}
}
