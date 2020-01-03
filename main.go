package main

import (
	"fmt"
	"net/http"

	"github.com/dangduoc08/api-crud/libs/router"
)

func main() {
	r := router.Init()
	var handler router.RequestHandler = r

	handler.Get("/users", func(req *router.Request, res router.Response) {
		fmt.Println("up")
	})

	handler.Get("/users/:userId", func(req *router.Request, res router.Response) {
		var userId string = req.Params["userId"]
		res.Write([]byte(userId))
	})

	http.ListenAndServe(":8080", nil)
}
