package aggregation

import (
	"github.com/dangduoc08/ginject/ctx"
)

type AggregationOperator = func(*ctx.Context, any) any

type Operator struct {
	Name        string
	Aggregation AggregationOperator
}

type Aggregation struct {
	IsMainHandlerCalled bool
	InterceptorData     any
	mainData            any
	operators           []Operator
}

func NewAggregation() *Aggregation {
	return &Aggregation{}
}

func (aggregation *Aggregation) Pipe(operators ...AggregationOperator) any {
	aggregation.IsMainHandlerCalled = true

	return nil
}

func (aggregation *Aggregation) SetMainData(d any) *Aggregation {
	aggregation.mainData = d
	return aggregation
}

// Use on app.go where it need to get error aggregation
func (aggregation *Aggregation) GetAggregationOperators(oprName string) []Operator {
	var result []Operator
	for _, op := range aggregation.operators {
		if op.Name == oprName {
			result = append(result, op)
		}
	}
	return result
}

func (aggregation *Aggregation) setOperators(name string, op AggregationOperator) *Aggregation {
	aggregation.operators = append(aggregation.operators, Operator{
		Name:        name,
		Aggregation: op,
	})

	return aggregation
}

func (aggregation *Aggregation) Aggregate(c *ctx.Context) any {
	for _, operator := range aggregation.operators {
		switch operator.Name {
		case OperatorTransform:
			aggregation.mainData = operator.Aggregation(c, aggregation.mainData)
		case OperatorTap:
			operator.Aggregation(c, aggregation.mainData)
		}
	}

	return aggregation.mainData
}
