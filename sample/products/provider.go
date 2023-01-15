package products

import (
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type Producter interface {
	GetProductByID(string) string
}

type ProductProvider struct {
	ProductEntities        []ProductEntity       // state
	InjectedConfigProvider config.ConfigProvider // props
}

func (productProvider ProductProvider) Inject() core.Provider {
	return productProvider
}

func (productProvider *ProductProvider) GetProductByID(id string) string {
	return id
}
