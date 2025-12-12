package cache

import (
	"testing"

	"github.com/dangduoc08/ginject/utils"
)

func TestDLLModifyNode(t *testing.T) {
	dll := NewDLL[string]()

	headArr := []string{}
	tailArr := []string{}
	genCounter := 1000

	for i := 0; i < genCounter; i++ {
		randStr := utils.StrRandom(20)
		headArr = append(headArr, randStr)
		dll.unshift(randStr)
	}

	for i := 0; i < genCounter; i++ {
		randStr := utils.StrRandom(20)
		tailArr = append(tailArr, randStr)
		dll.push(randStr)
	}

	dll.pop()
	dll.shift()
	dll.delete(dll.tail.prev.prev)

	output1 := dll.head.data
	expect1 := headArr[len(headArr)-2]
	if output1 != expect1 {
		t.Errorf("dll.head.data = %v; expect = %v", output1, expect1)
	}

	output2 := dll.tail.data
	expect2 := tailArr[len(tailArr)-2]
	if output2 != expect2 {
		t.Errorf("dll.tail.data = %v; expect = %v", output2, expect2)
	}

	output3 := dll.size
	expect3 := genCounter*2 - 2
	if int(dll.size) != genCounter*2-3 {
		t.Errorf("dll.size = %v; expect = %v", output3, expect3)
	}
}
