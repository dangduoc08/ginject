package throttler

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/memcache"
	"github.com/dangduoc08/ginject/modules/cache"
)

type Strategy int

const (
	FixedWindow Strategy = iota
	SlidingWindow
	TokenBucket
)

type Throttler struct {
	Limit    int64
	TTL      time.Duration
	Strategy Strategy
	KeyFunc  func(*ctx.HTTPContext) string
	Backend  cache.Cache
}

func (g Throttler) NewGuard() Throttler {
	if g.Limit <= 0 {
		g.Limit = 100
	}
	if g.TTL <= 0 {
		g.TTL = time.Minute
	}
	if g.KeyFunc == nil {
		g.KeyFunc = defaultThrottlerKeyFunc
	}
	if g.Backend == nil {
		g.Backend = memcache.NewMemoryCache()
	}
	return g
}

func (g Throttler) CanActivate(c *ctx.HTTPContext) bool {
	res := g.check(c)

	h := c.ResponseWriter.Header()
	h.Set("X-RateLimit-Limit", strconv.FormatInt(res.limit, 10))
	h.Set("X-RateLimit-Remaining", strconv.FormatInt(res.remaining, 10))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(res.resetAt, 10))

	if !res.isAllowed {
		retryAfter := max(res.resetAt-time.Now().Unix(), 0)
		h.Set("Retry-After", strconv.FormatInt(retryAfter, 10))
		panic(exception.TooManyRequestsException("Too Many Requests"))
	}
	return true
}

type rateLimitResult struct {
	isAllowed   bool
	limit     int64
	remaining int64
	resetAt   int64
}

func (g Throttler) check(c *ctx.HTTPContext) rateLimitResult {
	key := g.KeyFunc(c)
	switch g.Strategy {
	case SlidingWindow:
		return g.slidingWindow(c.Context(), key)
	case TokenBucket:
		return g.tokenBucket(c.Context(), key)
	default:
		return g.fixedWindow(c.Context(), key)
	}
}

func (g Throttler) fixedWindow(bgCtx context.Context, key string) rateLimitResult {
	windowSec := max(int64(g.TTL.Seconds()), 1)
	nowSec := time.Now().Unix()
	windowID := nowSec / windowSec
	cacheKey := "rl:fw:" + key + ":" + strconv.FormatInt(windowID, 10)
	resetAt := (windowID + 1) * windowSec

	var count int64 = 1
	if raw, ok := g.Backend.Get(bgCtx, cacheKey); ok && len(raw) == 8 {
		count = int64(binary.BigEndian.Uint64(raw)) + 1
	}

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(count))
	ttlRemaining := max(time.Duration(resetAt-nowSec)*time.Second, time.Second)
	_ = g.Backend.Set(bgCtx, cacheKey, buf[:], ttlRemaining)

	remaining := max(g.Limit-count, 0)
	return rateLimitResult{
		isAllowed:   count <= g.Limit,
		limit:     g.Limit,
		remaining: remaining,
		resetAt:   resetAt,
	}
}

func (g Throttler) slidingWindow(bgCtx context.Context, key string) rateLimitResult {
	windowSec := max(int64(g.TTL.Seconds()), 1)
	nowSec := time.Now().Unix()
	currWindowID := nowSec / windowSec
	prevWindowID := currWindowID - 1

	currKey := "rl:sw:c:" + key + ":" + strconv.FormatInt(currWindowID, 10)
	prevKey := "rl:sw:p:" + key + ":" + strconv.FormatInt(prevWindowID, 10)
	resetAt := (currWindowID + 1) * windowSec

	elapsedInWindow := nowSec - currWindowID*windowSec
	ratio := float64(elapsedInWindow) / float64(windowSec)

	var prevCount int64
	if raw, ok := g.Backend.Get(bgCtx, prevKey); ok && len(raw) == 8 {
		prevCount = int64(binary.BigEndian.Uint64(raw))
	}

	var currCount int64 = 1
	if raw, ok := g.Backend.Get(bgCtx, currKey); ok && len(raw) == 8 {
		currCount = int64(binary.BigEndian.Uint64(raw)) + 1
	}

	weighted := int64(math.Round(float64(prevCount)*(1-ratio))) + currCount

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(currCount))
	_ = g.Backend.Set(bgCtx, currKey, buf[:], time.Duration(2*windowSec)*time.Second)

	remaining := max(g.Limit-weighted, 0)
	return rateLimitResult{
		isAllowed:   weighted <= g.Limit,
		limit:     g.Limit,
		remaining: remaining,
		resetAt:   resetAt,
	}
}

// token bucket state layout: [8 bytes float64 tokens][8 bytes int64 last_refill_ns]
func (g Throttler) tokenBucket(bgCtx context.Context, key string) rateLimitResult {
	cacheKey := "rl:tb:" + key
	refillRate := float64(g.Limit) / float64(g.TTL.Nanoseconds())

	now := time.Now().UnixNano()
	var tokens float64

	if raw, ok := g.Backend.Get(bgCtx, cacheKey); ok && len(raw) == 16 {
		tokens = math.Float64frombits(binary.BigEndian.Uint64(raw[:8]))
		lastRefill := int64(binary.BigEndian.Uint64(raw[8:]))
		elapsed := float64(now - lastRefill)
		tokens = math.Min(float64(g.Limit), tokens+elapsed*refillRate)
	} else {
		tokens = float64(g.Limit)
	}

	isAllowed := tokens >= 1.0
	if isAllowed {
		tokens--
	}

	var buf [16]byte
	binary.BigEndian.PutUint64(buf[:8], math.Float64bits(tokens))
	binary.BigEndian.PutUint64(buf[8:], uint64(now))
	_ = g.Backend.Set(bgCtx, cacheKey, buf[:], g.TTL*2)

	var resetAt int64
	if !isAllowed && refillRate > 0 {
		nsUntilNext := (1.0 - tokens) / refillRate
		resetAt = time.Unix(0, now+int64(nsUntilNext)).Unix()
	} else {
		resetAt = time.Unix(0, now+int64(g.TTL)).Unix()
	}

	return rateLimitResult{
		isAllowed:   isAllowed,
		limit:     g.Limit,
		remaining: int64(math.Floor(tokens)),
		resetAt:   resetAt,
	}
}

func defaultThrottlerKeyFunc(c *ctx.HTTPContext) string {
	if xrip := c.Request.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(c.RemoteAddr)
	if err != nil || host == "" {
		return c.RemoteAddr
	}
	return host
}
