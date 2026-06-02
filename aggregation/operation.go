package aggregation

const (
	OperatorTransform              = "Transform"
	OperatorTap                    = "Tap"
	OperatorError                  = "Error"
	ErrorAggregationCtxValueKey = "ErrorAggregationOperators"
)

func (aggregation *Aggregation) Transform(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OperatorTransform, opr)
	return opr
}

func (aggregation *Aggregation) Tap(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OperatorTap, opr)
	return opr
}

func (aggregation *Aggregation) Error(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OperatorError, opr)
	return opr
}
