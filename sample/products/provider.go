package products

import (
	"fmt"

	"github.com/dangduoc08/gooh/common"
)

type Producter interface {
	GetProductByID(string) string
}

type ProductProvider struct {
	ProductEntities []ProductEntity
}

func (productProvider ProductProvider) New() common.Provider {
	return productProvider
}

func (productProvider *ProductProvider) GetProductByID(id string) string {
	fmt.Println("GetProductByID", id)
	return id
}
