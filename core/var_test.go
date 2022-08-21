package core

import (
	"fmt"
	"testing"
)

func TestNewVar(test *testing.T) {
	routeWithVar, newVarStruct := NewVar("/foo/{var1}/{var2}/baz/{var3}/")
	vars := []string{}
	for k := range newVarStruct.KeyValue {
		vars = append(vars, k)
	}

	output1 := len(vars)
	if len(vars) != 3 {
		test.Errorf("len(vars) = %v; expect = 3", output1)
	}

	output2 := vars[0]
	expect2 := "var1"
	if output2 != expect2 {
		test.Errorf("vars[0] = %v; expect = %v", output2, expect2)
	}

	output3 := vars[1]
	expect3 := "var2"
	if output3 != expect3 {
		test.Errorf("vars[1] = %v; expect = %v", output3, expect3)
	}

	output4 := vars[2]
	expect4 := "var3"
	if output4 != expect4 {
		test.Errorf("vars[2] = %v; expect = %v", output4, expect4)
	}

	expect5 := fmt.Sprintf("/foo/%v/%v/baz/%v/", VAR_SYMBOL, VAR_SYMBOL, VAR_SYMBOL)
	if routeWithVar != expect5 {
		test.Errorf("routeWithVararamSymbol = %v; expect = %v", routeWithVar, expect5)
	}
}
