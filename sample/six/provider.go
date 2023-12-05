package six

import (
	"fmt"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/seven"
)

type Provider struct {
	Name          string
	Logger        common.Logger
	SevenProvider seven.Provider
}

func (provider Provider) NewProvider() core.Provider {
	fmt.Println(provider.Name, "see", provider.SevenProvider.Name)
	return provider
}
