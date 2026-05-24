package aggregation

import (
	"testing"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/testutils"
)

func TestNewAggregation(t *testing.T) {
	a := NewAggregation()
	if a == nil {
		t.Fatal(testutils.DiffMessage(nil, "*Aggregation", "NewAggregation must not return nil"))
	}
	if a.IsMainHandlerCalled {
		t.Error(testutils.DiffMessage(a.IsMainHandlerCalled, false, "IsMainHandlerCalled must start false"))
	}
	if a.mainData != nil {
		t.Error(testutils.DiffMessage(a.mainData, nil, "mainData must start nil"))
	}
}

func TestSetMainData(t *testing.T) {
	a := NewAggregation()
	ret := a.SetMainData("hello")
	if ret != a {
		t.Error(testutils.DiffMessage(ret, a, "SetMainData must return same *Aggregation for chaining"))
	}
	if a.mainData != "hello" {
		t.Error(testutils.DiffMessage(a.mainData, "hello", "SetMainData must store value"))
	}
	a.SetMainData(42)
	if a.mainData != 42 {
		t.Error(testutils.DiffMessage(a.mainData, 42, "SetMainData must overwrite previous value"))
	}
}

func TestSetMainData_Nil(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("something")
	a.SetMainData(nil)
	if a.mainData != nil {
		t.Error(testutils.DiffMessage(a.mainData, nil, "SetMainData with nil must clear value"))
	}
}

func TestPipe_SetsIsMainHandlerCalled(t *testing.T) {
	a := NewAggregation()
	if a.IsMainHandlerCalled {
		t.Error(testutils.DiffMessage(a.IsMainHandlerCalled, false, "must start false"))
	}
	a.Pipe()
	if !a.IsMainHandlerCalled {
		t.Error(testutils.DiffMessage(a.IsMainHandlerCalled, true, "Pipe must set IsMainHandlerCalled"))
	}
}

func TestPipe_ReturnsNil(t *testing.T) {
	a := NewAggregation()
	got := a.Pipe()
	if got != nil {
		t.Error(testutils.DiffMessage(got, nil, "Pipe must return nil"))
	}
}

func TestTransform_RegisteredAndApplied(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("original")
	called := false
	a.Transform(func(c *ctx.Context, data any) any {
		called = true
		return "transformed"
	})
	result := a.Aggregate(nil)
	if !called {
		t.Error(testutils.DiffMessage(called, true, "Transform operator must be called"))
	}
	if result != "transformed" {
		t.Error(testutils.DiffMessage(result, "transformed", "Transform must update mainData"))
	}
}

func TestTransform_ReturnsOperator(t *testing.T) {
	a := NewAggregation()
	noop := func(c *ctx.Context, data any) any { return data }
	got := a.Transform(noop)
	if got == nil {
		t.Error(testutils.DiffMessage(got, "non-nil", "Transform must return the operator"))
	}
}

func TestTap_CalledButDoesNotTransform(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("original")
	called := false
	a.Tap(func(c *ctx.Context, data any) any {
		called = true
		return "tap-result"
	})
	result := a.Aggregate(nil)
	if !called {
		t.Error(testutils.DiffMessage(called, true, "Tap operator must be called"))
	}
	if result != "original" {
		t.Error(testutils.DiffMessage(result, "original", "Tap must not change mainData"))
	}
}

func TestError_StoredButNotAppliedInAggregate(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	called := false
	a.Error(func(c *ctx.Context, data any) any {
		called = true
		return data
	})
	result := a.Aggregate(nil)
	if called {
		t.Error(testutils.DiffMessage(called, false, "Error operator must not be called in Aggregate"))
	}
	if result != "data" {
		t.Error(testutils.DiffMessage(result, "data", "mainData must be unchanged when only Error is registered"))
	}
}

func TestGetAggregationOperators_Match(t *testing.T) {
	a := NewAggregation()
	a.Transform(func(c *ctx.Context, data any) any { return data })
	a.Error(func(c *ctx.Context, data any) any { return data })
	ops := a.GetAggregationOperators(OPERATOR_ERROR)
	if len(ops) != 1 {
		t.Error(testutils.DiffMessage(len(ops), 1, "must return exactly 1 Error operator"))
	}
	if ops[0].Name != OPERATOR_ERROR {
		t.Error(testutils.DiffMessage(ops[0].Name, OPERATOR_ERROR, "returned operator must have correct name"))
	}
}

func TestGetAggregationOperators_NoMatch(t *testing.T) {
	a := NewAggregation()
	a.Transform(func(c *ctx.Context, data any) any { return data })
	ops := a.GetAggregationOperators(OPERATOR_ERROR)
	if len(ops) != 0 {
		t.Error(testutils.DiffMessage(len(ops), 0, "must return empty slice when no match"))
	}
}

func TestGetAggregationOperators_MultipleMatches(t *testing.T) {
	a := NewAggregation()
	a.Error(func(c *ctx.Context, data any) any { return "e1" })
	a.Error(func(c *ctx.Context, data any) any { return "e2" })
	ops := a.GetAggregationOperators(OPERATOR_ERROR)
	if len(ops) != 2 {
		t.Error(testutils.DiffMessage(len(ops), 2, "must return all matching operators"))
	}
}

func TestGetAggregationOperators_EmptyAggregation(t *testing.T) {
	a := NewAggregation()
	ops := a.GetAggregationOperators(OPERATOR_TRANSFORM)
	if len(ops) != 0 {
		t.Error(testutils.DiffMessage(len(ops), 0, "empty aggregation must return empty result"))
	}
}

func TestAggregate_NoOperators(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	result := a.Aggregate(nil)
	if result != "data" {
		t.Error(testutils.DiffMessage(result, "data", "no operators must return mainData unchanged"))
	}
}

func TestAggregate_NilData(t *testing.T) {
	a := NewAggregation()
	result := a.Aggregate(nil)
	if result != nil {
		t.Error(testutils.DiffMessage(result, nil, "nil mainData must be returned as nil"))
	}
}

func TestAggregate_TransformAndTap_Order(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(0)
	order := []string{}
	a.Transform(func(c *ctx.Context, data any) any {
		order = append(order, "transform")
		return data.(int) + 1
	})
	a.Tap(func(c *ctx.Context, data any) any {
		order = append(order, "tap")
		return data
	})
	result := a.Aggregate(nil)
	if result != 1 {
		t.Error(testutils.DiffMessage(result, 1, "Transform must increment value"))
	}
	if len(order) != 2 || order[0] != "transform" || order[1] != "tap" {
		t.Error(testutils.DiffMessage(order, []string{"transform", "tap"}, "operators must run in registration order"))
	}
}

func TestAggregate_MultipleTransforms(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(0)
	a.Transform(func(c *ctx.Context, data any) any { return data.(int) + 1 })
	a.Transform(func(c *ctx.Context, data any) any { return data.(int) * 3 })
	result := a.Aggregate(nil)
	if result != 3 {
		t.Error(testutils.DiffMessage(result, 3, "(0+1)*3 = 3"))
	}
}

// ---- Timeout operator ----

func newCtxWithTimestamp(ts time.Time) *ctx.Context {
	c := ctx.NewContext()
	c.Timestamp = ts
	return c
}

func TestTimeout_ReturnsAggregation(t *testing.T) {
	a := NewAggregation()
	ret := a.Timeout(time.Second)
	if ret != a {
		t.Error(testutils.DiffMessage(ret, a, "Timeout must return *Aggregation for chaining"))
	}
}

func TestTimeout_NotExpired_DoesNotPanic(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("ok")
	a.Timeout(time.Hour) // expires 1 hour from now — will not fire
	c := newCtxWithTimestamp(time.Now())
	result := a.Aggregate(c)
	if result != "ok" {
		t.Error(testutils.DiffMessage(result, "ok", "no timeout: mainData must be unchanged"))
	}
}

func TestTimeout_Expired_Panics(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Timeout(time.Millisecond) // 1ms — context formed 1 second ago
	c := newCtxWithTimestamp(time.Now().Add(-time.Second))
	defer func() {
		r := recover()
		if r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "expired timeout must panic"))
			return
		}
		ex, ok := r.(exception.Exception)
		if !ok {
			t.Error(testutils.DiffMessage(r, "exception.Exception", "panic value must be exception.Exception"))
			return
		}
		if ex.GetCode() != "408" {
			t.Error(testutils.DiffMessage(ex.GetCode(), "408", "must panic with 408 RequestTimeout"))
		}
	}()
	a.Aggregate(c)
}

func TestTimeout_NilContext_DoesNotPanic(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Timeout(time.Millisecond)
	// nil context → skip check
	result := a.Aggregate(nil)
	if result != "data" {
		t.Error(testutils.DiffMessage(result, "data", "nil context must not panic"))
	}
}

func TestTimeout_ZeroTimestamp_DoesNotPanic(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Timeout(time.Millisecond)
	c := ctx.NewContext() // Timestamp is zero value
	result := a.Aggregate(c)
	if result != "data" {
		t.Error(testutils.DiffMessage(result, "data", "zero Timestamp must not panic"))
	}
}

func TestTimeout_MeasuredFromContextTimestamp(t *testing.T) {
	// Context formed 500ms ago; timeout is 100ms — must fire
	a := NewAggregation()
	a.SetMainData("data")
	a.Timeout(100 * time.Millisecond)
	c := newCtxWithTimestamp(time.Now().Add(-500 * time.Millisecond))
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		a.Aggregate(c)
	}()
	if !panicked {
		t.Error(testutils.DiffMessage(panicked, true, "elapsed > timeout must panic"))
	}
}

func TestTimeout_ExactBoundary_Panics(t *testing.T) {
	// elapsed == timeout (>= comparison) must also fire
	a := NewAggregation()
	a.SetMainData("data")
	a.Timeout(0) // 0 duration: time.Since(...) >= 0 is always true for any non-zero timestamp
	c := newCtxWithTimestamp(time.Now().Add(-time.Nanosecond))
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		a.Aggregate(c)
	}()
	if !panicked {
		t.Error(testutils.DiffMessage(panicked, true, "elapsed >= 0 must always panic when timestamp is set"))
	}
}

func TestTimeout_Chaining_WithTransform(t *testing.T) {
	// timeout not expired → transform still runs
	a := NewAggregation()
	a.SetMainData("original")
	a.Timeout(time.Hour)
	a.Transform(func(c *ctx.Context, data any) any { return "transformed" })
	c := newCtxWithTimestamp(time.Now())
	result := a.Aggregate(c)
	if result != "transformed" {
		t.Error(testutils.DiffMessage(result, "transformed", "transform must run when timeout has not fired"))
	}
}

func TestTimeout_MultipleTimeouts_FirstExpiredPanics(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	// first timeout: already expired
	a.Timeout(time.Millisecond)
	// second timeout: would not expire
	a.Timeout(time.Hour)
	c := newCtxWithTimestamp(time.Now().Add(-time.Second))
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		a.Aggregate(c)
	}()
	if !panicked {
		t.Error(testutils.DiffMessage(panicked, true, "first expired timeout must panic before second is checked"))
	}
}
