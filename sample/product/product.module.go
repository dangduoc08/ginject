package product

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/book"
)

var ProductModule = func() *core.Module {
	productController := ProductController{}
	productProvider := ProductProvider{}

	module := core.ModuleBuilder().
		Imports(book.BookModule).
		Controllers(
			productController,
		).
		Providers(
			productProvider,
		).
		Exports(
			productProvider,
		).
		Build()

	module.OnInit = func() {
		fmt.Println("ProductModule OnInit")
	}

	module.Middleware.
		Apply(func(c gooh.Context) {
			fmt.Println("Module bound middleware from ProductModule")
			c.Next()
		}).
		Exclude([]any{
			productController.DELETE_BY_productID,
			productController.DELETE_categories_BY_categoryID_OF_BY_productID,
		})

	return module
}()
