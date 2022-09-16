package ctx

// func TestNewParam(t *testing.T) {
// 	routeWithParamSymbol, p := NewParam("/foo/{param1}/{param2}/baz/{param3}/")
// 	keys := p.Keys()

// 	p.For(func(value interface{}, key string) {
// 		paramOrder := value.(int)
// 		keys[paramOrder] = key
// 	})

// 	output1 := len(keys)
// 	if len(keys) != 3 {
// 		t.Errorf("len(keys) = %v; expect = 3", output1)
// 	}

// 	output2 := keys[0]
// 	expect2 := "param1"
// 	if output2 != expect2 {
// 		t.Errorf("keys[0] = %v; expect = %v", output2, expect2)
// 	}

// 	output3 := keys[1]
// 	expect3 := "param2"
// 	if output3 != expect3 {
// 		t.Errorf("keys[1] = %v; expect = %v", output3, expect3)
// 	}

// 	output4 := keys[2]
// 	expect4 := "param3"
// 	if output4 != expect4 {
// 		t.Errorf("keys[2] = %v; expect = %v", output4, expect4)
// 	}

// 	expect5 := fmt.Sprintf("/foo/%v/%v/baz/%v/", "$", "$", "$")
// 	if routeWithParamSymbol != expect5 {
// 		t.Errorf("routeWithParamSymbol = %v; expect = %v", routeWithParamSymbol, expect5)
// 	}
// }
