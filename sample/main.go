package main

import (
	"log"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/sample/auths"
	"github.com/dangduoc08/gooh/sample/categories"
	"github.com/dangduoc08/gooh/sample/products"
)

func main() {
	app := core.New()
	app.Use(middlewares.RequestLogger)

	module := common.Module{
		Providers: []common.Provider{
			auths.AuthProvider{},
			products.ProductProvider{},
			categories.CategoryProvider{},
		},
		Controllers: []common.Controller{
			auths.AuthController{},
			products.ProductController{},
			categories.CategoryController{},
		},
	}

	module.Create(app)

	log.Fatal(app.ListenAndServe(":8080", nil))
}
