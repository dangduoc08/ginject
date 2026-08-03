package aggregation

import (
	"testing"
)

var benchNoop = func(data any) any { return data }
var benchPredicate = func(data any) bool { return true }

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


func BenchmarkSetOperators_Two(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := NewAggregation()
		a.Transform(benchNoop)
		a.Tap(benchNoop)
	}
}

func BenchmarkAggregate_Filter(b *testing.B) {
	a := NewAggregation()
	a.SetMainData(10)
	a.Filter(benchPredicate)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate()
	}
}

func BenchmarkAggregate_Filter_False(b *testing.B) {
	a := NewAggregation()
	a.SetMainData(10)
	a.Filter(func(data any) bool { return false })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate()
	}
}

func BenchmarkAggregate_MultipleFilters(b *testing.B) {
	a := NewAggregation()
	a.SetMainData(30)
	a.Filter(benchPredicate)
	a.Filter(benchPredicate)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate()
	}
}

func BenchmarkAggregate_Filter_And_Transform(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Filter(benchPredicate)
	a.Transform(benchNoop)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate()
	}
}
