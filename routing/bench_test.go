package routing

import (
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/internal/crypto"
)

var tr = NewTrie()
var l = 1000
var arr = make([]string, l)

func init() {
	for i := 0; i < l; i++ {
		randStr := crypto.Random(10) +
			"/" +
			crypto.Random(10) +
			"/" +
			crypto.Random(10) +
			"/" +
			crypto.Random(10) +
			"/" +
			crypto.Random(10) +
			"/" +
			crypto.Random(10) +
			"/" +
			crypto.Random(10) +
			"/" +
			crypto.Random(10)

		arr[i] = randStr
		tr.insert(randStr, randStr, '/', i)
	}
}

func BenchmarkTrieInsert(b *testing.B) {
	var trie = NewTrie()
	j := 0
	for i := 0; i < b.N; i++ {
		j++
		if j == l-1 {
			j = 0
		}
		trie.insert(arr[j], arr[j], '/', i)
	}
}

func BenchmarkTrieFind(b *testing.B) {
	j := 0
	for i := 0; i < b.N; i++ {
		j++
		if j == l-1 {
			j = 0
		}
		tr.find("", arr[j], "", '/')
	}
}

func BenchmarkRouterMatch(b *testing.B) {
	r := NewRouter()
	r.Add(http.MethodGet, "/users/{userId}/all", "", nil)

	for i := 0; i < b.N; i++ {
		r.Match(http.MethodGet, "/users/123/all/", "")
	}
}
