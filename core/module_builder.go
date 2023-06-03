package core

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/utils"
)

type moduleBuilder struct {
	imports     []any
	providers   []Provider
	exports     []Provider
	controllers []Controller
}

func ModuleBuilder() *moduleBuilder {
	return &moduleBuilder{
		imports:     []any{},
		providers:   []Provider{},
		exports:     []Provider{},
		controllers: []Controller{},
	}
}

func (m *moduleBuilder) Imports(modules ...any) *moduleBuilder {
	m.imports = append(m.imports, modules...)
	return m
}

func (m *moduleBuilder) getModuleType() ([]*Module, []any) {
	staticModules := []*Module{}
	dynamicModules := []any{}
	errors := []string{}

	for _, arg := range m.imports {
		switch module := arg.(type) {
		case *Module:
			staticModules = append(staticModules, module)
		default:
			moduleType := reflect.TypeOf(module)
			isDynamic, e := isDynamicModule(moduleType.String())
			if e != nil {
				panic(e)
			}

			if isDynamic {
				dynamicModules = append(dynamicModules, module)
			} else {
				errors = append(errors, fmt.Sprintf("can't pass '%v' type as module", moduleType))
			}
		}
	}

	if len(errors) > 0 {
		panic(utils.FmtRed(strings.Join(errors, "\n       ")))
	}

	return staticModules, dynamicModules
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
	staticModules, dynamicModules := m.getModuleType()

	return &Module{
		Mutex:          &sync.Mutex{},
		staticModules:  staticModules,
		dynamicModules: dynamicModules,
		exports:        m.exports,
		providers:      m.providers,
		controllers:    m.controllers,
		Middleware:     &Middleware{},
		Middlewares: []struct {
			Method   string
			Route    string
			Handlers []func(*context.Context)
		}{},
		Guards: []struct {
			Method  string
			Route   string
			Handler any
		}{},
		MainHandlers: []struct {
			Method  string
			Route   string
			Handler any
		}{},
	}
}
