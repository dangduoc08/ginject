package main

import (
	"net/http"
	"unicode"

	expr "github.com/dangduoc08/api-crud/express"
)

type ServerError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func validateGetUser(req *expr.Request, res expr.ResponseExtender, next func()) {
	defer res.Error(func(rec interface{}) {
		err := rec.(ServerError)
		res.JSON(err.Code, err)
	})

	var userId string = req.Params["userId"]

	for _, runeStr := range userId {
		if !unicode.IsDigit(runeStr) {
			panic(ServerError{
				Message: "Error: Ivalid user ID",
				Code:    http.StatusUnprocessableEntity,
			})
		}
	}

	req.Middleware["isUserIdValid"] = true

	next()
}

func getUser(req *expr.Request, res expr.ResponseExtender, next func()) {
	res.JSON(http.StatusOK, `{
		"message": "Success",
		"code": %v
	}`, http.StatusOK)
}

func all(req *expr.Request, res expr.ResponseExtender, next func()) {
	res.Send(200, "Matched all URL")
}

func slash(req *expr.Request, res expr.ResponseExtender, next func()) {
	res.Send(200, "You accessed root")
}

func main() {
	var app *expr.Express = expr.Init()
	var router expr.Router = app

	router.Get("/users/:userId", validateGetUser, getUser)
	router.Get("*", all)
	router.Get("/*", all)
	router.Get("/", slash)

	http.ListenAndServe(":8080", nil)
}
