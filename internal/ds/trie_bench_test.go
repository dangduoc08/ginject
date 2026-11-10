package ds

import (
	"testing"
)

func buildBenchTrie() *Trie {
	routes := []string{
		"/users/$/",
		"/users/$/friends/$/",
		"/feeds/all/",
		"/products/$/",
		"/products/$/reviews/$/",
		"/*/feeds/{feed*Id}/*/files/*.html/*/",
	}
	tr := NewTrie()
	for _, r := range routes {
		tr.Insert(r, r, '/')
	}
	return tr
}

func BenchmarkTrieFind_Static(b *testing.B) {
	b.ReportAllocs()
	tr := buildBenchTrie()
	path := "/feeds/all/"
	for b.Loop() {
		tr.Find(path, '/')
	}
}

func BenchmarkTrieFind_WithParam(b *testing.B) {
	b.ReportAllocs()
	tr := buildBenchTrie()
	path := "/users/633b0aa5d7fc3578b655b9bd/"
	for b.Loop() {
		tr.Find(path, '/')
	}
}

func BenchmarkTrieFind_DeepParam(b *testing.B) {
	b.ReportAllocs()
	tr := buildBenchTrie()
	path := "/users/633b0aa5d7fc3578b655b9bd/friends/633b0af45f4fe7d45b00fba5/"
	for b.Loop() {
		tr.Find(path, '/')
	}
}

func BenchmarkTrieFind_NoMatch(b *testing.B) {
	b.ReportAllocs()
	tr := buildBenchTrie()
	path := "/notexist/route/"
	for b.Loop() {
		tr.Find(path, '/')
	}
}

func BenchmarkTrieFind_Realistic(b *testing.B) {
	tr := NewTrie()
	for _, route := range []string{"/api/v1/users/$/", "/api/v1/users/$/orders/", "/api/v1/users/me/", "/api/v1/products/$/reviews/$/", "/static/*/", "/health/"} {
		tr.Insert(route, route, '/')
	}
	for _, path := range []string{"/health/", "/api/v1/users/123/orders/", "/api/v1/products/9/reviews/3/", "/api/v1/users/me/orders/", "/api/v2/nothing/"} {
		b.Run(path, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tr.Find(path, '/')
			}
		})
	}
}

func BenchmarkMatchWildcard(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		matchWildcard("index.html", "in*.html")
	}
}
