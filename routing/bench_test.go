package routing

import (
	"testing"

	"github.com/dangduoc08/gooh/utils"
)

var tr = NewTrie()
var l = 1000
var arr = make([]string, l)

func init() {
	for i := 0; i < l; i++ {
		randStr := utils.StrRandom(10) + "/" + utils.StrRandom(10) + "/" + utils.StrRandom(10) + "/" + utils.StrRandom(10) + "/" + utils.StrRandom(10) + "/" + utils.StrRandom(10)
		arr[i] = randStr
		tr.insert(randStr, '/', i, nil, nil)
	}
}

func BenchmarkTrieInsert(b *testing.B) {
	b.StopTimer()
	var trie = NewTrie()
	j := 0
	for i := 0; i < b.N; i++ {
		j++
		if j == l-1 {
			j = 0
		}
		b.StartTimer()
		trie.insert(arr[j], '/', i, nil, nil)
		b.StopTimer()
	}
}

func BenchmarkTrieFind(b *testing.B) {
	b.StopTimer()
	j := 0
	for i := 0; i < b.N; i++ {
		j++
		if j == l-1 {
			j = 0
		}
		b.StartTimer()
		tr.find(arr[j], '/')
		b.StopTimer()
	}
}
