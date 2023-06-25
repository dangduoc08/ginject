package book

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/guard"
	"github.com/dangduoc08/gooh/sample/interceptor"
)

type BookController struct {
	common.Rest
	common.Guard
	common.Interceptor
	BookProvider BookProvider
}

func (bookController BookController) NewController() core.Controller {
	bookController.
		Prefix("v1").
		Prefix("books")
	bookController.BindGuard(guard.JWTGuard{})
	bookController.BindInterceptor(interceptor.CacheInterceptor{})

	return bookController
}

func (bookController BookController) READ(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) CREATE(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) READ_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) UPDATE_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) MODIFY_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) DELETE_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) READ_authors_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) CREATE_authors_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) DELETE_authors_BY_authorID_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) READ_reviews_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) CREATE_reviews_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) UPDATE_reviews_BY_reviewID_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) MODIFY_reviews_BY_reviewID_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}

func (bookController BookController) DELETE_reviews_BY_reviewID_OF_BY_bookID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return bookController.BookProvider.Handler(c, p, q, h)
}
