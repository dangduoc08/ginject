package aggregation

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

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

func BenchmarkAggregate_Consume(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	a.Consume(func(c *ctx.Context, data any) any { return data })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate(nil)
	}
}

func BenchmarkAggregate_AllOperators(b *testing.B) {
	a := NewAggregation()
	a.SetMainData("data")
	noop := func(c *ctx.Context, data any) any { return data }
	a.Consume(noop)
	a.Error(noop)
	a.Map(noop)
	a.Of(noop)
	a.First()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Aggregate(nil)
	}
}

func BenchmarkGetAggregationOperator_Hit(b *testing.B) {
	a := NewAggregation()
	a.Consume(func(c *ctx.Context, data any) any { return data })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.GetAggregationOperator(OPERATOR_CONSUME)
	}
}

func BenchmarkGetAggregationOperator_Miss(b *testing.B) {
	a := NewAggregation()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.GetAggregationOperator(OPERATOR_CONSUME)
	}
}
