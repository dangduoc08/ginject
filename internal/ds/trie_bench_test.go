package ds

import (
	"strconv"
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
	tr := buildBenchTrie()
	path := "/feeds/all/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/', false)
	}
}

func BenchmarkTrieFind_WithParam(b *testing.B) {
	tr := buildBenchTrie()
	path := "/users/633b0aa5d7fc3578b655b9bd/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/', true)
	}
}

func BenchmarkTrieFind_DeepParam(b *testing.B) {
	tr := buildBenchTrie()
	path := "/users/633b0aa5d7fc3578b655b9bd/friends/633b0af45f4fe7d45b00fba5/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/', true)
	}
}

func BenchmarkTrieFind_NoMatch(b *testing.B) {
	tr := buildBenchTrie()
	path := "/notexist/route/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Find(path, '/', false)
	}
}

func BenchmarkTrieRemove(b *testing.B) {
	tr := buildBenchTrie()
	paths := make([]string, b.N)
	for i := range paths {
		p := "/bench/remove/" + strconv.Itoa(i) + "/"
		paths[i] = p
		tr.Insert(p, p, '/')
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Remove(paths[i], '/')
	}
}
