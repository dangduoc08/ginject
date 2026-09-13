package catalog

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/exception"
	dbstorage "github.com/dangduoc08/ginject/modules/storage"
)

const (
	storesTable     = "stores"
	categoriesTable = "categories"
	productsTable   = "products"
)

type StoreService struct {
	Store dbstorage.StoreService
}

func (svc StoreService) NewProvider() core.Provider {
	svc.stores().Schema(dbstorage.ModelSchema{
		Fields: []dbstorage.FieldSchema{
			{Name: "ownerId", Index: true},
		},
	})
	svc.categories().Schema(dbstorage.ModelSchema{
		Fields: []dbstorage.FieldSchema{
			{Name: "storeId", Index: true},
		},
	})
	svc.products().Schema(dbstorage.ModelSchema{
		Fields: []dbstorage.FieldSchema{
			{Name: "storeId", Index: true},
			{Name: "categoryId", Index: true},
		},
	})

	return svc
}

func (svc StoreService) stores() *dbstorage.Model {
	return svc.Store.Model(storesTable)
}

func (svc StoreService) categories() *dbstorage.Model {
	return svc.Store.Model(categoriesTable)
}

func (svc StoreService) products() *dbstorage.Model {
	return svc.Store.Model(productsTable)
}

func (svc StoreService) CreateStore(ownerID, name string) (Store, error) {
	existing, err := svc.stores().Find().Where("ownerId", dbstorage.OpEq, ownerID).Exec()
	if err != nil {
		return Store{}, exception.InternalServerErrorException("failed to look up store")
	}
	if len(existing) > 0 {
		return Store{}, exception.ConflictException("owner already has a store")
	}

	doc, err := svc.stores().Create(map[string]any{
		"ownerId": ownerID,
		"name":    name,
	})
	if err != nil {
		return Store{}, exception.InternalServerErrorException("failed to create store")
	}

	return storeFromDocument(doc), nil
}

func (svc StoreService) StoreByOwner(ownerID string) (Store, error) {
	docs, err := svc.stores().Find().Where("ownerId", dbstorage.OpEq, ownerID).Exec()
	if err != nil || len(docs) == 0 {
		return Store{}, exception.NotFoundException("store not found")
	}

	return storeFromDocument(docs[0]), nil
}

func (svc StoreService) CreateCategory(storeID, name string) (Category, error) {
	if _, err := svc.stores().FindByID(storeID); err != nil {
		return Category{}, exception.NotFoundException("store not found")
	}

	doc, err := svc.categories().Create(map[string]any{
		"storeId": storeID,
		"name":    name,
	})
	if err != nil {
		return Category{}, exception.InternalServerErrorException("failed to create category")
	}

	return categoryFromDocument(doc), nil
}

func (svc StoreService) Categories(storeID string, page, limit int) Page[Category] {
	docs, err := svc.categories().Find().Where("storeId", dbstorage.OpEq, storeID).Exec()
	if err != nil {
		return newPage[Category](nil, page, limit, 0)
	}

	categories := make([]Category, 0, len(docs))
	for _, doc := range docs {
		categories = append(categories, categoryFromDocument(doc))
	}

	return newPage(categories, page, limit, len(categories))
}

func (svc StoreService) Category(storeID, categoryID string) (Category, error) {
	doc, err := svc.categories().FindByID(categoryID)
	if err != nil || doc.Data["storeId"] != storeID {
		return Category{}, exception.NotFoundException("category not found")
	}

	return categoryFromDocument(doc), nil
}

func (svc StoreService) UpdateCategory(storeID, categoryID, name string) (Category, error) {
	doc, err := svc.categories().FindByID(categoryID)
	if err != nil || doc.Data["storeId"] != storeID {
		return Category{}, exception.NotFoundException("category not found")
	}

	if err := svc.categories().UpdateByID(categoryID, map[string]any{
		"storeId": storeID,
		"name":    name,
	}); err != nil {
		return Category{}, exception.InternalServerErrorException("failed to update category")
	}

	return Category{ID: categoryID, StoreID: storeID, Name: name}, nil
}

func (svc StoreService) DeleteCategory(storeID, categoryID string) error {
	doc, err := svc.categories().FindByID(categoryID)
	if err != nil || doc.Data["storeId"] != storeID {
		return exception.NotFoundException("category not found")
	}

	products, err := svc.products().Find().Where("categoryId", dbstorage.OpEq, categoryID).Exec()
	if err == nil {
		for _, product := range products {
			_ = svc.products().DeleteByID(product.ID)
		}
	}

	if err := svc.categories().DeleteByID(categoryID); err != nil {
		return exception.InternalServerErrorException("failed to delete category")
	}

	return nil
}

func (svc StoreService) CreateProduct(storeID, categoryID, name string, price float64) (Product, error) {
	doc, err := svc.categories().FindByID(categoryID)
	if err != nil || doc.Data["storeId"] != storeID {
		return Product{}, exception.NotFoundException("category not found")
	}

	created, err := svc.products().Create(map[string]any{
		"storeId":    storeID,
		"categoryId": categoryID,
		"name":       name,
		"price":      price,
	})
	if err != nil {
		return Product{}, exception.InternalServerErrorException("failed to create product")
	}

	return productFromDocument(created), nil
}

func (svc StoreService) Products(storeID, categoryID string, page, limit int) Page[Product] {
	docs, err := svc.products().
		Find().
		Where("storeId", dbstorage.OpEq, storeID).
		Where("categoryId", dbstorage.OpEq, categoryID).
		Exec()
	if err != nil {
		return newPage[Product](nil, page, limit, 0)
	}

	products := make([]Product, 0, len(docs))
	for _, doc := range docs {
		products = append(products, productFromDocument(doc))
	}

	return newPage(products, page, limit, len(products))
}

func (svc StoreService) Product(storeID, categoryID, productID string) (Product, error) {
	doc, err := svc.products().FindByID(productID)
	if err != nil || doc.Data["storeId"] != storeID || doc.Data["categoryId"] != categoryID {
		return Product{}, exception.NotFoundException("product not found")
	}

	return productFromDocument(doc), nil
}

func (svc StoreService) UpdateProduct(storeID, categoryID, productID, name string, price float64) (Product, error) {
	doc, err := svc.products().FindByID(productID)
	if err != nil || doc.Data["storeId"] != storeID || doc.Data["categoryId"] != categoryID {
		return Product{}, exception.NotFoundException("product not found")
	}

	if err := svc.products().UpdateByID(productID, map[string]any{
		"storeId":    storeID,
		"categoryId": categoryID,
		"name":       name,
		"price":      price,
	}); err != nil {
		return Product{}, exception.InternalServerErrorException("failed to update product")
	}

	return Product{ID: productID, StoreID: storeID, CategoryID: categoryID, Name: name, Price: price}, nil
}

func (svc StoreService) DeleteProduct(storeID, categoryID, productID string) error {
	doc, err := svc.products().FindByID(productID)
	if err != nil || doc.Data["storeId"] != storeID || doc.Data["categoryId"] != categoryID {
		return exception.NotFoundException("product not found")
	}

	if err := svc.products().DeleteByID(productID); err != nil {
		return exception.InternalServerErrorException("failed to delete product")
	}

	return nil
}

func storeFromDocument(doc dbstorage.Document) Store {
	ownerID, _ := doc.Data["ownerId"].(string)
	name, _ := doc.Data["name"].(string)

	return Store{ID: doc.ID, OwnerID: ownerID, Name: name}
}

func categoryFromDocument(doc dbstorage.Document) Category {
	storeID, _ := doc.Data["storeId"].(string)
	name, _ := doc.Data["name"].(string)

	return Category{ID: doc.ID, StoreID: storeID, Name: name}
}

func productFromDocument(doc dbstorage.Document) Product {
	storeID, _ := doc.Data["storeId"].(string)
	categoryID, _ := doc.Data["categoryId"].(string)
	name, _ := doc.Data["name"].(string)
	price, _ := doc.Data["price"].(float64)

	return Product{ID: doc.ID, StoreID: storeID, CategoryID: categoryID, Name: name, Price: price}
}
