package products

import (
	"github.com/dangduoc08/gooh/common"
)

type Producter interface {
	GetProductByID(string) string
}

type ProductProvider struct {
	ProductEntities []ProductEntity
}

func (productProvider ProductProvider) NewProvider() common.Provider {
	return productProvider
}

func (productProvider ProductProvider) GetProductByID(id string) string {
	return id
}
