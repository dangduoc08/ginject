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
	aggregation.operators = map[string]AggregationOperator{}

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

	// handle operators
	for name, operator := range aggregation.operators {
		// kind := reflect.TypeOf(aggregation.data).Kind()
		// switch kind {
		// case
		// 	reflect.Map,
		// 	reflect.Slice,
		// 	reflect.Struct,
		// 	reflect.Interface:

		// 	// iterable
		// case
		// 	reflect.Bool,
		// 	reflect.Int,
		// 	reflect.Int8,
		// 	reflect.Int16,
		// 	reflect.Int32,
		// 	reflect.Int64,
		// 	reflect.Uint,
		// 	reflect.Uint8,
		// 	reflect.Uint16,
		// 	reflect.Uint32,
		// 	reflect.Uint64,
		// 	reflect.Float32,
		// 	reflect.Float64,
		// 	reflect.Complex64,
		// 	reflect.Complex128:

		// 	// to number
		// case
		// 	reflect.Pointer,
		// 	reflect.UnsafePointer:

		// 	// to pointer
		// case
		// 	reflect.String:

		// 	// to string
		// case
		// 	reflect.Func:

		// 	// to pointer address as well
		// 	// c.Text(data.Type().String())
		// }

		switch name {
		// case OPERATOR_OF:
		// 	aggregation.mainData = operator(aggregation.mainData)

		// case OPERATOR_MAP:
		// 	aggregation.mainData = operator(aggregation.mainData)

		case OPERATOR_CONSUME:
			aggregation.mainData = operator(c, aggregation.mainData)

		case OPERATOR_ERROR:
			// error operator will be handle inside recover
			continue

		case OPERATOR_FIRST:
			// aggregation.mainData = aggregation.mainData
		}

	}

	return aggregation.mainData
}
