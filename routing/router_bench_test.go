package routing

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

func randomSegment(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func buildBenchRouter(n int) (*Router, []string) {
	r := NewRouter()
	routes := make([]string, n)
	noop := ctx.Handler(func(c *ctx.Context) {})
	for i := 0; i < n; i++ {
		route := fmt.Sprintf("/%s/%s/{id}/%s", randomSegment(8), randomSegment(8), randomSegment(8))
		routes[i] = route
		r.Add(http.MethodGet, route, "", noop)
	}
	return r, routes
}

func BenchmarkRouterAdd(b *testing.B) {
	r := NewRouter()
	routes := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		routes[i] = fmt.Sprintf("/%s/%s/{id}/%s", randomSegment(8), randomSegment(8), randomSegment(8))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Add(http.MethodGet, routes[i], "", nil)
	}
}

func BenchmarkRouterMatch_Static(b *testing.B) {
	r, _ := buildBenchRouter(1000)
	r.Add(http.MethodGet, "/users/all", "", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(http.MethodGet, "/users/all/", "")
	}
}

func BenchmarkRouterMatch_Param(b *testing.B) {
	r, routes := buildBenchRouter(1000)
	requestPath := strings.Replace(routes[len(routes)/2], "{id}", "123", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(http.MethodGet, requestPath, "")
	}
}

func BenchmarkRouterMatch_NoMatch(b *testing.B) {
	r, _ := buildBenchRouter(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Match(http.MethodGet, "/notexist/route/", "")
	}
}

func BenchmarkRouterUse(b *testing.B) {
	noop := ctx.Handler(func(c *ctx.Context) {})

	b.StopTimer()
	for i := 0; i < b.N; i++ {
		r, _ := buildBenchRouter(100)
		b.StartTimer()
		r.Use(noop)
		b.StopTimer()
	}
}
