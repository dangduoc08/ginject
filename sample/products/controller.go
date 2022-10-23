package products

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/sample/categories"
)

type ProductController struct {
	CategoryProvider categories.CategoryProvider
	Routers          map[string]gooh.Handler
}

func (productProvider ProductController) NewController() common.Controller {
	productProvider.Routers = map[string]gooh.Handler{
		routing.Get("/products/list"):                                    productProvider.List,
		routing.Get("/products/{productId}/get"):                         productProvider.List,
		routing.Post("/products/create"):                                 productProvider.List,
		routing.Put("/products/{productId}/update"):                      productProvider.List,
		routing.Get("/categories/{categoryId}/products/{productId}/get"): productProvider.List,
	}

	return productProvider
}

func (productController ProductController) List(c gooh.Context) {
	c.Text("ProductController List")
}
