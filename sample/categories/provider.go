package categories

import (
	"github.com/dangduoc08/gooh/common"
)

type Categorier interface {
	GetProductByID(string) string
}

type CategoryProvider struct {
	CategoryEntities []CategoryEntity
}

func (categoryProvider CategoryProvider) New() common.Provider {
	return categoryProvider
}

func (categoryProvider *CategoryProvider) GetProductByID(id string) string {
	return id
}
