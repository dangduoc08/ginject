package utils

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNumF64ToAnyNum(t *testing.T) {
	input1 := 1231231.1231

	output1 := NumF64ToAnyNum(input1, reflect.Int)

	fmt.Println(output1)
}
