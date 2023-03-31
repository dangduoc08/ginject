package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/modules/config"
)

type Controller struct {
	core.Rest
}

func (controller Controller) Inject() core.Controller {
	controller.
		Prefix("/users").
		All("/{userId}/all", func(c gooh.Context) {
			c.JSON(gooh.Map{
				"name": "Hello World!",
			})
		})

	return controller
}

func main() {
	app := core.New()
	app.Use(middlewares.RequestLogger, middlewares.Recovery)

	app.Create(
		core.ModuleBuilder().
			Imports(
				config.Register(config.ConfigModuleOptions{
					IsGlobal: true,
				}),
			).
			Controllers(Controller{}).
			Build(),
	)

	configService, ok := app.Get(config.ConfigService{}).(config.ConfigService)
	if !ok {
		panic(errors.New("cannot get config.ConfigService"))
	}

	port := fmt.Sprintf(":%v", configService.Get("PORT"))

	log.Fatal(app.ListenAndServe(port))
}
