package three

import (
	"fmt"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/five"
	"github.com/dangduoc08/gooh/sample/four"
)

type Provider struct {
	Name         string
	Logger       common.Logger
	FourProvider four.Provider
	FiveProvider five.Provider
}

func (provider Provider) NewProvider() core.Provider {
	fmt.Println(provider.Name, "see", provider.FourProvider.Name)
	fmt.Println(provider.Name, "see", provider.FiveProvider.Name)
	return provider
}
