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
	for i, r := range routes {
		tr.Insert(r, r, '/', i)
	}
	return tr
}

func BenchmarkTrieFind_Static(b *testing.B) {
	tr := buildBenchTrie()
	path := "/feeds/all/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/')
	}
}

func BenchmarkTrieFind_WithParam(b *testing.B) {
	tr := buildBenchTrie()
	path := "/users/633b0aa5d7fc3578b655b9bd/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/')
	}
}

func BenchmarkTrieFind_DeepParam(b *testing.B) {
	tr := buildBenchTrie()
	path := "/users/633b0aa5d7fc3578b655b9bd/friends/633b0af45f4fe7d45b00fba5/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/')
	}
}

func BenchmarkTrieFind_NoMatch(b *testing.B) {
	tr := buildBenchTrie()
	path := "/notexist/route/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/')
	}
}
