//go:build !race

package aggregation

import (
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

func TestAggregation_ConcurrentTransformAndGetAggregationOperators_RequiresExternalLock(t *testing.T) {
	a := NewAggregation()
	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			a.Transform(func(c *ctx.Context, data any) any { return data })
			_ = a.GetAggregationOperators(OperatorTransform)
		}()
	}
	wg.Wait()
}
