package routing

import (
	"fmt"
	"os"
	"testing"

	"github.com/dangduoc08/gooh/core"
)

func TestAdd(test *testing.T) {
	const (
		routeParams1 = "foo/{param1}/{param2}/baz/{param3}"
		routeParams2 = "/foo/{param1}/{param2}/baz/param3/"
	)

	expectTotalNode := len("/foo/$/$/baz/$/") + len("param3/")

	ro := NewRouter()
	ro.Add(routeParams1, nil)
	ro.Add(routeParams2, func(req *core.Request, res core.ResponseExtender, next func()) {

	})

	actualTotalNode := ro.Trie.Len()
	output1 := actualTotalNode == uint(expectTotalNode)
	if !output1 {
		test.Errorf("ro.len() = %v; expect = %v", actualTotalNode, expectTotalNode)
	}
}

func TestMatch(test *testing.T) {
	ro := NewRouter()
	for _, r := range []string{
		"/v1/users/get/jobs/get",
		"/v2/users/get/*/jobs/{jobId}get",
		"/v2/users/{userId}/*/jobs/{jobId}/get",
		"/v1/users/{userId}/*/jobs/{jobId}/delete",
		"/v2/users/{userId}/*/jobs/{jobId}/*",
	} {
		ro.Add(r, func(req *core.Request, res core.ResponseExtender, next func()) {})
	}

	isFound1, _, rD1 := ro.Match("/v1/users/get/jobs/get")
	var expect1Rd1Var interface{}
	if !isFound1 {
		test.Errorf("ro.Match(\"/v1/users/get/jobs/get\") = %v; expect = %v", isFound1, true)
	}
	if rD1.Vars.Get("any") != expect1Rd1Var {
		test.Errorf("rD1.Vars.Get(\"any\") = %v; expect = %v", rD1.Vars.Get("any"), expect1Rd1Var)
	}

	isFound2, _, rD2 := ro.Match("/v2/users/63029905408d8ed70d411662/update/jobs/63029924b2584a856cbb8baf/get")
	expect2UserId := "63029905408d8ed70d411662"
	output2UserId := rD2.Vars.Get("userId")
	expect2JobId := "63029924b2584a856cbb8baf"
	output2JobId := rD2.Vars.Get("jobId")
	if !isFound2 {
		test.Errorf("ro.Match(\"/v2/users/63029905408d8ed70d411662/update/jobs/63029924b2584a856cbb8baf/get\") = %v; expect = %v", isFound2, true)
	}
	if output2UserId != expect2UserId {
		test.Errorf("rD2.Vars.Get(\"userId\") = %v; expect = %v", output2UserId, expect2UserId)
	}
	if output2JobId != expect2JobId {
		test.Errorf("rD2.Vars.Get(\"jobId\") = %v; expect = %v", output2JobId, expect2JobId)
	}

	isFound3, _, rD3 := ro.Match("/v1/users/63029c3246fd350a3ffc276c/update/jobs/63029c3bb998dabb261d99a1/delete")
	expect3UserId := "63029c3246fd350a3ffc276c"
	output3UserId := rD3.Vars.Get("userId")
	expect3JobId := "63029c3bb998dabb261d99a1"
	output3JobId := rD3.Vars.Get("jobId")

	if !isFound3 {
		test.Errorf("ro.Match(\"/v1/users/63029c3246fd350a3ffc276c/update/jobs/63029c3bb998dabb261d99a1/delete\") = %v; expect = %v", isFound3, true)
	}
	if expect3UserId != output3UserId {
		test.Errorf("rD3.Vars.Get(\"userId\") = %v; expect = %v", output3UserId, expect3UserId)
	}
	if expect3JobId != output3JobId {
		test.Errorf("rD3.Vars.Get(\"jobId\") = %v; expect = %v", output3JobId, expect3JobId)
	}

	isFound4, _, rD4 := ro.Match("/v2/users/63029e1271f0bfaab1697c01/delete/jobs/63029e20f75076f6e6b8fdee/move")
	expect4UserId := "63029e1271f0bfaab1697c01"
	output4UserId := rD4.Vars.Get("userId")
	expect4JobId := "63029e20f75076f6e6b8fdee"
	output4JobId := rD4.Vars.Get("jobId")
	if !isFound4 {
		test.Errorf("ro.Match(\"/v2/users/63029e1271f0bfaab1697c01/update/jobs/63029e20f75076f6e6b8fdee/move\") = %v; expect = %v", isFound4, true)
	}
	if expect4UserId != output4UserId {
		test.Errorf("rD4.Vars.Get(\"userId\") = %v; expect = %v", output4UserId, expect4UserId)
	}
	if expect4JobId != output4JobId {
		test.Errorf("rD4.Vars.Get(\"jobId\") = %v; expect = %v", output4JobId, expect4JobId)
	}

	// nagative test
	isFound5, _, _ := ro.Match("/v2/users/gett/delete/jobs/6302a6d946dc9b4a37c1d281/get")
	if isFound5 {
		test.Errorf("ro.Match(\"/v2/users/gett/delete/jobs/6302a6d946dc9b4a37c1d281/get\") = %v; expect = %v", isFound5, false)
	}
}

func TestGroup(test *testing.T) {
	ro1 := NewRouter()
	for _, r1 := range []string{
		"/users/get",
		"/users/get/{userId}",
	} {
		ro1.Add(r1, func(req *core.Request, res core.ResponseExtender, next func()) {})
	}

	ro2 := NewRouter()
	for _, r2 := range []string{
		"/users/update/{userId}",
		"/users/delete/{userId}",
	} {
		ro2.Add(r2, func(req *core.Request, res core.ResponseExtender, next func()) {})
	}

	gr := NewRouter()
	gr.Group("/v1", ro1, ro2)

	_, matchedRoute, _ := gr.Match("/v1/users/update/123")
	expectMatchedRoute := "/v1/users/update/{userId}/"
	if matchedRoute != expectMatchedRoute {
		test.Errorf("gr.Match(\"/v1/users/update/123\") = %v; expect = %v", matchedRoute, expectMatchedRoute)
	}
}

func TestUse(test *testing.T) {
	ro1 := NewRouter()
	ro1.Use("as", func(req *core.Request, res core.ResponseExtender, next func()) {})
	ro1.Add("/as", nil)
}

func TestV(test *testing.T) {
	if os.Getenv("V") == "true" {
		ro := NewRouter()
		for _, r := range []string{
			"/v1/users/get/jobs/get",
			"/v2/users/{userId}/*/jobs/{jobId}/get",
			"/v1/users/{userId}/*/jobs/{jobId}/delete",
			"/v2/users/{userId}/*/jobs/{jobId}/*",
		} {
			ro.Add(r, func(req *core.Request, res core.ResponseExtender, next func()) {})
		}
		jsonStr, err := ro.visualize()
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
		} else {
			fmt.Println(string(jsonStr))
		}
	} else {
		test.Skip()
	}
}
