package list

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
)

type ListController struct {
	core.Rest
	ListProvider ListProvider
}

func (listController ListController) handler(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
) any {
	return listController.ListProvider.handler(c, p, q)
}

func (listController ListController) Inject() core.Controller {
	listController.
		Prefix("lists").
		Get("{id}", listController.handler).                      // List by ID
		Post("", listController.handler).                         // Create a List
		Put("{id}", listController.handler).                      // Update a List
		Delete("{id}", listController.handler).                   // Delete a List
		Get("{id}/tweets", listController.handler).               // List Tweets lookup
		Post("{id}/members", listController.handler).             // Add a member
		Delete("{id}/members/{user_id}", listController.handler). // Remove a member
		Get("{id}/members", listController.handler).              // Members lookup
		Get("{id}/followers", listController.handler)             // Follower lookup

	return listController
}
