package guards

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/modules/cache"
)

type Strategy int

const (
	FixedWindow   Strategy = iota
	SlidingWindow Strategy = iota
	TokenBucket   Strategy = iota
)

type ThrottlerOptions struct {
	Limit    int64
	TTL      time.Duration
	Strategy Strategy
	KeyFunc  func(*ctx.Context) string
	Store    cache.Cache
}

type ThrottlerGuard struct {
	Limit    int64
	TTL      time.Duration
	Strategy Strategy
	KeyFunc  func(*ctx.Context) string
	Store    cache.Cache
}

func NewThrottler(opts ThrottlerOptions) ThrottlerGuard {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.TTL <= 0 {
		opts.TTL = time.Minute
	}
	if opts.KeyFunc == nil {
		opts.KeyFunc = defaultThrottlerKeyFunc
	}
	if opts.Store == nil {
		opts.Store = cache.NewMemoryCache()
	}
	return ThrottlerGuard(opts)
}

func (g ThrottlerGuard) CanActivate(c *ctx.Context) bool {
	res := g.check(c)

	h := c.ResponseWriter.Header()
	h.Set("X-RateLimit-Limit", strconv.FormatInt(res.limit, 10))
	h.Set("X-RateLimit-Remaining", strconv.FormatInt(res.remaining, 10))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(res.resetAt, 10))

	if !res.allowed {
		retryAfter := res.resetAt - time.Now().Unix()
		if retryAfter < 0 {
			retryAfter = 0
		}
		h.Set("Retry-After", strconv.FormatInt(retryAfter, 10))
		panic(exception.TooManyRequestsException("Too Many Requests"))
	}
	return true
}

type rateLimitResult struct {
	allowed   bool
	limit     int64
	remaining int64
	resetAt   int64
}

func (g ThrottlerGuard) check(c *ctx.Context) rateLimitResult {
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

func (g ThrottlerGuard) fixedWindow(bgCtx context.Context, key string) rateLimitResult {
	windowSec := int64(g.TTL.Seconds())
	if windowSec < 1 {
		windowSec = 1
	}
	nowSec := time.Now().Unix()
	windowID := nowSec / windowSec
	cacheKey := fmt.Sprintf("rl:fw:%s:%d", key, windowID)
	resetAt := (windowID + 1) * windowSec

	var count int64 = 1
	if raw, ok := g.Store.Get(bgCtx, cacheKey); ok && len(raw) == 8 {
		count = int64(binary.BigEndian.Uint64(raw)) + 1
	}

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(count))
	ttlRemaining := time.Duration(resetAt-nowSec) * time.Second
	if ttlRemaining < time.Second {
		ttlRemaining = time.Second
	}
	_ = g.Store.Set(bgCtx, cacheKey, buf, ttlRemaining)

	remaining := g.Limit - count
	if remaining < 0 {
		remaining = 0
	}
	return rateLimitResult{
		allowed:   count <= g.Limit,
		limit:     g.Limit,
		remaining: remaining,
		resetAt:   resetAt,
	}
}

func (g ThrottlerGuard) slidingWindow(bgCtx context.Context, key string) rateLimitResult {
	windowSec := int64(g.TTL.Seconds())
	if windowSec < 1 {
		windowSec = 1
	}
	nowSec := time.Now().Unix()
	currWindowID := nowSec / windowSec
	prevWindowID := currWindowID - 1

	currKey := fmt.Sprintf("rl:sw:c:%s:%d", key, currWindowID)
	prevKey := fmt.Sprintf("rl:sw:p:%s:%d", key, prevWindowID)
	resetAt := (currWindowID + 1) * windowSec

	elapsedInWindow := nowSec - currWindowID*windowSec
	ratio := float64(elapsedInWindow) / float64(windowSec)

	var prevCount int64
	if raw, ok := g.Store.Get(bgCtx, prevKey); ok && len(raw) == 8 {
		prevCount = int64(binary.BigEndian.Uint64(raw))
	}

	var currCount int64 = 1
	if raw, ok := g.Store.Get(bgCtx, currKey); ok && len(raw) == 8 {
		currCount = int64(binary.BigEndian.Uint64(raw)) + 1
	}

	weighted := int64(math.Round(float64(prevCount)*(1-ratio))) + currCount

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(currCount))
	_ = g.Store.Set(bgCtx, currKey, buf, time.Duration(2*windowSec)*time.Second)

	remaining := g.Limit - weighted
	if remaining < 0 {
		remaining = 0
	}
	return rateLimitResult{
		allowed:   weighted <= g.Limit,
		limit:     g.Limit,
		remaining: remaining,
		resetAt:   resetAt,
	}
}

// token bucket state layout: [8 bytes float64 tokens][8 bytes int64 last_refill_ns]
func (g ThrottlerGuard) tokenBucket(bgCtx context.Context, key string) rateLimitResult {
	cacheKey := fmt.Sprintf("rl:tb:%s", key)
	refillRate := float64(g.Limit) / float64(g.TTL.Nanoseconds())

	now := time.Now().UnixNano()
	var tokens float64

	if raw, ok := g.Store.Get(bgCtx, cacheKey); ok && len(raw) == 16 {
		tokens = math.Float64frombits(binary.BigEndian.Uint64(raw[:8]))
		lastRefill := int64(binary.BigEndian.Uint64(raw[8:]))
		elapsed := float64(now - lastRefill)
		tokens = math.Min(float64(g.Limit), tokens+elapsed*refillRate)
	} else {
		tokens = float64(g.Limit)
	}

	allowed := tokens >= 1.0
	if allowed {
		tokens--
	}

	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[:8], math.Float64bits(tokens))
	binary.BigEndian.PutUint64(buf[8:], uint64(now))
	_ = g.Store.Set(bgCtx, cacheKey, buf, g.TTL*2)

	var resetAt int64
	if !allowed && refillRate > 0 {
		nsUntilNext := (1.0 - tokens) / refillRate
		resetAt = time.Unix(0, now+int64(nsUntilNext)).Unix()
	} else {
		resetAt = time.Unix(0, now).Add(g.TTL).Unix()
	}

	return rateLimitResult{
		allowed:   allowed,
		limit:     g.Limit,
		remaining: int64(math.Floor(tokens)),
		resetAt:   resetAt,
	}
}

func defaultThrottlerKeyFunc(c *ctx.Context) string {
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
