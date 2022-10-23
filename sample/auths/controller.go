package auths

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/sample/products"
)

type AuthController struct {
	AuthProvider    AuthProvider
	ProductProvider products.ProductProvider
	Routers         map[string]gooh.Handler
}

func (authController AuthController) NewController() common.Controller {
	authController.Routers = map[string]gooh.Handler{
		routing.Get("/auths/signup/{username}"): authController.Signin,
	}

	return authController
}

func (authController AuthController) Signin(c gooh.Context) {
	authController.AuthProvider.Signin(c.URL.Query().Get("username"), c.URL.Query().Get("password"))
	c.Text("Signin controller")
}
