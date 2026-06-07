package catalog

import (
	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/sample/shop/accounts"
	"github.com/dangduoc08/ginject/sample/shop/catalog/dto"
)

// StoreController lets an authenticated user manage their own store's
// catalog: read the store, and create/read/update/delete its categories and
// the products placed under them. Every route requires AuthGuard, and every
// lookup is scoped to the caller's store, so one user can never see or
// modify another user's catalog.
type StoreController struct {
	common.REST
	common.Guard

	StoreService
}

func (instance StoreController) NewController() core.Controller {
	instance.BindGuard(accounts.AuthGuard{})

	return instance
}

// myStore resolves the store owned by the authenticated caller.
func (instance StoreController) myStore(c *ctx.Context) Store {
	store, err := instance.StoreService.StoreByOwner(accounts.CurrentUser(c).ID)
	if err != nil {
		panic(err)
	}

	return store
}

// READ_store returns the caller's own store.
//
//	GET /store
func (instance StoreController) READ_store(c *ctx.Context) Store {
	return instance.myStore(c)
}

// CREATE_categories_OF_store adds a category to the caller's store.
//
//	POST /store/categories
//	body: { "name": string }
func (instance StoreController) CREATE_categories_OF_store(c *ctx.Context, categoryDTO dto.CategoryDTO) Category {
	category, err := instance.StoreService.CreateCategory(instance.myStore(c).ID, categoryDTO.Name)
	if err != nil {
		panic(err)
	}

	return category
}

// READ_categories_OF_store lists the categories in the caller's store.
//
//	GET /store/categories?page=<int>&limit=<int>
func (instance StoreController) READ_categories_OF_store(c *ctx.Context, pagination dto.PaginationDTO) Page[Category] {
	return instance.StoreService.Categories(instance.myStore(c).ID, pagination.Page, pagination.Limit)
}

// READ_categories_BY_categoryId_OF_store returns one category from the
// caller's store.
//
//	GET /store/categories/:categoryId
func (instance StoreController) READ_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param) Category {
	category, err := instance.StoreService.Category(instance.myStore(c).ID, param.Get("categoryId"))
	if err != nil {
		panic(err)
	}

	return category
}

// UPDATE_categories_BY_categoryId_OF_store renames a category in the
// caller's store.
//
//	PUT /store/categories/:categoryId
//	body: { "name": string }
func (instance StoreController) UPDATE_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param, categoryDTO dto.CategoryDTO) Category {
	category, err := instance.StoreService.UpdateCategory(instance.myStore(c).ID, param.Get("categoryId"), categoryDTO.Name)
	if err != nil {
		panic(err)
	}

	return category
}

// DELETE_categories_BY_categoryId_OF_store removes a category, and every
// product placed under it, from the caller's store.
//
//	DELETE /store/categories/:categoryId
func (instance StoreController) DELETE_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param) ginject.Map {
	if err := instance.StoreService.DeleteCategory(instance.myStore(c).ID, param.Get("categoryId")); err != nil {
		panic(err)
	}

	return ginject.Map{
		"message": "category deleted",
	}
}

// CREATE_products_OF_categories_BY_categoryId_OF_store adds a product to a
// category in the caller's store.
//
//	POST /store/categories/:categoryId/products
//	body: { "name": string, "price": number }
func (instance StoreController) CREATE_products_OF_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param, productDTO dto.ProductDTO) Product {
	product, err := instance.StoreService.CreateProduct(instance.myStore(c).ID, param.Get("categoryId"), productDTO.Name, productDTO.Price)
	if err != nil {
		panic(err)
	}

	return product
}

// READ_products_OF_categories_BY_categoryId_OF_store lists the products in a
// category of the caller's store.
//
//	GET /store/categories/:categoryId/products?page=<int>&limit=<int>
func (instance StoreController) READ_products_OF_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param, pagination dto.PaginationDTO) Page[Product] {
	return instance.StoreService.Products(instance.myStore(c).ID, param.Get("categoryId"), pagination.Page, pagination.Limit)
}

// READ_products_BY_productId_OF_categories_BY_categoryId_OF_store returns one
// product from a category of the caller's store.
//
//	GET /store/categories/:categoryId/products/:productId
func (instance StoreController) READ_products_BY_productId_OF_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param) Product {
	product, err := instance.StoreService.Product(instance.myStore(c).ID, param.Get("categoryId"), param.Get("productId"))
	if err != nil {
		panic(err)
	}

	return product
}

// UPDATE_products_BY_productId_OF_categories_BY_categoryId_OF_store replaces
// the name and price of a product in the caller's store.
//
//	PUT /store/categories/:categoryId/products/:productId
//	body: { "name": string, "price": number }
func (instance StoreController) UPDATE_products_BY_productId_OF_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param, productDTO dto.ProductDTO) Product {
	product, err := instance.StoreService.UpdateProduct(instance.myStore(c).ID, param.Get("categoryId"), param.Get("productId"), productDTO.Name, productDTO.Price)
	if err != nil {
		panic(err)
	}

	return product
}

// DELETE_products_BY_productId_OF_categories_BY_categoryId_OF_store removes a
// product from a category of the caller's store.
//
//	DELETE /store/categories/:categoryId/products/:productId
func (instance StoreController) DELETE_products_BY_productId_OF_categories_BY_categoryId_OF_store(c *ctx.Context, param ginject.Param) ginject.Map {
	if err := instance.StoreService.DeleteProduct(instance.myStore(c).ID, param.Get("categoryId"), param.Get("productId")); err != nil {
		panic(err)
	}

	return ginject.Map{
		"message": "product deleted",
	}
}
