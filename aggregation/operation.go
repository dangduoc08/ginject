package aggregation

const (
	OperatorTransform = "Transform"
	OperatorTap       = "Tap"
	OperatorFilter    = "Filter"
)

func (aggregation *Aggregation) Transform(opr func(any) any) AggregationOperator {
	aggregation.setOperators(OperatorTransform, opr)
	return opr
}

func (aggregation *Aggregation) Tap(opr func(any) any) AggregationOperator {
	aggregation.setOperators(OperatorTap, opr)
	return opr
}

func (aggregation *Aggregation) Filter(predicate func(any) bool) AggregationOperator {
	op := func(data any) any {
		if predicate(data) {
			return data
		}
		return nil
	}
	aggregation.setOperators(OperatorFilter, op)
	return op
}
