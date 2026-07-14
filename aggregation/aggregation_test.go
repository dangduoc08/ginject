package aggregation

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func TestNewAggregation(t *testing.T) {
	a := NewAggregation()
	if a == nil {
		t.Fatal(test.DiffMessage(nil, "*Aggregation", "NewAggregation must not return nil"))
	}
	if a.IsMainHandlerCalled {
		t.Error(test.DiffMessage(a.IsMainHandlerCalled, false, "IsMainHandlerCalled must start false"))
	}
	if a.mainData != nil {
		t.Error(test.DiffMessage(a.mainData, nil, "mainData must start nil"))
	}
}

func TestSetMainData(t *testing.T) {
	a := NewAggregation()
	ret := a.SetMainData("hello")
	if ret != a {
		t.Error(test.DiffMessage(ret, a, "SetMainData must return same *Aggregation for chaining"))
	}
	if a.mainData != "hello" {
		t.Error(test.DiffMessage(a.mainData, "hello", "SetMainData must store value"))
	}
	a.SetMainData(42)
	if a.mainData != 42 {
		t.Error(test.DiffMessage(a.mainData, 42, "SetMainData must overwrite previous value"))
	}
}

func TestSetMainData_Nil(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("something")
	a.SetMainData(nil)
	if a.mainData != nil {
		t.Error(test.DiffMessage(a.mainData, nil, "SetMainData with nil must clear value"))
	}
}

func TestPipe_SetsIsMainHandlerCalled(t *testing.T) {
	a := NewAggregation()
	if a.IsMainHandlerCalled {
		t.Error(test.DiffMessage(a.IsMainHandlerCalled, false, "must start false"))
	}
	a.Pipe()
	if !a.IsMainHandlerCalled {
		t.Error(test.DiffMessage(a.IsMainHandlerCalled, true, "Pipe must set IsMainHandlerCalled"))
	}
}

func TestPipe_ReturnsNil(t *testing.T) {
	a := NewAggregation()
	got := a.Pipe()
	if got != nil {
		t.Error(test.DiffMessage(got, nil, "Pipe must return nil"))
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
		t.Error(test.DiffMessage(called, true, "Transform operator must be called"))
	}
	if result != "transformed" {
		t.Error(test.DiffMessage(result, "transformed", "Transform must update mainData"))
	}
}

func TestTransform_ReturnsOperator(t *testing.T) {
	a := NewAggregation()
	noop := func(c *ctx.Context, data any) any { return data }
	got := a.Transform(noop)
	if got == nil {
		t.Error(test.DiffMessage(got, "non-nil", "Transform must return the operator"))
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
		t.Error(test.DiffMessage(called, true, "Tap operator must be called"))
	}
	if result != "original" {
		t.Error(test.DiffMessage(result, "original", "Tap must not change mainData"))
	}
}

func TestGetAggregationOperators_Match(t *testing.T) {
	a := NewAggregation()
	a.Transform(func(c *ctx.Context, data any) any { return data })
	a.Tap(func(c *ctx.Context, data any) any { return data })
	ops := a.GetAggregationOperators(OperatorTap)
	if len(ops) != 1 {
		t.Error(test.DiffMessage(len(ops), 1, "must return exactly 1 Tap operator"))
	}
	if ops[0].Name != OperatorTap {
		t.Error(test.DiffMessage(ops[0].Name, OperatorTap, "returned operator must have correct name"))
	}
}

func TestGetAggregationOperators_NoMatch(t *testing.T) {
	a := NewAggregation()
	a.Transform(func(c *ctx.Context, data any) any { return data })
	ops := a.GetAggregationOperators(OperatorTap)
	if len(ops) != 0 {
		t.Error(test.DiffMessage(len(ops), 0, "must return empty slice when no match"))
	}
}

func TestGetAggregationOperators_MultipleMatches(t *testing.T) {
	a := NewAggregation()
	a.Tap(func(c *ctx.Context, data any) any { return "e1" })
	a.Tap(func(c *ctx.Context, data any) any { return "e2" })
	ops := a.GetAggregationOperators(OperatorTap)
	if len(ops) != 2 {
		t.Error(test.DiffMessage(len(ops), 2, "must return all matching operators"))
	}
}

func TestGetAggregationOperators_EmptyAggregation(t *testing.T) {
	a := NewAggregation()
	ops := a.GetAggregationOperators(OperatorTransform)
	if len(ops) != 0 {
		t.Error(test.DiffMessage(len(ops), 0, "empty aggregation must return empty result"))
	}
}

func TestAggregate_NoOperators(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	result := a.Aggregate(nil)
	if result != "data" {
		t.Error(test.DiffMessage(result, "data", "no operators must return mainData unchanged"))
	}
}

func TestAggregate_NilData(t *testing.T) {
	a := NewAggregation()
	result := a.Aggregate(nil)
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "nil mainData must be returned as nil"))
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
		t.Error(test.DiffMessage(result, 1, "Transform must increment value"))
	}
	if len(order) != 2 || order[0] != "transform" || order[1] != "tap" {
		t.Error(test.DiffMessage(order, []string{"transform", "tap"}, "operators must run in registration order"))
	}
}

func TestAggregate_MultipleTransforms(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(0)
	a.Transform(func(c *ctx.Context, data any) any { return data.(int) + 1 })
	a.Transform(func(c *ctx.Context, data any) any { return data.(int) * 3 })
	result := a.Aggregate(nil)
	if result != 3 {
		t.Error(test.DiffMessage(result, 3, "(0+1)*3 = 3"))
	}
}
