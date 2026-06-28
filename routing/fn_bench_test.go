package routing

import (
	"net/http"
	"testing"
)

func BenchmarkPatternToMethodRouteVersion(b *testing.B) {
	pattern := MethodRouteVersionToPattern(http.MethodGet, "/users/{userId}/friends/{friendId}", "v2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PatternToMethodRouteVersion(pattern)
	}
}

func BenchmarkParseToParamKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseToParamKey("/users/$/friends/$/schools/$/subjects/$/")
	}
}

func BenchmarkMethodRouteVersionToPattern(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MethodRouteVersionToPattern(http.MethodGet, "/users/{userId}/all", "v2")
	}
}
