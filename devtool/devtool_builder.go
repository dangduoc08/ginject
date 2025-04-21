package devtool

import (
	"crypto/sha256"
	"encoding/base64"
)

type devtoolBuilder struct {
	controller DevtoolController
}

func DevtoolBuilder() *devtoolBuilder {
	return &devtoolBuilder{}
}

func (d *devtoolBuilder) generateHandlerID(str string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(str))
	hash := sha256.Sum256([]byte(encoded))
	encoded = base64.RawURLEncoding.EncodeToString(hash[:])
	return encoded[:12]
}

func (d *devtoolBuilder) AddREST(controllerPath string, restComponent RESTComponent) *devtoolBuilder {
	restComponent.ID = d.generateHandlerID(controllerPath + restComponent.Handler)
	d.controller.REST = append(d.controller.REST, restComponent)
	return d
}

func (d *devtoolBuilder) Build() *Devtool {
	return &Devtool{
		Controller: d.controller,
	}
}
