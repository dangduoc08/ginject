package ctx

import (
	"fmt"
	"testing"
)

func TestNewParam(test *testing.T) {
	routeWithParamSymbol, newParamStruct := NewParam("/foo/{param1}/{param2}/baz/{param3}/")
	keys := newParamStruct.Keys()

	newParamStruct.ForEach(func(value interface{}, key string) {
		paramOrder := value.(int)
		keys[paramOrder] = key
	})

	output1 := len(keys)
	if len(keys) != 3 {
		test.Errorf("len(keys) = %v; expect = 3", output1)
	}

	output2 := keys[0]
	expect2 := "param1"
	if output2 != expect2 {
		test.Errorf("keys[0] = %v; expect = %v", output2, expect2)
	}

	output3 := keys[1]
	expect3 := "param2"
	if output3 != expect3 {
		test.Errorf("keys[1] = %v; expect = %v", output3, expect3)
	}

	output4 := keys[2]
	expect4 := "param3"
	if output4 != expect4 {
		test.Errorf("keys[2] = %v; expect = %v", output4, expect4)
	}

	expect5 := fmt.Sprintf("/foo/%v/%v/baz/%v/", PARAM_SYMBOL, PARAM_SYMBOL, PARAM_SYMBOL)
	if routeWithParamSymbol != expect5 {
		test.Errorf("routeWithParamSymbol = %v; expect = %v", routeWithParamSymbol, expect5)
	}
}
