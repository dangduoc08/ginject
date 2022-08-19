package core

import (
	"fmt"
	"testing"
)

func TestNewParam(test *testing.T) {
	routeWithP, newParamStruct := NewParam("/foo/:param1/:param2/baz/:param3/")
	params := []string{}
	for k := range newParamStruct.KeyValue {
		params = append(params, k)
	}

	output1 := len(params)
	if len(params) != 3 {
		test.Errorf("len(params) = %v; expect = 3", output1)
	}

	output2 := params[0]
	expect2 := "param1"
	if output2 != expect2 {
		test.Errorf("params[0] = %v; expect = %v", output2, expect2)
	}

	output3 := params[1]
	expect3 := "param2"
	if output3 != expect3 {
		test.Errorf("params[0] = %v; expect = %v", output3, expect3)
	}

	output4 := params[2]
	expect4 := "param3"
	if output4 != expect4 {
		test.Errorf("params[0] = %v; expect = %v", output4, expect4)
	}

	expect5 := fmt.Sprintf("/foo/%v/%v/baz/%v/", PARAM_SYMBOL, PARAM_SYMBOL, PARAM_SYMBOL)
	if routeWithP != expect5 {
		test.Errorf("routeWithParamSymbol = %v; expect = %v", routeWithP, expect5)
	}
}
