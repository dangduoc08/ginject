package core

import (
	"sync"

	"github.com/dangduoc08/gooh/routing"
)

type moduleBuilder struct {
	imports     []*Module
	providers   []Provider
	exports     []Provider
	controllers []Controller
}

func ModuleBuilder() *moduleBuilder {
	return &moduleBuilder{
		imports:     []*Module{},
		providers:   []Provider{},
		exports:     []Provider{},
		controllers: []Controller{},
	}
}

func (m *moduleBuilder) Imports(modules ...*Module) *moduleBuilder {
	m.imports = append(m.imports, modules...)
	return m
}

func (m *moduleBuilder) Exports(providers ...Provider) *moduleBuilder {
	m.exports = append(m.exports, providers...)
	return m
}

func (m *moduleBuilder) Providers(providers ...Provider) *moduleBuilder {
	m.providers = append(m.providers, providers...)
	return m
}

func (m *moduleBuilder) Controllers(controllers ...Controller) *moduleBuilder {
	m.controllers = append(m.controllers, controllers...)
	return m
}

func (m *moduleBuilder) Build() *Module {
	return &Module{
		Mutex:       &sync.Mutex{},
		Imports:     m.imports,
		Exports:     m.exports,
		Providers:   m.providers,
		Controllers: m.controllers,
		Router:      routing.NewRoute(),
	}
}
