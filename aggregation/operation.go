package aggregation

const (
	OPERATOR_MAP     = "Map"
	OPERATOR_OF      = "Of"
	OPERATOR_CONSUME = "Consume"
	OPERATOR_FIRST   = "Fiest"
)

func (aggregation *Aggregation) Map(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OPERATOR_MAP, opr)
	return opr
}

func (aggregation *Aggregation) Of(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OPERATOR_OF, opr)
	return opr
}

func (aggregation *Aggregation) Consume(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OPERATOR_CONSUME, opr)
	return opr
}

func (aggregation *Aggregation) First() AggregationOperator {
	aggregation.setOperators(OPERATOR_FIRST, nil)
	return nil
}
