package shop

import (
	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/accounts"
	accountsdto "github.com/dangduoc08/ginject/sample/shop/accounts/dto"
	"github.com/dangduoc08/ginject/sample/shop/catalog"
)

// UsersController handles account registration. Every new account is
// provisioned with its own Store, ready to manage categories and products.
// It's the one controller that spans both the accounts and catalog modules.
type UsersController struct {
	common.REST

	accounts.UserService
	catalog.StoreService
}

func (instance UsersController) NewController() core.Controller {
	return instance
}

// CREATE_users registers a new account and provisions its store.
//
//	POST /users
//	body: { "email": string, "name": string, "password": string }
func (instance UsersController) CREATE_users(userDTO accountsdto.UserDTO) ginject.Map {
	user, err := instance.UserService.Register(userDTO.Email, userDTO.Name, userDTO.Password)
	if err != nil {
		panic(err)
	}

	store, err := instance.StoreService.CreateStore(user.ID, userDTO.Name+"'s Store")
	if err != nil {
		panic(err)
	}

	return ginject.Map{
		"user":  user,
		"store": store,
	}
}
