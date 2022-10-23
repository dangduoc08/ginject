package routing

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/middlewares"
)

func TestRouteAdd(t *testing.T) {
	paths := []string{
		"/users/{userId}/",
		"/feeds/all/",
		"/users/{userId}/friends/all/",
		"/schools/{schoolId}/subjects/{subjectId}/{schoolId}",
	}
	r := NewRoute()

	for _, path := range paths {
		r.Add(path, nil)
	}

	expect1 := 11
	output1 := r.Trie.len()
	if output1 != expect1 {
		t.Errorf("r.Trie.Len() = %v; expect = %v", expect1, output1)
	}

	expect2 := map[string][]int{
		"schoolId":  {0, 2},
		"subjectId": {1},
	}
	output2 := r.Children["schools"].Children["$"].Children["subjects"].Children["$"].Children["$"].ParamKeys
	for key, indexs := range expect2 {
		if output2[key] == nil {
			t.Errorf("ParamKey[%v] = %v; expect ≠ %v", key, output2[key], nil)
		}

		for i, index := range indexs {
			if output2[key][i] != index {
				t.Errorf("ParamKey[%v][%v] = %v; expect = %v", key, i, output2[key][i], index)
			}
		}
	}
}

func TestRouterMatch(t *testing.T) {
	r := NewRoute()
	for _, route := range []string{
		"/v1/users/get/jobs/get",
		"/v2/users/get/*/jobs/{jobId}get",
		"/v2/users/{userId}/*/jobs/{jobId}/get",
		"/v1/users/{userId}/*/jobs/{jobId}/delete",
		"/v2/users/{userId}/*/jobs/{jobId}/*",
	} {
		r.Post(route)
		r.Put(route)
		r.Get(route)
	}

	isMatched1, _, _, _, _ := r.Match("/v1/users/get/jobs/get", http.MethodPost)
	if !isMatched1 {
		t.Errorf("r.Match(\"/v1/users/get/jobs/get\", http.MethodPost) = %v; expect = %v", isMatched1, true)
	}

	isMatched2, _, _, _, _ := r.Match("/v2/users/63029905408d8ed70d411662/update/jobs/63029924b2584a856cbb8baf/get", http.MethodPut)
	if !isMatched2 {
		t.Errorf("r.Match(\"/v2/users/63029905408d8ed70d411662/update/jobs/63029924b2584a856cbb8baf/get\", http.MethodPut) = %v; expect = %v", isMatched2, true)
	}

	isMatched3, _, _, _, _ := r.Match("/v1/users/63029c3246fd350a3ffc276c/update/jobs/63029c3bb998dabb261d99a1/delete", http.MethodGet)
	if !isMatched3 {
		t.Errorf("r.Match(\"/v1/users/63029c3246fd350a3ffc276c/update/jobs/63029c3bb998dabb261d99a1/delete\", http.MethodGet) = %v; expect = %v", isMatched3, true)
	}

	isMatched4, _, _, _, _ := r.Match("/v2/users/gett/delete/jobs/6302a6d946dc9b4a37c1d281/get", http.MethodHead)
	if isMatched4 {
		t.Errorf("r.Match(\"/v2/users/gett/delete/jobs/6302a6d946dc9b4a37c1d281/get\", http.MethodHead) = %v; expect = %v", isMatched4, false)
	}
}

func TestRouterGroup(test *testing.T) {
	r1 := NewRoute()
	for _, route := range []string{
		"/users/get",
		"/users/get/{userId}",
	} {
		r1.Delete(route)
		r1.Post(route)
		r1.Put(route)
		r1.Get(route)
		r1.Patch(route)
		r1.Head(route)
		r1.Options(route)
		r1.Head(route)
	}

	r2 := NewRoute()
	for _, route := range []string{
		"/users/update/{userId}",
		"/users/delete/{userId}",
	} {
		r2.Delete(route)
		r2.Post(route)
		r2.Put(route)
		r2.Get(route)
		r2.Patch(route)
		r2.Head(route)
		r2.Options(route)
		r2.Head(route)
	}

	gr := NewRoute()
	gr.Group("/v1", r1, r2)

	_, outputMatchedRoute1, _, _, _ := gr.Match("/v1/users/update/123", http.MethodPatch)
	expectMatchedRoute1 := addMethodToRoute("/v1/users/update/{userId}/", http.MethodPatch)
	if outputMatchedRoute1 != expectMatchedRoute1 {
		test.Errorf("routerGr.match(\"/v1/users/update/123\") = %v; expect = %v", outputMatchedRoute1, expectMatchedRoute1)
	}
}

func TestRouterMiddleware(t *testing.T) {
	handler1 := func(ctx *ctx.Context) {
		fmt.Println("handler1")
	}

	r1 := NewRoute()
	r1.Get("/test", handler1)
	r1.For("/test")(middlewares.RequestLogger)

	r1.Get("/test", handler1)
	r1.Use(middlewares.RequestLogger)

	r2 := NewRoute()
	r2.Use(middlewares.RequestLogger)
	r2.Get("/test", handler1)
	r2.For("/test")(middlewares.RequestLogger)
}

func TestRouteToJSON(t *testing.T) {
	if os.Getenv("v") == "true" {
		paths := []string{
			"/users/{userId}",
			"/feeds/all",
			"/users/{userId}/friends/{friendId}",
			"/schools/{schoolId}/subjects/{subjectId}/{subjectId}",
			"/schools/*",
			"/*/feeds/{feed***Id}/**/files/*.html/***",
		}
		r := NewRoute()

		for _, path := range paths {
			r.Add(path, func(c *ctx.Context) {}, func(c *ctx.Context) {})
		}

		json, err := r.ToJSON()
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
		} else {
			fmt.Println(json)
		}
	} else {
		t.Skip()
	}

}
