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
	"github.com/dangduoc08/ginject/memorycache"
	"github.com/dangduoc08/ginject/modules/cache"
)

type Strategy int

const (
	FixedWindow Strategy = iota
	SlidingWindow
	TokenBucket
)

type Throttler struct {
	Backend           cache.Cache
	KeyFunc           func(*ctx.HTTPContext) string
	Limit             int64
	TTL               time.Duration
	Strategy          Strategy
	TrustProxyHeaders bool
}

func (g Throttler) NewGuard() Throttler {
	if g.Limit <= 0 {
		g.Limit = 100
	}
	if g.TTL <= 0 {
		g.TTL = time.Minute
	}
	if g.KeyFunc == nil {
		if g.TrustProxyHeaders {
			g.KeyFunc = proxyAwareThrottlerKeyFunc
		} else {
			g.KeyFunc = remoteAddrThrottlerKeyFunc
		}
	}
	if g.Backend == nil {
		g.Backend = memorycache.NewMemoryCache()
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
	isAllowed bool
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
	ttlRemaining := max(time.Duration(resetAt-nowSec)*time.Second, time.Second)

	count, err := incrementCounter(bgCtx, g.Backend, cacheKey, ttlRemaining)
	if err != nil {
		count = 1
	}

	remaining := max(g.Limit-count, 0)
	return rateLimitResult{
		isAllowed: count <= g.Limit,
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

	currCount, err := incrementCounter(bgCtx, g.Backend, currKey, time.Duration(2*windowSec)*time.Second)
	if err != nil {
		currCount = 1
	}

	weighted := int64(math.Round(float64(prevCount)*(1-ratio))) + currCount

	remaining := max(g.Limit-weighted, 0)
	return rateLimitResult{
		isAllowed: weighted <= g.Limit,
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
	setTTL := g.TTL * 2

	if am, ok := g.Backend.(cache.AtomicMutator); ok {
		var result rateLimitResult
		_, err := am.Mutate(bgCtx, cacheKey, func(old []byte, exists bool) ([]byte, time.Duration) {
			var encoded []byte
			result, encoded = computeTokenBucket(old, exists, now, g.Limit, g.TTL, refillRate)
			return encoded, setTTL
		})
		if err == nil {
			return result
		}
	}

	raw, exists := g.Backend.Get(bgCtx, cacheKey)
	result, encoded := computeTokenBucket(raw, exists, now, g.Limit, g.TTL, refillRate)
	_ = g.Backend.Set(bgCtx, cacheKey, encoded, setTTL)
	return result
}

func computeTokenBucket(raw []byte, exists bool, now int64, limit int64, ttl time.Duration, refillRate float64) (rateLimitResult, []byte) {
	var tokens float64
	if exists && len(raw) == 16 {
		tokens = math.Float64frombits(binary.BigEndian.Uint64(raw[:8]))
		lastRefill := int64(binary.BigEndian.Uint64(raw[8:]))
		elapsed := float64(now - lastRefill)
		tokens = math.Min(float64(limit), tokens+elapsed*refillRate)
	} else {
		tokens = float64(limit)
	}

	isAllowed := tokens >= 1.0
	if isAllowed {
		tokens--
	}

	var resetAt int64
	if !isAllowed && refillRate > 0 {
		nsUntilNext := (1.0 - tokens) / refillRate
		resetAt = time.Unix(0, now+int64(nsUntilNext)).Unix()
	} else {
		resetAt = time.Unix(0, now+int64(ttl)).Unix()
	}

	result := rateLimitResult{
		isAllowed: isAllowed,
		limit:     limit,
		remaining: int64(math.Floor(tokens)),
		resetAt:   resetAt,
	}

	var buf [16]byte
	binary.BigEndian.PutUint64(buf[:8], math.Float64bits(tokens))
	binary.BigEndian.PutUint64(buf[8:], uint64(now))
	return result, buf[:]
}

func incrementCounter(bgCtx context.Context, backend cache.Cache, key string, ttl time.Duration) (int64, error) {
	mutate := func(old []byte, exists bool) ([]byte, time.Duration) {
		var count int64 = 1
		if exists && len(old) == 8 {
			count = int64(binary.BigEndian.Uint64(old)) + 1
		}
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(count))
		return buf[:], ttl
	}

	if am, ok := backend.(cache.AtomicMutator); ok {
		newVal, err := am.Mutate(bgCtx, key, mutate)
		if err == nil {
			return int64(binary.BigEndian.Uint64(newVal)), nil
		}
	}

	var count int64 = 1
	if raw, ok := backend.Get(bgCtx, key); ok && len(raw) == 8 {
		count = int64(binary.BigEndian.Uint64(raw)) + 1
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(count))
	if err := backend.Set(bgCtx, key, buf[:], ttl); err != nil {
		return 0, err
	}
	return count, nil
}

// remoteAddrThrottlerKeyFunc keys purely off the transport peer address.
// X-Real-IP and X-Forwarded-For are attacker-controlled unless a trusted proxy
// overwrites them, so honouring them by default would let any client pick a
// fresh bucket per request and bypass the limit entirely. Opt in with
// Throttler.TrustProxyHeaders when a trusted proxy really does set them.
func remoteAddrThrottlerKeyFunc(c *ctx.HTTPContext) string {
	host, _, err := net.SplitHostPort(c.RemoteAddr)
	if err != nil || host == "" {
		return c.RemoteAddr
	}
	return host
}

func proxyAwareThrottlerKeyFunc(c *ctx.HTTPContext) string {
	if xrip := c.Request.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	return remoteAddrThrottlerKeyFunc(c)
}
