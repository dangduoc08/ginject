package aggregation

const (
	OperatorTransform = "Transform"
	OperatorTap       = "Tap"
)

func (aggregation *Aggregation) Transform(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OperatorTransform, opr)
	return opr
}

func (aggregation *Aggregation) Tap(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OperatorTap, opr)
	return opr
}
