package common

import (
	"sync"

	"github.com/dangduoc08/gooh/routing"
)

type moduleBuilder struct {
	imports    []*Module
	providers  []Provider
	exports    []Provider
	presenters []Presenter
}

func ModuleBuilder() *moduleBuilder {
	return &moduleBuilder{
		imports:    []*Module{},
		providers:  []Provider{},
		exports:    []Provider{},
		presenters: []Presenter{},
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

func (m *moduleBuilder) Presenters(presenters ...Presenter) *moduleBuilder {
	m.presenters = append(m.presenters, presenters...)
	return m
}

func (m *moduleBuilder) Build() *Module {
	return &Module{
		Mutex:      &sync.Mutex{},
		Imports:    m.imports,
		Exports:    m.exports,
		Providers:  m.providers,
		Presenters: m.presenters,
		Router:     routing.NewRoute(),
	}
}
