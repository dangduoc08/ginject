package routing

import (
	"strings"
	"testing"

	"github.com/dangduoc08/gooh/context"
)

func TestAdd(test *testing.T) {
	const (
		routeParams1 = "foo/{param1}/{param2}/baz/{param3}"
		routeParams2 = "/foo/{param1}/{param2}/baz/param3/"
	)

	expectTotalNode := len("/foo/$/$/baz/$/") + len("param3/")

	routerInstance := NewRouter()
	routerInstance.add(routeParams1, nil)
	routerInstance.add(routeParams2, nil)

	actualTotalNode := routerInstance.Trie.Len()
	output1 := actualTotalNode == uint(expectTotalNode)
	if !output1 {
		test.Errorf("ro.len() = %v; expect = %v", actualTotalNode, expectTotalNode)
	}
}

func TestMatch(test *testing.T) {
	routerInstance := NewRouter()
	for _, route := range []string{
		"/v1/users/get/jobs/get",
		"/v2/users/get/*/jobs/{jobId}get",
		"/v2/users/{userId}/*/jobs/{jobId}/get",
		"/v1/users/{userId}/*/jobs/{jobId}/delete",
		"/v2/users/{userId}/*/jobs/{jobId}/*",
	} {
		routerInstance.add(route, nil)
	}

	isFound1, _, routerData1 := routerInstance.Match("/v1/users/get/jobs/get")
	var expect1Rd1Var interface{}
	if !isFound1 {
		test.Errorf("routerInstance.Match(\"/v1/users/get/jobs/get\") = %v; expect = %v", isFound1, true)
	}
	if routerData1.Params.Get("any") != expect1Rd1Var {
		test.Errorf("routerData1.Params.Get(\"any\") = %v; expect = %v", routerData1.Params.Get("any"), expect1Rd1Var)
	}

	isFound2, _, routerData2 := routerInstance.Match("/v2/users/63029905408d8ed70d411662/update/jobs/63029924b2584a856cbb8baf/get")
	expect2UserId := "63029905408d8ed70d411662"
	output2UserId := routerData2.Params.Get("userId")
	expect2JobId := "63029924b2584a856cbb8baf"
	output2JobId := routerData2.Params.Get("jobId")
	if !isFound2 {
		test.Errorf("routerInstance.Match(\"/v2/users/63029905408d8ed70d411662/update/jobs/63029924b2584a856cbb8baf/get\") = %v; expect = %v", isFound2, true)
	}
	if output2UserId != expect2UserId {
		test.Errorf("routerData2.Params.Get(\"userId\") = %v; expect = %v", output2UserId, expect2UserId)
	}
	if output2JobId != expect2JobId {
		test.Errorf("routerData2.Params.Get(\"jobId\") = %v; expect = %v", output2JobId, expect2JobId)
	}

	isFound3, _, routerData3 := routerInstance.Match("/v1/users/63029c3246fd350a3ffc276c/update/jobs/63029c3bb998dabb261d99a1/delete")
	expect3UserId := "63029c3246fd350a3ffc276c"
	output3UserId := routerData3.Params.Get("userId")
	expect3JobId := "63029c3bb998dabb261d99a1"
	output3JobId := routerData3.Params.Get("jobId")

	if !isFound3 {
		test.Errorf("routerInstance.Match(\"/v1/users/63029c3246fd350a3ffc276c/update/jobs/63029c3bb998dabb261d99a1/delete\") = %v; expect = %v", isFound3, true)
	}
	if expect3UserId != output3UserId {
		test.Errorf("routerData3.Params.Get(\"userId\") = %v; expect = %v", output3UserId, expect3UserId)
	}
	if expect3JobId != output3JobId {
		test.Errorf("routerData3.Params.Get(\"jobId\") = %v; expect = %v", output3JobId, expect3JobId)
	}

	isFound4, _, routerData4 := routerInstance.Match("/v2/users/63029e1271f0bfaab1697c01/delete/jobs/63029e20f75076f6e6b8fdee/move")
	expect4UserId := "63029e1271f0bfaab1697c01"
	output4UserId := routerData4.Params.Get("userId")
	expect4JobId := "63029e20f75076f6e6b8fdee"
	output4JobId := routerData4.Params.Get("jobId")
	if !isFound4 {
		test.Errorf("routerInstance.Match(\"/v2/users/63029e1271f0bfaab1697c01/update/jobs/63029e20f75076f6e6b8fdee/move\") = %v; expect = %v", isFound4, true)
	}
	if expect4UserId != output4UserId {
		test.Errorf("routerData4.Params.Get(\"userId\") = %v; expect = %v", output4UserId, expect4UserId)
	}
	if expect4JobId != output4JobId {
		test.Errorf("routerData4.Params.Get(\"jobId\") = %v; expect = %v", output4JobId, expect4JobId)
	}

	// nagative test
	isFound5, _, _ := routerInstance.Match("/v2/users/gett/delete/jobs/6302a6d946dc9b4a37c1d281/get")
	if isFound5 {
		test.Errorf("routerInstance.Match(\"/v2/users/gett/delete/jobs/6302a6d946dc9b4a37c1d281/get\") = %v; expect = %v", isFound5, false)
	}
}

func TestGroup(test *testing.T) {
	routerInstance1 := NewRouter()
	for _, route1 := range []string{
		"/users/get",
		"/users/get/{userId}",
	} {
		routerInstance1.add(route1, nil)
	}

	routerInstance2 := NewRouter()
	for _, route2 := range []string{
		"/users/update/{userId}",
		"/users/delete/{userId}",
	} {
		routerInstance2.add(route2, nil)
	}

	routerGr := NewRouter()
	routerGr.Group("/v1", routerInstance1, routerInstance2)

	_, matchedRoute, _ := routerGr.Match("/v1/users/update/123")
	expectMatchedRoute := "/v1/users/update/{userId}/"
	if matchedRoute != expectMatchedRoute {
		test.Errorf("routerGr.Match(\"/v1/users/update/123\") = %v; expect = %v", matchedRoute, expectMatchedRoute)
	}
}

var holdValueFromMiddleware = make([]string, 0)

func middleware1(ctx *context.Context) {
	holdValueFromMiddleware = append(holdValueFromMiddleware, "middleware1")
}
func middleware2(ctx *context.Context) {
	holdValueFromMiddleware = append(holdValueFromMiddleware, "middleware2")
}
func handler1(ctx *context.Context) {
	holdValueFromMiddleware = append(holdValueFromMiddleware, "handler1")
}
func handler2(ctx *context.Context) {
	holdValueFromMiddleware = append(holdValueFromMiddleware, "handler2")
}
func middleware3(ctx *context.Context) {
	holdValueFromMiddleware = append(holdValueFromMiddleware, "middleware3")
}
func middleware4(ctx *context.Context) {
	holdValueFromMiddleware = append(holdValueFromMiddleware, "middleware4")
}

func TestMiddleware(test *testing.T) {
	routerInstance1 := NewRouter()
	routerInstance1.Use(middleware1)
	routerInstance1.For("/users/{userId}")(middleware2, middleware3)
	routerInstance1.add("/[POST]/users/{userId}", handler1)
	routerInstance1.For("/users/{userId}")(middleware3, middleware1)
	routerInstance1.Use(middleware4, middleware2)

	routerInstance1.Use(middleware1)
	routerInstance1.For("/products")(middleware2, middleware3)
	routerInstance1.add("/[TRACE]/products", handler1)
	routerInstance1.For("/products")(middleware3, middleware1)
	routerInstance1.Use(middleware4, middleware2)

	routerGr := NewRouter()
	routerGr.Group("/v1", routerInstance1)
	routerGr.Use(middleware2)

	_, matchedRoute, routerData := routerGr.Match("/[POST]/v1/users/631253712bf56df421c80977")

	expectMatchedRoute := "/[POST]/v1/users/{userId}/"
	expectUserId := "631253712bf56df421c80977"

	if matchedRoute != expectMatchedRoute {
		test.Errorf("matchedRoute = %v; expect = %v", matchedRoute, expectMatchedRoute)
	}

	if routerData.Params.Get("userId") != expectUserId {
		test.Errorf("routerData.Params.Get(\"userId\") = %v; expect = %v", routerData.Params.Get("userId"), expectUserId)
	}

	for _, handlers := range *routerData.Handlers {
		handlers(&context.Context{})
	}

	expectMiddlewareExecutedOrder := "middleware1, middleware2, middleware3, handler1, middleware3, middleware1, middleware4, middleware2, middleware1, middleware4, middleware2, middleware2"
	actualMiddlewareExecutedOrder := strings.Join(holdValueFromMiddleware[:], ", ")

	if expectMiddlewareExecutedOrder != actualMiddlewareExecutedOrder {
		test.Errorf("actualMiddlewareExecutedOrder = %v; expect = %v", actualMiddlewareExecutedOrder, expectMiddlewareExecutedOrder)
	}
}

func TestRoutable(test *testing.T) {
	userRouter := NewRouter()
	userRouter.Use(middleware1)
	userRouter.For("/users")(middleware2, middleware3)
	userRouter.Get("/users/get", handler1)
	userRouter.Get("/users/{userId}", handler1)
	userRouter.Post("/users", handler1)
	userRouter.Put("/users/{userId}", handler1)
	userRouter.Delete("/users/{userId}", handler1)
	userRouter.For("/users/{userId}")(middleware2, middleware3)

	productRouter := NewRouter()
	productRouter.Get("/products", handler1)
	productRouter.Get("/products/{productId}", handler1)
	productRouter.Post("/products", handler1)
	productRouter.Put("/products/{productId}", handler1)
	productRouter.Delete("/products/{productId}", handler1)

	v1 := NewRouter()
	v1.Group("/v1/", userRouter, productRouter)

	v2 := NewRouter()
	v2.Group("/v2", userRouter, productRouter)

	all := NewRouter()
	all.Group("/all", v1, v2)

}
