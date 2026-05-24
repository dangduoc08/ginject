package aggregation

import (
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

const (
	OPERATOR_TRANSFORM              = "Transform"
	OPERATOR_TAP                    = "Tap"
	OPERATOR_TIMEOUT                = "Timeout"
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

// Timeout registers a timeout check in the aggregation pipeline and returns
// the aggregation for chaining. Two checks are applied in order:
//  1. Real context cancellation (client disconnect, framework deadline).
//  2. Wall-clock elapsed time since c.Timestamp >= d.
func (aggregation *Aggregation) Timeout(d time.Duration) *Aggregation {
	opr := func(c *ctx.Context, data any) any {
		if c == nil {
			return data
		}

		// Real context cancellation check first.
		select {
		case <-c.GetExec().Done():
			panic(exception.RequestTimeoutException("request cancelled or deadline exceeded"))
		default:
		}

		// Wall-clock SLA check.
		if !c.Timestamp.IsZero() && time.Since(c.Timestamp) >= d {
			panic(exception.RequestTimeoutException("request timeout"))
		}

		return data
	}

	aggregation.setOperators(OPERATOR_TIMEOUT, opr)
	return aggregation
}

func (aggregation *Aggregation) Error(opr AggregationOperator) AggregationOperator {
	aggregation.setOperators(OPERATOR_ERROR, opr)
	return opr
}
