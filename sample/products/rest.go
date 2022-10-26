package products

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
)

type ProductPresenter struct {
	common.Rest
}

func (productPresenter ProductPresenter) New() common.Presenter {
	productPresenter.
		Prefix("products").
		Get("/list", productPresenter.List).
		Get("/{productId}/get", productPresenter.List).
		Post("/create", productPresenter.List).
		Put("/{productId}/update", productPresenter.List)

	return productPresenter
}

func (productPresenter *ProductPresenter) List(c gooh.Context) {
	c.Text("ProductPresenter List")
}
