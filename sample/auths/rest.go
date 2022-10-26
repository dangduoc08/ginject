package auths

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/sample/categories"
	"github.com/dangduoc08/gooh/sample/products"
)

type AuthPresenter struct {
	common.Rest
	AuthProvider     AuthProvider
	ProductProvider  products.ProductProvider
	CategoryProvider categories.CategoryProvider
}

func (authPresenter AuthPresenter) New() common.Presenter {
	authPresenter.
		Prefix("auths").
		Post("/signin", authPresenter.Signin).
		Get("/ping", authPresenter.Signin).
		Post("/signup", authPresenter.Signin)

	return authPresenter
}

func (authPresenter *AuthPresenter) Signin(c gooh.Context) {
	authPresenter.AuthProvider.Signin(c.URL.Query().Get("username"), c.URL.Query().Get("password"))
	authPresenter.ProductProvider.GetProductByID("Signin")
	c.JSON(gooh.Map{
		"name": "Hello World!",
	})
}
