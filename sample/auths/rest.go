package auths

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/categories"
	"github.com/dangduoc08/gooh/sample/products"
)

type AuthController struct {
	core.Rest
	AuthProvider     AuthProvider
	ProductProvider  products.ProductProvider
	CategoryProvider categories.CategoryProvider
}

func (authController AuthController) Inject() core.Controller {
	authController.
		Prefix("auths").
		Post("/signin", authController.Signin).
		Get("/ping", authController.Signin).
		Post("/signup", authController.Signin)

	return authController
}

func (authController *AuthController) Signin(c gooh.Context) {
	authController.AuthProvider.Signin(c.URL.Query().Get("username"), c.URL.Query().Get("password"))
	authController.ProductProvider.GetProductByID("Signin")
	c.JSON(gooh.Map{
		"name": "Hello World!",
	})
}
