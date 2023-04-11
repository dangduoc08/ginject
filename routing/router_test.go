package routing

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/dangduoc08/gooh/context"
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
		r.Add(path, "", nil)
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
	cases := map[string]string{
		"/lv1/ping":                             "/lv1/ping",
		"/lv1/642e948adef44c303cdd2df3":         "/lv1/{id1}",
		"/lv1/foo/bar":                          "/lv1/*",
		"/lv1":                                  "/lv1/*",
		"/lv1/lv2/pong":                         "/lv1/lv2/pong",
		"/lv1/lv2/642e951525714c1ec609b338":     "/lv1/lv2/{id2}",
		"/lv1/lv2/foo/bar":                      "/lv1/lv2/*",
		"/lv1/lv2/file/index.html":              "lv1/lv2/file/in*.html",
		"/lv1/lv2/file/in.html":                 "lv1/lv2/file/in*.html",
		"/lv1/lv2/file/image.png":               "lv1/lv2/file/image.*",
		"/lv1/lv2/lv3/peng":                     "/lv1/lv2/lv3/peng",
		"/lv1/lv2/lv3/642e95c4fbb2ad847ca96840": "/lv1/lv2/lv3/{id3}",
		"/lv1/lv2/lv3/foo/bar":                  "/lv1/lv2/lv3/*",
		"/lv1/lv2/lv3":                          "/lv1/lv2/lv3/*",
		"/lv1/lv2/lv3/file/index.html":          "lv1/lv2/lv3/file/in*.html",
		"/lv1/lv2/lv3/file/in.html":             "lv1/lv2/lv3/file/in*.html",
		"/lv1/lv2/lv3/file/image.jpeg":          "lv1/lv2/lv3/file/image.*",
		"/api/feeds/{feedApiId}/next/files/index.html/endpoint/any/things/after": "/*/feeds/{feed*Id}/*/files/*.html/*/",
		"/users/633b0aa5d7fc3578b655b9bd/friends/633b0af45f4fe7d45b00fba5":       "/users/{userId}/friends/{friendId}",
	}

	r := NewRoute()
	for _, path := range cases {
		r.All(path, nil)
	}

	for requestedRoute, expectedRoute := range cases {
		expectedRoute = AddMethodToRoute(expectedRoute, http.MethodPost)
		_, matchedRoute, _, _, _ := r.Match(requestedRoute, http.MethodPost)

		if matchedRoute != expectedRoute {
			t.Errorf("request = %v, matched = %v, expected = %v", requestedRoute, matchedRoute, expectedRoute)
		}
	}
}

func TestRouterGroup(t *testing.T) {
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
	expectMatchedRoute1 := AddMethodToRoute("/v1/users/update/{userId}/", http.MethodPatch)
	if outputMatchedRoute1 != expectMatchedRoute1 {
		t.Errorf("routerGr.match(\"/v1/users/update/123\") = %v; expect = %v", outputMatchedRoute1, expectMatchedRoute1)
	}
}

func TestRouterMiddleware(t *testing.T) {
	counter := 0

	handler1 := func(c *context.Context) {
		counter++
		c.Next()
	}

	r1 := NewRoute()
	r1.Use(handler1)
	r1.Get("/test", handler1)
	r1.For("/test", []string{})(handler1)
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
			r.Add(path, "", func(c *context.Context) {}, func(c *context.Context) {})
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
