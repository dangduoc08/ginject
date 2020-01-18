package main

import (
	"fmt"
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
	fmt.Println("Users", req.Middleware)
	res.JSON(http.StatusOK, `{
		"message": "Success",
		"code": %v
	}`, http.StatusOK)
}

func validateGetProduct(req *expr.Request, res expr.ResponseExtender, next func()) {
	defer res.Error(func(rec interface{}) {
		err := rec.(ServerError)
		res.JSON(err.Code, err)
	})

	var productId string = req.Params["productId"]

	for _, runeStr := range productId {
		if !unicode.IsDigit(runeStr) {
			panic(ServerError{
				Message: "Error: Ivalid product ID",
				Code:    http.StatusUnprocessableEntity,
			})
		}
	}

	req.Middleware["isProductIdValid"] = true

	next()
}

func serveST(req *expr.Request, res expr.ResponseExtender, next func()) {
	req.Middleware["S.T"] = "Served"

	next()
}

func getProduct(req *expr.Request, res expr.ResponseExtender, next func()) {
	fmt.Println("Products", req.Middleware)
	res.JSON(http.StatusOK, `{
		"message": "Success",
		"code": %v
	}`, http.StatusOK)
}

func main() {
	var app *expr.Express = expr.Init()
	var router expr.Router = app

	router.Get("/users/:userId", validateGetUser, getUser)
	router.Get("/products/:productId", validateGetProduct, serveST, getProduct)

	http.ListenAndServe(":8080", nil)
}
