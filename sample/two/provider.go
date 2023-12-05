package two

import (
	"fmt"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/five"
	"github.com/dangduoc08/gooh/sample/four"
	"github.com/dangduoc08/gooh/sample/six"
	"github.com/dangduoc08/gooh/sample/three"
)

type Provider struct {
	Name          string
	Logger        common.Logger
	ThreeProvider three.Provider
	FourProvider  four.Provider
	FiveProvider  five.Provider
	SixProvider   six.Provider
}

func (provider Provider) NewProvider() core.Provider {
	fmt.Println(provider.Name, "see", provider.ThreeProvider.Name)
	fmt.Println(provider.Name, "see", provider.FourProvider.Name)
	fmt.Println(provider.Name, "see", provider.FiveProvider.Name)
	fmt.Println(provider.Name, "see", provider.SixProvider.Name)
	return provider
}
