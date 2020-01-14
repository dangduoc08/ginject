package main

import (
	"fmt"
	"net/http"

	"github.com/dangduoc08/api-crud/express"
)

func middleware1(req *express.Request, res express.ResponseExtender, next func()) {
	fmt.Println("passed middleware 1")
	next()
}

func wrapper() express.Handler {

	return func(req *express.Request, res express.ResponseExtender, next func()) {
		fmt.Println(req.Params)
		next()
	}
}

func test(req *express.Request, res express.ResponseExtender, next func()) {
	res.Send(200, "Hi there")
}

func main() {
	app := express.Init()
	var router express.Router = app

	router.Get("/test/:params", middleware1, wrapper(), test)

	http.ListenAndServe(":8080", nil)
}
