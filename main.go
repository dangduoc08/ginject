package main

import (
	"net/http"

	"github.com/dangduoc08/api-crud/libs/router"
)

func main() {
	r := router.Init()
	var handler router.RequestHandler = r

	handler.Get("/users", func(req *router.Request, res router.ResponseExtender) {
		res.Send(req.URL.Path)
	})

	handler.Get("/users/:userId", func(req *router.Request, res router.ResponseExtender) {
		var userId string = req.Params["userId"]
		res.Send(userId)
	})

	handler.Post("/products/:productId", func(req *router.Request, res router.ResponseExtender) {
		var productId string = req.Params["productId"]
		res.Send(productId)
	})

	http.ListenAndServe(":8080", nil)
}
