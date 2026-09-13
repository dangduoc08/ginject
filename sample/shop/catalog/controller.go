package catalog

import (
	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/sample/shop/accounts"
	"github.com/dangduoc08/ginject/sample/shop/catalog/dto"
)

type StoreController struct {
	common.HTTP
	common.Guard

	StoreService
}

func (instance StoreController) NewController() core.Controller {
	instance.BindGuard(accounts.AuthGuard{})

	return instance
}

func (instance StoreController) myStore(c *ctx.HTTPContext) Store {
	store, err := instance.StoreService.StoreByOwner(accounts.CurrentUser(c).ID)
	if err != nil {
		panic(err)
	}

	return store
}

func (instance StoreController) READ_store(c *ctx.HTTPContext) Store {
	return instance.myStore(c)
}

func (instance StoreController) CREATE_categories_OF_store(c *ctx.HTTPContext, categoryDTO dto.CategoryDTO) Category {
	category, err := instance.StoreService.CreateCategory(instance.myStore(c).ID, categoryDTO.Name)
	if err != nil {
		panic(err)
	}

	return category
}

func (instance StoreController) READ_categories_OF_store(c *ctx.HTTPContext, pagination dto.PaginationDTO) Page[Category] {
	return instance.StoreService.Categories(instance.myStore(c).ID, pagination.Page, pagination.Limit)
}

func (instance StoreController) READ_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param) Category {
	category, err := instance.StoreService.Category(instance.myStore(c).ID, param.Get("categoryId"))
	if err != nil {
		panic(err)
	}

	return category
}

func (instance StoreController) UPDATE_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param, categoryDTO dto.CategoryDTO) Category {
	category, err := instance.StoreService.UpdateCategory(instance.myStore(c).ID, param.Get("categoryId"), categoryDTO.Name)
	if err != nil {
		panic(err)
	}

	return category
}

func (instance StoreController) DELETE_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param) ginject.Map {
	if err := instance.StoreService.DeleteCategory(instance.myStore(c).ID, param.Get("categoryId")); err != nil {
		panic(err)
	}

	return ginject.Map{
		"message": "category deleted",
	}
}

func (instance StoreController) CREATE_products_OF_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param, productDTO dto.ProductDTO) Product {
	product, err := instance.StoreService.CreateProduct(instance.myStore(c).ID, param.Get("categoryId"), productDTO.Name, productDTO.Price)
	if err != nil {
		panic(err)
	}

	return product
}

func (instance StoreController) READ_products_OF_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param, pagination dto.PaginationDTO) Page[Product] {
	return instance.StoreService.Products(instance.myStore(c).ID, param.Get("categoryId"), pagination.Page, pagination.Limit)
}

func (instance StoreController) READ_products_BY_productId_OF_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param) Product {
	product, err := instance.StoreService.Product(instance.myStore(c).ID, param.Get("categoryId"), param.Get("productId"))
	if err != nil {
		panic(err)
	}

	return product
}

func (instance StoreController) UPDATE_products_BY_productId_OF_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param, productDTO dto.ProductDTO) Product {
	product, err := instance.StoreService.UpdateProduct(instance.myStore(c).ID, param.Get("categoryId"), param.Get("productId"), productDTO.Name, productDTO.Price)
	if err != nil {
		panic(err)
	}

	return product
}

func (instance StoreController) DELETE_products_BY_productId_OF_categories_BY_categoryId_OF_store(c *ctx.HTTPContext, param ginject.Param) ginject.Map {
	if err := instance.StoreService.DeleteProduct(instance.myStore(c).ID, param.Get("categoryId"), param.Get("productId")); err != nil {
		panic(err)
	}

	return ginject.Map{
		"message": "product deleted",
	}
}
