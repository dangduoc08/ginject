package routing

import (
	"fmt"
	"testing"
)

func TestA(test *testing.T) {
	rData := routerData{}
	r := new(Router)
	r.trie = newTrie[routerData]()

	r.insert("/", rData)

	fmt.Println(*r.trie)
}
