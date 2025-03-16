package devtool

import (
	"crypto/sha256"
	"encoding/base64"
)

type DevtoolMenu struct {
	REST []RESTComponent `json:"rest"`
}

type Devtool struct {
	Menus DevtoolMenu `json:"menus"`
}

type DevtoolBuilder struct {
	Menus DevtoolMenu
}

func NewDevtoolBuilder() *DevtoolBuilder {
	return &DevtoolBuilder{}
}

func (d *DevtoolBuilder) AddRESTMenu(controllerPath string, restComponent RESTComponent) *DevtoolBuilder {
	restComponent.ID = d.generateHandlerID(controllerPath + restComponent.Handler)
	d.Menus.REST = append(d.Menus.REST, restComponent)
	return d
}

func (d *DevtoolBuilder) Build() *Devtool {
	return &Devtool{
		Menus: d.Menus,
	}
}

func (d *DevtoolBuilder) generateHandlerID(str string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(str))
	hash := sha256.Sum256([]byte(encoded))
	encoded = base64.RawURLEncoding.EncodeToString(hash[:])
	return encoded[:12]
}
