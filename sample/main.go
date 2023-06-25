package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/modules/config"
	"github.com/dangduoc08/gooh/sample/book"
	"github.com/dangduoc08/gooh/sample/global"
	"github.com/dangduoc08/gooh/sample/order"
	"github.com/dangduoc08/gooh/sample/product"
)

func main() {
	app := core.New()
	app.
		Use(middlewares.Recovery, global.Middleware, middlewares.RequestLogger).
		BindGlobalGuards(global.Guard{}).
		BindGlobalInterceptors(global.Interceptor{})

	app.Create(
		core.ModuleBuilder().
			Imports(
				config.Register(config.ConfigModuleOptions{
					IsGlobal:          true,
					IsExpandVariables: true,
				}),
				book.BookModule,
				product.ProductModule,
				order.OrderModule,
			).
			Build(),
	)

	configService, ok := app.Get(config.ConfigService{}).(config.ConfigService)
	if !ok {
		panic(errors.New("cannot get config.ConfigService"))
	}

	port := fmt.Sprintf(":%v", configService.Get("PORT"))

	log.Fatal(app.ListenAndServe(port))
}
