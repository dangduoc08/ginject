package main

import (
	"log"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/modules/config"
	"github.com/dangduoc08/gooh/sample/auths"
	"github.com/dangduoc08/gooh/sample/categories"
	"github.com/dangduoc08/gooh/sample/products"
)

func main() {
	app := core.New()
	app.Use(middlewares.RequestLogger)

	appModule := core.ModuleBuilder().
		Imports(
			config.Module,
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
