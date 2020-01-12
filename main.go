package main

import (
	"github.com/dangduoc08/api-crud/libs/router"
	"net/http"
)

func handleRoot(req *router.Request, res router.ResponseExtender) {
	res.Status(200).Send("This is root path")
}

func getUser(req *router.Request, res router.ResponseExtender) {
	var userId string = req.Params["userId"]
	res.Status(500).Send("User ID: %v", userId)
}

func main() {
	r := router.Init()
	var handler router.RequestHandler = r

	handler.Get("/", handleRoot).Get("/users/:userId", getUser)

	http.ListenAndServe(":8080", nil)
}
