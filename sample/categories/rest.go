package categories

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/products"
)

type CategoryController struct {
	core.Rest
	ProductProvider products.ProductProvider
}

func (categoryController CategoryController) Inject() core.Controller {
	categoryController.
		Prefix("categories").
		Get("/list", categoryController.List).
		Get("/{categoryId}/get", categoryController.List).
		Post("/create", categoryController.List).
		Put("/{categoryId}/update", categoryController.List).
		Get("/{categoryId}/products/{productId}/get", categoryController.GetProductOnCategoryId)

	return categoryController
}

func (categoryController *CategoryController) List(c gooh.Context) {
	c.Text("CategoryController List")
}

func (categoryController *CategoryController) GetProductOnCategoryId(c gooh.Context) {
	// categoryController.ProductProvider.GetProductByID("/{categoryId}/products/{productId}/get")
	c.Text("GetProductOnCategoryId List")
}
