package throttler

import (
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/memorycache"
	"github.com/dangduoc08/ginject/modules/cache"
)

type WSThrottler struct {
	Backend  cache.Cache
	KeyFunc  func(*ctx.WSContext) string
	Limit    int64
	TTL      time.Duration
	Strategy Strategy
}

func ConnScopedKeyFunc(c *ctx.WSContext) string {
	return "conn:" + c.ConnID()
}

func ConnTopicScopedKeyFunc(c *ctx.WSContext) string {
	return "conn:" + c.ConnID() + ":" + c.Operation() + ":" + c.Topic()
}

func (g WSThrottler) NewGuard() WSThrottler {
	if g.Limit <= 0 {
		g.Limit = 100
	}
	if g.TTL <= 0 {
		g.TTL = time.Minute
	}
	if g.KeyFunc == nil {
		g.KeyFunc = ConnScopedKeyFunc
	}
	if g.Backend == nil {
		g.Backend = memorycache.NewMemoryCache()
	}
	return g
}

func (g WSThrottler) CanActivate(c *ctx.WSContext) bool {
	inner := Throttler{
		Backend:  g.Backend,
		Limit:    g.Limit,
		TTL:      g.TTL,
		Strategy: g.Strategy,
	}

	key := g.KeyFunc(c)
	var res rateLimitResult
	switch g.Strategy {
	case SlidingWindow:
		res = inner.slidingWindow(c.Context(), key)
	case TokenBucket:
		res = inner.tokenBucket(c.Context(), key)
	default:
		res = inner.fixedWindow(c.Context(), key)
	}

	if !res.isAllowed {
		panic(exception.PolicyViolationException("rate limit exceeded for " + c.Operation() + " on " + c.Topic()))
	}

	return true
}
