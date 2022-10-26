package categories

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/sample/products"
)

type CategoryPresenter struct {
	common.Rest
	ProductProvider products.ProductProvider
}

func (categoryPresenter CategoryPresenter) New() common.Presenter {
	categoryPresenter.
		Prefix("categories").
		Get("/list", categoryPresenter.List).
		Get("/{categoryId}/get", categoryPresenter.List).
		Post("/create", categoryPresenter.List).
		Put("/{categoryId}/update", categoryPresenter.List).
		Get("/{categoryId}/products/{productId}/get", categoryPresenter.GetProductOnCategoryId)

	return categoryPresenter
}

func (categoryPresenter *CategoryPresenter) List(c gooh.Context) {
	c.Text("CategoryPresenter List")
}

func (categoryPresenter *CategoryPresenter) GetProductOnCategoryId(c gooh.Context) {
	categoryPresenter.ProductProvider.GetProductByID("/{categoryId}/products/{productId}/get")
	c.Text("GetProductOnCategoryId List")
}
