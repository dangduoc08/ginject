package product

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/book"
	"github.com/dangduoc08/gooh/sample/guard"
	"github.com/dangduoc08/gooh/sample/interceptor"
)

type ProductController struct {
	common.Rest
	common.Guard
	common.Interceptor
	ProductProvider ProductProvider
	BookProvider    book.BookProvider
}

func (productController ProductController) NewController() core.Controller {
	productController.
		Prefix("v1").
		Prefix("products")
	productController.BindGuard(guard.JWTGuard{})
	productController.BindInterceptor(interceptor.CacheInterceptor{})

	return productController
}

func (productController ProductController) READ(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) CREATE(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) READ_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) UPDATE_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) MODIFY_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) DELETE_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) READ_categories_OF_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) CREATE_categories_OF_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) DELETE_categories_BY_categoryID_OF_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) READ_reviews_OF_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) CREATE_reviews_OF_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) READ_related_OF_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}

func (productController ProductController) READ_availability_OF_BY_productID(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return productController.BookProvider.Handler(c, p, q, h)
}
