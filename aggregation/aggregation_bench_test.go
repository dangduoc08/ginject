package aggregation

import (
	"testing"
)

var benchNoop = func(data any) any { return data }

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
		a.Aggregate()
	}
}

func BenchmarkAggregate_Transform(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Transform(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate()
	}
}

func BenchmarkAggregate_AllOperators(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Transform(benchNoop)
	a.Tap(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate()
	}
}

func BenchmarkGetAggregationOperators_Hit(b *testing.B) {
	a := NewAggregation()
	a.Transform(benchNoop)
	a.Tap(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.GetAggregationOperators(OperatorTap)
	}
}

func BenchmarkGetAggregationOperators_Miss(b *testing.B) {
	a := NewAggregation()
	a.Transform(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.GetAggregationOperators(OperatorTap)
	}
}

func BenchmarkSetOperators_Two(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := NewAggregation()
		a.Transform(benchNoop)
		a.Tap(benchNoop)
	}
}
