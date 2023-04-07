package userlist

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
)

type UserListController struct {
	core.Rest
	UserListProvider UserListProvider
}

func (userListController UserListController) handler(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
) any {
	return userListController.UserListProvider.handler(c, p, q)
}

func (userListController UserListController) Inject() core.Controller {
	userListController.
		Prefix("/users/{id}").
		Get("/owned_lists", userListController.handler).                 // User owned Lists
		Get("/members/{user_id}", userListController.handler).           // User memberships
		Post("/followed_lists", userListController.handler).             // Follow a list
		Delete("/followed_lists/{list_id}", userListController.handler). // Unfollow a list
		Get("/followed_lists", userListController.handler).              // User's followed Lists
		Post("/pinned_lists", userListController.handler).               // Pin a List
		Delete("/pinned_lists/{list_id}", userListController.handler).   // Unpin a List
		Get("/pinned_lists", userListController.handler)                 // User's pinned Lists

	return userListController
}
