package aggregation

import (
	"github.com/dangduoc08/ginject/ctx"
)

type AggregationOperator = func(*ctx.Context, any) any

type Aggregation struct {
	IsMainHandlerCalled bool
	InterceptorData     any
	mainData            any
	operators           map[string]AggregationOperator
}

func NewAggregation() *Aggregation {
	aggregation := new(Aggregation)
	aggregation.operators = make(map[string]AggregationOperator, 5)
	return aggregation
}

func (aggregation *Aggregation) Pipe(
	operators ...AggregationOperator,
) any {
	aggregation.IsMainHandlerCalled = true
	return nil
}

func (aggregation *Aggregation) SetMainData(d any) *Aggregation {
	aggregation.mainData = d
	return aggregation
}

// if Pointer return duplicate value
// this way won't be work
func (aggregation *Aggregation) setOperators(name string, op AggregationOperator) *Aggregation {
	if _, ok := aggregation.operators[name]; !ok {
		aggregation.operators[name] = op
	}

	return aggregation
}

func (aggregation *Aggregation) GetAggregationOperator(oprName string) AggregationOperator {
	if aggregationOperator, ok := aggregation.operators[oprName]; ok {
		return aggregationOperator
	}
	return nil
}

func (aggregation *Aggregation) Aggregate(c *ctx.Context) any {
	if operator, ok := aggregation.operators[OPERATOR_CONSUME]; ok {
		aggregation.mainData = operator(c, aggregation.mainData)
	}
	return aggregation.mainData
}
