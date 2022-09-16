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
	}
}

func BenchmarkTrieInsert(b *testing.B) {
	j := 0
	for i := 0; i < b.N; i++ {
		j++
		if j == l-1 {
			j = 0
		}
		tr.insert(arr[j], '/', i, nil, nil)
	}
}

func BenchmarkTrieFind(b *testing.B) {
	j := 0
	for i := 0; i < b.N; i++ {
		j++
		if j == l-1 {
			j = 0
		}
		tr.find(arr[j], '/')
	}
}
