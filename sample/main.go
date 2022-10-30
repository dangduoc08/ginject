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

	appModule := common.ModuleBuilder().
		Imports(
			products.Module,
			auths.Module,
			categories.Module,
		).
		Build()

	appModule.OnInit = func() {
		log.Default().Println("AppModule OnInit")
	}

	app.Create(appModule)

	log.Fatal(app.ListenAndServe(":8080", nil))
}
