package main

import (
	"net/http"

	"github.com/dangduoc08/api-crud/express"
)

func test(req *express.Request, res express.ResponseExtender) {
	res.Send(200, "this is %v", "hihih")
}

func main() {
	app := express.Init()
	var router express.Router = app

	router.Get("/test", test)

	http.ListenAndServe(":8080", nil)
}
