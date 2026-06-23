package routing

import (
	"fmt"
	"net/http"
	"testing"
)

func buildBenchTrie() *Trie {
	routes := []string{
		fmt.Sprintf("/users/$/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/users/$/friends/$/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/feeds/all/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/products/$/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/products/$/reviews/$/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/*/feeds/{feed*Id}/*/files/*.html/*/%v/", fromMethodtoPattern(http.MethodGet)),
	}
	tr := NewTrie()
	for i, r := range routes {
		tr.insert(r, r, '/', i)
	}
	return tr
}

func BenchmarkTrieFind_Static(b *testing.B) {
	tr := buildBenchTrie()
	path := fmt.Sprintf("/feeds/all/[%v]/", http.MethodGet)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.find(path, http.MethodGet, "", '/')
	}
}

func BenchmarkTrieFind_WithParam(b *testing.B) {
	tr := buildBenchTrie()
	path := fmt.Sprintf("/users/633b0aa5d7fc3578b655b9bd/[%v]/", http.MethodGet)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.find(path, http.MethodGet, "", '/')
	}
}

func BenchmarkTrieFind_DeepParam(b *testing.B) {
	tr := buildBenchTrie()
	path := fmt.Sprintf("/users/633b0aa5d7fc3578b655b9bd/friends/633b0af45f4fe7d45b00fba5/[%v]/", http.MethodGet)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.find(path, http.MethodGet, "", '/')
	}
}

func BenchmarkTrieFind_NoMatch(b *testing.B) {
	tr := buildBenchTrie()
	path := fmt.Sprintf("/notexist/route/[%v]/", http.MethodGet)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.find(path, http.MethodGet, "", '/')
	}
}
