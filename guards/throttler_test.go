package guards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/modules/cache"
	"github.com/dangduoc08/ginject/testutils"
)

type testCache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

func (tc *testCache) Get(_ context.Context, key string) ([]byte, bool) {
	tc.mu.RLock()
	v, ok := tc.items[key]
	tc.mu.RUnlock()
	return v, ok
}

func (tc *testCache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	tc.mu.Lock()
	tc.items[key] = append([]byte(nil), val...)
	tc.mu.Unlock()
	return nil
}

func (tc *testCache) Delete(_ context.Context, key string) error {
	tc.mu.Lock()
	delete(tc.items, key)
	tc.mu.Unlock()
	return nil
}

func newGuard(limit int64, ttl time.Duration, strategy Strategy) ThrottlerGuard {
	return ThrottlerGuard{
		Limit:    limit,
		TTL:      ttl,
		Strategy: strategy,
		KeyFunc:  func(*ctx.Context) string { return "testkey" },
		Store:    &testCache{items: make(map[string][]byte)},
	}
}

func newCtx(remoteAddr string) *ctx.Context {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	c := ctx.NewContext()
	c.Request = req
	c.ResponseWriter = httptest.NewRecorder()
	return c
}

// --- NewThrottler defaults ---

func TestNewThrottler_Defaults(t *testing.T) {
	g := NewThrottler(ThrottlerOptions{})
	if g.Limit != 100 {
		t.Error(testutils.DiffMessage(g.Limit, int64(100), "default Limit must be 100"))
	}
	if g.TTL != time.Minute {
		t.Error(testutils.DiffMessage(g.TTL, time.Minute, "default TTL must be 1 minute"))
	}
	if g.Store == nil {
		t.Error(testutils.DiffMessage(g.Store, "non-nil", "default Store must be set"))
	}
	if g.KeyFunc == nil {
		t.Error(testutils.DiffMessage(g.KeyFunc, "non-nil", "default KeyFunc must be set"))
	}
}

func TestNewThrottler_CustomStore(t *testing.T) {
	tc := &testCache{items: make(map[string][]byte)}
	g := NewThrottler(ThrottlerOptions{Store: tc})
	if g.Store != tc {
		t.Error(testutils.DiffMessage(g.Store, tc, "custom Store must be used"))
	}
}

// --- Fixed Window ---

func TestFixedWindow_AllowsUpToLimit(t *testing.T) {
	g := newGuard(3, time.Minute, FixedWindow)
	for i := 0; i < 3; i++ {
		res := g.fixedWindow(context.Background(), "ip")
		if !res.allowed {
			t.Error(testutils.DiffMessage(res.allowed, true, "all requests within limit must be allowed"))
		}
	}
}

func TestFixedWindow_BlocksOverLimit(t *testing.T) {
	g := newGuard(3, time.Minute, FixedWindow)
	for i := 0; i < 3; i++ {
		g.fixedWindow(context.Background(), "ip")
	}
	res := g.fixedWindow(context.Background(), "ip")
	if res.allowed {
		t.Error(testutils.DiffMessage(res.allowed, false, "4th request must be blocked"))
	}
}

func TestFixedWindow_RemainingDecreases(t *testing.T) {
	g := newGuard(5, time.Minute, FixedWindow)
	res := g.fixedWindow(context.Background(), "ip")
	if res.remaining != 4 {
		t.Error(testutils.DiffMessage(res.remaining, int64(4), "remaining after first request must be 4"))
	}
	res = g.fixedWindow(context.Background(), "ip")
	if res.remaining != 3 {
		t.Error(testutils.DiffMessage(res.remaining, int64(3), "remaining after second must be 3"))
	}
}

func TestFixedWindow_RemainingNeverNegative(t *testing.T) {
	g := newGuard(1, time.Minute, FixedWindow)
	g.fixedWindow(context.Background(), "ip")
	res := g.fixedWindow(context.Background(), "ip")
	if res.remaining < 0 {
		t.Error(testutils.DiffMessage(res.remaining, int64(0), "remaining must never be negative"))
	}
}

func TestFixedWindow_ResetAtIsInFuture(t *testing.T) {
	g := newGuard(5, time.Minute, FixedWindow)
	res := g.fixedWindow(context.Background(), "ip")
	if res.resetAt <= time.Now().Unix() {
		t.Error(testutils.DiffMessage(res.resetAt, ">now", "resetAt must be in the future"))
	}
}

func TestFixedWindow_DifferentKeysAreIndependent(t *testing.T) {
	g := newGuard(1, time.Minute, FixedWindow)
	g.fixedWindow(context.Background(), "ip1")
	res := g.fixedWindow(context.Background(), "ip2")
	if !res.allowed {
		t.Error(testutils.DiffMessage(res.allowed, true, "different keys must be independent"))
	}
}

// --- Sliding Window ---

func TestSlidingWindow_AllowsUpToLimit(t *testing.T) {
	g := newGuard(3, time.Minute, SlidingWindow)
	for i := 0; i < 3; i++ {
		res := g.slidingWindow(context.Background(), "ip")
		if !res.allowed {
			t.Error(testutils.DiffMessage(res.allowed, true, "all requests within limit must be allowed"))
		}
	}
}

func TestSlidingWindow_BlocksOverLimit(t *testing.T) {
	g := newGuard(3, time.Minute, SlidingWindow)
	for i := 0; i < 3; i++ {
		g.slidingWindow(context.Background(), "ip")
	}
	res := g.slidingWindow(context.Background(), "ip")
	if res.allowed {
		t.Error(testutils.DiffMessage(res.allowed, false, "4th request must be blocked"))
	}
}

func TestSlidingWindow_ResetAtIsInFuture(t *testing.T) {
	g := newGuard(5, time.Minute, SlidingWindow)
	res := g.slidingWindow(context.Background(), "ip")
	if res.resetAt <= time.Now().Unix() {
		t.Error(testutils.DiffMessage(res.resetAt, ">now", "resetAt must be in the future"))
	}
}

// --- Token Bucket ---

func TestTokenBucket_AllowsUpToLimit(t *testing.T) {
	g := newGuard(5, time.Minute, TokenBucket)
	for i := 0; i < 5; i++ {
		res := g.tokenBucket(context.Background(), "ip")
		if !res.allowed {
			t.Error(testutils.DiffMessage(res.allowed, true, "all requests within limit must be allowed"))
		}
	}
}

func TestTokenBucket_BlocksWhenExhausted(t *testing.T) {
	g := newGuard(3, time.Minute, TokenBucket)
	for i := 0; i < 3; i++ {
		g.tokenBucket(context.Background(), "ip")
	}
	res := g.tokenBucket(context.Background(), "ip")
	if res.allowed {
		t.Error(testutils.DiffMessage(res.allowed, false, "bucket must be exhausted after limit requests"))
	}
}

func TestTokenBucket_RemainingNeverNegative(t *testing.T) {
	g := newGuard(1, time.Minute, TokenBucket)
	g.tokenBucket(context.Background(), "ip")
	res := g.tokenBucket(context.Background(), "ip")
	if res.remaining < 0 {
		t.Error(testutils.DiffMessage(res.remaining, int64(0), "remaining must never be negative"))
	}
}

func TestTokenBucket_ResetAtIsSet(t *testing.T) {
	g := newGuard(1, time.Minute, TokenBucket)
	g.tokenBucket(context.Background(), "ip")
	res := g.tokenBucket(context.Background(), "ip")
	if res.resetAt == 0 {
		t.Error(testutils.DiffMessage(res.resetAt, ">0", "resetAt must be non-zero when exhausted"))
	}
}

// --- defaultThrottlerKeyFunc ---

func TestDefaultKeyFunc_RemoteAddr(t *testing.T) {
	c := newCtx("192.168.1.1:1234")
	if key := defaultThrottlerKeyFunc(c); key != "192.168.1.1" {
		t.Error(testutils.DiffMessage(key, "192.168.1.1", "must extract IP from RemoteAddr"))
	}
}

func TestDefaultKeyFunc_XRealIP(t *testing.T) {
	c := newCtx("10.0.0.1:0")
	c.Request.Header.Set("X-Real-IP", "203.0.113.1")
	if key := defaultThrottlerKeyFunc(c); key != "203.0.113.1" {
		t.Error(testutils.DiffMessage(key, "203.0.113.1", "X-Real-IP must take priority"))
	}
}

func TestDefaultKeyFunc_XForwardedFor(t *testing.T) {
	c := newCtx("10.0.0.1:0")
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.2, 10.0.0.1")
	if key := defaultThrottlerKeyFunc(c); key != "203.0.113.2" {
		t.Error(testutils.DiffMessage(key, "203.0.113.2", "must use first IP from X-Forwarded-For"))
	}
}

func TestDefaultKeyFunc_XRealIPPriority(t *testing.T) {
	c := newCtx("10.0.0.1:0")
	c.Request.Header.Set("X-Real-IP", "203.0.113.1")
	c.Request.Header.Set("X-Forwarded-For", "198.51.100.1")
	if key := defaultThrottlerKeyFunc(c); key != "203.0.113.1" {
		t.Error(testutils.DiffMessage(key, "203.0.113.1", "X-Real-IP must beat X-Forwarded-For"))
	}
}

func TestDefaultKeyFunc_InvalidRemoteAddr(t *testing.T) {
	c := newCtx("not-an-addr")
	if key := defaultThrottlerKeyFunc(c); key != "not-an-addr" {
		t.Error(testutils.DiffMessage(key, "not-an-addr", "unparseable RemoteAddr must be returned as-is"))
	}
}

// --- ThrottlerGuard ---

func TestThrottlerGuard_SetsHeadersOnAllow(t *testing.T) {
	g := newGuard(10, time.Minute, FixedWindow)
	c := newCtx("127.0.0.1:0")

	if !g.CanActivate(c) {
		t.Error(testutils.DiffMessage(false, true, "first request must be allowed"))
	}
	rec := c.ResponseWriter.(*httptest.ResponseRecorder)
	for _, h := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		if rec.Header().Get(h) == "" {
			t.Error(testutils.DiffMessage("", "non-empty", h+" must be set"))
		}
	}
}

func TestThrottlerGuard_PanicsOnExceed(t *testing.T) {
	g := newGuard(1, time.Minute, FixedWindow)
	g.CanActivate(newCtx("127.0.0.1:0"))

	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "exceeded guard must panic"))
		}
	}()
	g.CanActivate(newCtx("127.0.0.1:0"))
}

func TestThrottlerGuard_SetsRetryAfterOnExceed(t *testing.T) {
	g := newGuard(1, time.Minute, FixedWindow)
	g.CanActivate(newCtx("127.0.0.1:0"))

	c2 := newCtx("127.0.0.1:0")
	func() {
		defer func() { recover() }()
		g.CanActivate(c2)
	}()

	if c2.ResponseWriter.(*httptest.ResponseRecorder).Header().Get("Retry-After") == "" {
		t.Error(testutils.DiffMessage("", "non-empty", "Retry-After must be set when rate limited"))
	}
}

// --- Concurrent safety ---

func TestFixedWindow_ConcurrentSafe(t *testing.T) {
	g := newGuard(1000, time.Minute, FixedWindow)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.fixedWindow(context.Background(), "ip")
		}()
	}
	wg.Wait()
}

func TestTokenBucket_ConcurrentSafe(t *testing.T) {
	g := newGuard(1000, time.Minute, TokenBucket)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.tokenBucket(context.Background(), "ip")
		}()
	}
	wg.Wait()
}

func TestSlidingWindow_ConcurrentSafe(t *testing.T) {
	g := newGuard(1000, time.Minute, SlidingWindow)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.slidingWindow(context.Background(), "ip")
		}()
	}
	wg.Wait()
}

// --- check dispatch ---

func TestCheck_DispatchesAllStrategies(t *testing.T) {
	for _, s := range []Strategy{FixedWindow, SlidingWindow, TokenBucket} {
		g := newGuard(5, time.Minute, s)
		c := newCtx("127.0.0.1:0")
		res := g.check(c)
		if res.limit != 5 {
			t.Error(testutils.DiffMessage(res.limit, int64(5), "limit must match for each strategy"))
		}
	}
}

// --- cache.Cache compatibility ---

func TestThrottlerGuard_WorksWithRealMemoryCache(t *testing.T) {
	g := ThrottlerGuard{
		Limit:    3,
		TTL:      time.Minute,
		Strategy: FixedWindow,
		KeyFunc:  func(*ctx.Context) string { return "ip" },
		Store:    cache.NewMemoryCache(),
	}
	for i := 0; i < 3; i++ {
		res := g.fixedWindow(context.Background(), "ip")
		if !res.allowed {
			t.Error(testutils.DiffMessage(res.allowed, true, "real cache must allow within limit"))
		}
	}
	res := g.fixedWindow(context.Background(), "ip")
	if res.allowed {
		t.Error(testutils.DiffMessage(res.allowed, false, "real cache must block over limit"))
	}
}
