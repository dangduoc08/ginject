package categories

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/routing"
)

type CategoryController struct {
	Routers map[string]gooh.Handler
}

func (categoryController CategoryController) NewController() common.Controller {
	categoryController.Routers = map[string]gooh.Handler{
		routing.Get("/categories/list"): categoryController.List,
	}

	return categoryController
}

func (categoryController CategoryController) List(c gooh.Context) {
	c.Text("CategoryController List")
}
