package book

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
)

var BookModule = func() *core.Module {
	bookController := BookController{}
	bookProvider := BookProvider{}

	module := core.ModuleBuilder().
		Controllers(
			bookController,
		).
		Providers(
			bookProvider,
		).
		Exports(
			bookProvider,
		).
		Build()

	module.OnInit = func() {
		fmt.Println("BookModule OnInit")
	}

	module.Middleware.
		Apply(func(c gooh.Context) {
			fmt.Println("Module bound middleware from BookModule")
			c.Next()
		}).
		Exclude([]any{
			bookController.DELETE_BY_bookID,
			bookController.DELETE_authors_BY_authorID_OF_BY_bookID,
			bookController.DELETE_reviews_BY_reviewID_OF_BY_bookID,
		})

	return module
}()
