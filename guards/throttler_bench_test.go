package guards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/modules/cache"
)

func benchGuard(strategy Strategy) ThrottlerGuard {
	return ThrottlerGuard{
		Limit:    1000,
		TTL:      time.Minute,
		Strategy: strategy,
		KeyFunc:  func(*ctx.Context) string { return "benchkey" },
		Store:    cache.NewMemoryCache(),
	}
}

func benchCtx() *ctx.Context {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	c := ctx.NewContext()
	c.Request = req
	c.ResponseWriter = httptest.NewRecorder()
	return c
}

func BenchmarkFixedWindow(b *testing.B) {
	g := benchGuard(FixedWindow)
	bg := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.fixedWindow(bg, "ip")
	}
}

func BenchmarkSlidingWindow(b *testing.B) {
	g := benchGuard(SlidingWindow)
	bg := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.slidingWindow(bg, "ip")
	}
}

func BenchmarkTokenBucket(b *testing.B) {
	g := benchGuard(TokenBucket)
	bg := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.tokenBucket(bg, "ip")
	}
}

func BenchmarkDefaultKeyFunc_RemoteAddr(b *testing.B) {
	c := benchCtx()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		defaultThrottlerKeyFunc(c)
	}
}

func BenchmarkDefaultKeyFunc_XForwardedFor(b *testing.B) {
	c := benchCtx()
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		defaultThrottlerKeyFunc(c)
	}
}

func BenchmarkGuard_Allow(b *testing.B) {
	g := ThrottlerGuard{
		Limit:    int64(b.N) + 1,
		TTL:      time.Minute,
		Strategy: FixedWindow,
		KeyFunc:  func(*ctx.Context) string { return "benchkey" },
		Store:    cache.NewMemoryCache(),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := benchCtx()
		g.CanActivate(c)
	}
}

func BenchmarkFixedWindow_Parallel(b *testing.B) {
	g := benchGuard(FixedWindow)
	bg := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.fixedWindow(bg, "ip")
		}
	})
}

func BenchmarkTokenBucket_Parallel(b *testing.B) {
	g := benchGuard(TokenBucket)
	bg := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.tokenBucket(bg, "ip")
		}
	})
}
