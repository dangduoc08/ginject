package aggregation

type AggregationOperator = func(any) any

type Operator struct {
	Name        string
	Aggregation AggregationOperator
}

type Aggregation struct {
	Name                string
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

func (aggregation *Aggregation) setOperators(name string, op AggregationOperator) *Aggregation {
	aggregation.operators = append(aggregation.operators, Operator{
		Name:        name,
		Aggregation: op,
	})

	return aggregation
}

func (aggregation *Aggregation) Aggregate() any {
	for _, operator := range aggregation.operators {
		switch operator.Name {
		case OperatorTransform:
			aggregation.mainData = operator.Aggregation(aggregation.mainData)
		case OperatorTap:
			operator.Aggregation(aggregation.mainData)
		case OperatorFilter:
			result := operator.Aggregation(aggregation.mainData)
			if result == nil {
				return nil
			}
			aggregation.mainData = result
		}
	}

	return aggregation.mainData
}
