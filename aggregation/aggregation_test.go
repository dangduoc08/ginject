package aggregation

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

func TestNewAggregation(t *testing.T) {
	a := NewAggregation()
	if a == nil {
		t.Error(testutils.DiffMessage(a, "non-nil *Aggregation", "NewAggregation should return non-nil"))
	}
	if a.operators == nil {
		t.Error(testutils.DiffMessage(a.operators, "non-nil map", "operators map should be initialized"))
	}
}

func TestSetMainData(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("hello")
	if a.mainData != "hello" {
		t.Error(testutils.DiffMessage(a.mainData, "hello", "SetMainData"))
	}
	a.SetMainData(42)
	if a.mainData != 42 {
		t.Error(testutils.DiffMessage(a.mainData, 42, "SetMainData override"))
	}
}

func TestSetMainData_Chaining(t *testing.T) {
	a := NewAggregation()
	result := a.SetMainData("val")
	if result != a {
		t.Error(testutils.DiffMessage(result, a, "SetMainData should return same *Aggregation"))
	}
}

func TestPipe(t *testing.T) {
	a := NewAggregation()
	if a.IsMainHandlerCalled {
		t.Error(testutils.DiffMessage(a.IsMainHandlerCalled, false, "IsMainHandlerCalled should start false"))
	}
	a.Pipe()
	if !a.IsMainHandlerCalled {
		t.Error(testutils.DiffMessage(a.IsMainHandlerCalled, true, "Pipe sets IsMainHandlerCalled"))
	}
}

func TestConsumeOperator(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("initial")

	called := false
	a.Consume(func(c *ctx.Context, data any) any {
		called = true
		return "transformed"
	})

	result := a.Aggregate(nil)
	if !called {
		t.Error(testutils.DiffMessage(called, true, "Consume operator should be called"))
	}
	if result != "transformed" {
		t.Error(testutils.DiffMessage(result, "transformed", "Aggregate returns consumed value"))
	}
}

func TestAggregate_NoOperators(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	result := a.Aggregate(nil)
	if result != "data" {
		t.Error(testutils.DiffMessage(result, "data", "Aggregate with no operators returns mainData"))
	}
}

func TestAggregate_NilData(t *testing.T) {
	a := NewAggregation()
	result := a.Aggregate(nil)
	if result != nil {
		t.Error(testutils.DiffMessage(result, nil, "Aggregate with nil mainData returns nil"))
	}
}

func TestGetAggregationOperator_Exists(t *testing.T) {
	a := NewAggregation()
	op := func(c *ctx.Context, data any) any { return data }
	a.Consume(op)
	got := a.GetAggregationOperator(OPERATOR_CONSUME)
	if got == nil {
		t.Error(testutils.DiffMessage(got, "non-nil operator", "GetAggregationOperator should find Consume"))
	}
}

func TestGetAggregationOperator_Missing(t *testing.T) {
	a := NewAggregation()
	got := a.GetAggregationOperator(OPERATOR_CONSUME)
	if got != nil {
		t.Error(testutils.DiffMessage(got, nil, "GetAggregationOperator missing key returns nil"))
	}
}

func TestSetOperators_NoOverwrite(t *testing.T) {
	a := NewAggregation()
	first := func(c *ctx.Context, data any) any { return "first" }
	second := func(c *ctx.Context, data any) any { return "second" }
	a.Consume(first)
	a.Consume(second)
	result := a.Aggregate(nil)
	if result != "first" {
		t.Error(testutils.DiffMessage(result, "first", "second Consume should not overwrite first"))
	}
}

func TestErrorOperator(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	called := false
	a.Error(func(c *ctx.Context, data any) any {
		called = true
		return data
	})
	result := a.Aggregate(nil)
	if called {
		t.Error(testutils.DiffMessage(called, false, "Error operator should not be called in Aggregate"))
	}
	if result != "data" {
		t.Error(testutils.DiffMessage(result, "data", "mainData unchanged when only Error operator"))
	}
}

func TestFirstOperator(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	a.First()
	result := a.Aggregate(nil)
	if result != "data" {
		t.Error(testutils.DiffMessage(result, "data", "First operator does not transform mainData"))
	}
}

func TestMapOperator_Registered(t *testing.T) {
	a := NewAggregation()
	op := func(c *ctx.Context, data any) any { return data }
	returned := a.Map(op)
	if returned == nil {
		t.Error(testutils.DiffMessage(returned, "non-nil", "Map should return the operator"))
	}
	if a.GetAggregationOperator(OPERATOR_MAP) == nil {
		t.Error(testutils.DiffMessage(nil, "non-nil", "Map operator should be stored"))
	}
}

func TestOfOperator_Registered(t *testing.T) {
	a := NewAggregation()
	op := func(c *ctx.Context, data any) any { return data }
	returned := a.Of(op)
	if returned == nil {
		t.Error(testutils.DiffMessage(returned, "non-nil", "Of should return the operator"))
	}
	if a.GetAggregationOperator(OPERATOR_OF) == nil {
		t.Error(testutils.DiffMessage(nil, "non-nil", "Of operator should be stored"))
	}
}
