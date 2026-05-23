package aggregation

const (
	OPERATOR_TRANSFORM              = "Transform"
	OPERATOR_TAP                    = "Tap"
	OPERATOR_ERROR                  = "Error"
	ERROR_AGGREGATION_CTX_VALUE_KEY = "ErrorAggregationOperators"
)

func (aggregation *Aggregation) Transform(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OPERATOR_TRANSFORM, opr)
	return opr
}

func (aggregation *Aggregation) Tap(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OPERATOR_TAP, opr)
	return opr
}

func (aggregation *Aggregation) Error(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OPERATOR_ERROR, opr)
	return opr
}
