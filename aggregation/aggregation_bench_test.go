package aggregation

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

var benchNoop = func(c *ctx.Context, data any) any { return data }

func BenchmarkNewAggregation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewAggregation()
	}
}

func BenchmarkAggregate_NoOperators(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate(nil)
	}
}

func BenchmarkAggregate_Transform(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Transform(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate(nil)
	}
}

func BenchmarkAggregate_AllOperators(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Transform(benchNoop)
	a.Tap(benchNoop)
	a.Error(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate(nil)
	}
}

func BenchmarkGetAggregationOperators_Hit(b *testing.B) {
	a := NewAggregation()
	a.Transform(benchNoop)
	a.Error(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.GetAggregationOperators(OPERATOR_ERROR)
	}
}

func BenchmarkGetAggregationOperators_Miss(b *testing.B) {
	a := NewAggregation()
	a.Transform(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.GetAggregationOperators(OPERATOR_ERROR)
	}
}

func BenchmarkSetOperators_Three(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := NewAggregation()
		a.Transform(benchNoop)
		a.Tap(benchNoop)
		a.Error(benchNoop)
	}
}
