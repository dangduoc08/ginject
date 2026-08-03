package aggregation

import (
	"testing"

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

func TestPipe_WithFilterTransformTap(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(10)
	result := a.Pipe(
		a.Filter(func(d any) bool { return d.(int) > 5 }),
		a.Transform(func(d any) any { return d.(int) * 2 }),
		a.Tap(func(d any) any { return nil }),
	)
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "Pipe must return nil"))
	}
	if !a.IsMainHandlerCalled {
		t.Error(test.DiffMessage(a.IsMainHandlerCalled, true, "Pipe must set IsMainHandlerCalled"))
	}
}

func TestTransform_RegisteredAndApplied(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("original")
	called := false
	a.Transform(func(data any) any {
		called = true
		return "transformed"
	})
	result := a.Aggregate()
	if !called {
		t.Error(test.DiffMessage(called, true, "Transform operator must be called"))
	}
	if result != "transformed" {
		t.Error(test.DiffMessage(result, "transformed", "Transform must update mainData"))
	}
}

func TestTransform_ReturnsOperator(t *testing.T) {
	a := NewAggregation()
	noop := func(data any) any { return data }
	got := a.Transform(noop)
	if got == nil {
		t.Error(test.DiffMessage(got, "non-nil", "Transform must return the operator"))
	}
}

func TestTap_CalledButDoesNotTransform(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("original")
	called := false
	a.Tap(func(data any) any {
		called = true
		return "tap-result"
	})
	result := a.Aggregate()
	if !called {
		t.Error(test.DiffMessage(called, true, "Tap operator must be called"))
	}
	if result != "original" {
		t.Error(test.DiffMessage(result, "original", "Tap must not change mainData"))
	}
}


func TestAggregate_NoOperators(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("data")
	result := a.Aggregate()
	if result != "data" {
		t.Error(test.DiffMessage(result, "data", "no operators must return mainData unchanged"))
	}
}

func TestAggregate_NilData(t *testing.T) {
	a := NewAggregation()
	result := a.Aggregate()
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "nil mainData must be returned as nil"))
	}
}

func TestAggregate_TransformAndTap_Order(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(0)
	order := []string{}
	a.Transform(func(data any) any {
		order = append(order, "transform")
		return data.(int) + 1
	})
	a.Tap(func(data any) any {
		order = append(order, "tap")
		return data
	})
	result := a.Aggregate()
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
	a.Transform(func(data any) any { return data.(int) + 1 })
	a.Transform(func(data any) any { return data.(int) * 3 })
	result := a.Aggregate()
	if result != 3 {
		t.Error(test.DiffMessage(result, 3, "(0+1)*3 = 3"))
	}
}

func TestFilter_PredicateTrue(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(10)
	called := false
	a.Filter(func(data any) bool {
		called = true
		return data.(int) > 5
	})
	result := a.Aggregate()

	if !called {
		t.Error(test.DiffMessage(called, true, "predicate must be called"))
	}
	if result != 10 {
		t.Error(test.DiffMessage(result, 10, "filter must return data when predicate is true"))
	}
}

func TestFilter_PredicateFalse(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(3)
	called := false
	a.Filter(func(data any) bool {
		called = true
		return data.(int) > 5
	})
	result := a.Aggregate()

	if !called {
		t.Error(test.DiffMessage(called, true, "predicate must be called"))
	}
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "filter must return nil when predicate is false"))
	}
}

func TestFilter_ReturnsOperator(t *testing.T) {
	a := NewAggregation()
	pred := func(data any) bool { return true }
	returned := a.Filter(pred)

	if returned == nil {
		t.Error(test.DiffMessage(returned, "non-nil", "Filter must return AggregationOperator"))
	}
}

func TestFilter_StopsTransform(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(3)
	transformCalled := false

	a.Filter(func(data any) bool {
		return data.(int) > 5
	})
	a.Transform(func(data any) any {
		transformCalled = true
		return data
	})
	result := a.Aggregate()

	if transformCalled {
		t.Error(test.DiffMessage(transformCalled, false, "Transform must not be called after filter returns nil"))
	}
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "result must be nil when filter fails"))
	}
}

func TestFilter_MultipleFilters(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(30)

	a.Filter(func(data any) bool {
		return data.(int) >= 18
	})
	a.Filter(func(data any) bool {
		return data.(int) <= 65
	})
	result := a.Aggregate()

	if result != 30 {
		t.Error(test.DiffMessage(result, 30, "both filters should pass"))
	}
}

func TestFilter_MultipleFilters_FailsSecond(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(70)

	a.Filter(func(data any) bool {
		return data.(int) >= 18
	})
	a.Filter(func(data any) bool {
		return data.(int) <= 65
	})
	result := a.Aggregate()

	if result != nil {
		t.Error(test.DiffMessage(result, nil, "second filter should fail"))
	}
}

func TestFilter_WithTap(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(10)
	tapCalled := false

	a.Filter(func(data any) bool {
		return data.(int) > 5
	})
	a.Tap(func(data any) any {
		tapCalled = true
		return nil
	})
	result := a.Aggregate()

	if !tapCalled {
		t.Error(test.DiffMessage(tapCalled, true, "Tap must be called after filter succeeds"))
	}
	if result != 10 {
		t.Error(test.DiffMessage(result, 10, "Tap must not change data"))
	}
}

func TestFilter_WithTap_FilterFails(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(3)
	tapCalled := false

	a.Filter(func(data any) bool {
		return data.(int) > 5
	})
	a.Tap(func(data any) any {
		tapCalled = true
		return nil
	})
	result := a.Aggregate()

	if tapCalled {
		t.Error(test.DiffMessage(tapCalled, false, "Tap must not be called when filter fails"))
	}
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "result must be nil"))
	}
}

func TestFilter_ThenTransform(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(10)

	a.Filter(func(data any) bool {
		return data.(int) > 5
	})
	a.Transform(func(data any) any {
		return data.(int) * 2
	})
	result := a.Aggregate()

	if result != 20 {
		t.Error(test.DiffMessage(result, 20, "filter pass, transform should execute"))
	}
}

func TestFilter_WithNilData(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(nil)

	a.Filter(func(data any) bool {
		return data == nil
	})
	result := a.Aggregate()

	if result != nil {
		t.Error(test.DiffMessage(result, nil, "filter with nil input should handle gracefully"))
	}
}

func TestFilter_ZeroValue(t *testing.T) {
	a := NewAggregation()
	a.SetMainData(0)

	a.Filter(func(data any) bool {
		return data.(int) == 0
	})
	result := a.Aggregate()

	if result != 0 {
		t.Error(test.DiffMessage(result, 0, "filter should distinguish zero from nil"))
	}
}

func TestFilter_EmptyString(t *testing.T) {
	a := NewAggregation()
	a.SetMainData("")

	a.Filter(func(data any) bool {
		return data == ""
	})
	result := a.Aggregate()

	if result != "" {
		t.Error(test.DiffMessage(result, "", "filter should handle empty string"))
	}
}

