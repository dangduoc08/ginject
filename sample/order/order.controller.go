package order

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/book"
	"github.com/dangduoc08/gooh/sample/guard"
	"github.com/dangduoc08/gooh/sample/interceptor"
)

type OrderController struct {
	common.Rest
	common.Guard
	common.Interceptor
	OrderProvider OrderProvider
	BookProvider  book.BookProvider
}

func (orderController OrderController) NewController() core.Controller {
	orderController.
		Prefix("v1").
		Prefix("orders")
	orderController.BindGuard(guard.JWTGuard{})
	orderController.BindInterceptor(interceptor.CacheInterceptor{})

	return orderController
}

func (orderController OrderController) READ(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) CREATE(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) READ_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) UPDATE_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) MODIFY_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) DELETE_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) READ_items_OF_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) CREATE_items_OF_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) DELETE_items_BY_itemID_OF_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) READ_payments_OF_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}

func (orderController OrderController) CREATE_payments_OF_BY_orderID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return orderController.BookProvider.Handler(c, p, q, h)
}
