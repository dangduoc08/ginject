package main

import (
	"net/http"

	"github.com/dangduoc08/api-crud/libs/router"
)

func getRoot(req *router.Request, res router.ResponseExtender) {
	res.Status(200).Send("This is root path")
}

func getUser(req *router.Request, res router.ResponseExtender) {
	var userId string = req.Params["userId"]
	res.Status(500).Send("User ID: %v", userId)
}

func getRawJSON(req *router.Request, res router.ResponseExtender) {
	rawJSON := `{
		"name": "John Doe",
		"age": 26,
		"skills": [
			"PHP",
			{
				"js": {
				"es6": true,
				"node_js": false
				}
			}
		],
		"scores": {
			"math": 10,
			"C++": "good"
		}
	}`
	res.JSON(rawJSON)
}

func getUnstructuredJSON(req *router.Request, res router.ResponseExtender) {
	type obj map[string]interface{}

	unstructuredJSON := obj{
		"name": "John Doe",
		"age":  26,
		"skills": []interface{}{
			"PHP",
			obj{
				"js": obj{
					"es6":     true,
					"node_js": false,
				},
			},
		},
		"scores": obj{
			"math": 10,
			"C++":  "good",
		},
	}
	res.JSON(map[string]interface{}(unstructuredJSON))
}

func getStructuredJSON(req *router.Request, res router.ResponseExtender) {
	type js struct {
		Es6     bool `json:"es6"`
		Node_js bool `json:"node_js"`
	}

	type score struct {
		Math int8   `json:"math"`
		C    string `json:"C++"`
	}

	type person struct {
		Name   string        `json:"name"`
		Age    int8          `json:"age"`
		Skills []interface{} `json:"skills"`
		Scores score         `json:"scores"`
	}

	sc := score{
		Math: 10,
		C:    "good",
	}

	sk := []interface{}{
		"PHP",
		map[string]js{
			"js": js{
				Es6:     true,
				Node_js: false,
			},
		},
	}

	structuredJSON := person{
		Name:   "John Doe",
		Age:    26,
		Scores: sc,
		Skills: sk,
	}

	res.JSON(structuredJSON)
}

func main() {
	r := router.Init()
	var handler router.RequestHandler = r

	handler.Get("/", getRoot).Get("/users/:userId", getUser)
	handler.Get("/raw-jsons", getRawJSON)
	handler.Get("/unstructured-jsons", getUnstructuredJSON)
	handler.Get("/structured-jsons", getStructuredJSON)

	http.ListenAndServe(":8080", nil)
}
