package products

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type ProductController struct {
	core.Rest
	ProductProvider        ProductProvider
	InjectedConfigProvider config.ConfigProvider // props
}

func (productController ProductController) Inject() core.Controller {
	productController.
		Prefix("products").
		Get("list", productController.List).
		Get("/{productId}/get", productController.List).
		Post("/create", productController.List).
		Put("/{productId}/update", productController.List)

	return productController
}

func (productController *ProductController) List(c gooh.Context) {
	fmt.Println("configProvider from LIST", productController.InjectedConfigProvider)
	productController.ProductProvider.GetProductByID("asdsa")
	c.Text("ProductController List")
}
