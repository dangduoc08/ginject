package order

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/book"
)

var OrderModule = func() *core.Module {
	orderController := OrderController{}
	orderProvider := OrderProvider{}

	module := core.ModuleBuilder().
		Imports(book.BookModule).
		Controllers(
			orderController,
		).
		Providers(
			orderProvider,
		).
		Exports(
			orderProvider,
		).
		Build()

	module.OnInit = func() {
		fmt.Println("OrderModule OnInit")
	}

	module.Middleware.
		Apply(func(c gooh.Context) {
			fmt.Println("Module bound middleware from OrderModule")
			c.Next()
		}).
		Exclude([]any{
			orderController.DELETE_BY_orderID,
			orderController.DELETE_items_BY_itemID_OF_BY_orderID,
		})

	return module
}()
