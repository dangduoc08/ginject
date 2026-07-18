package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/color"
)

type moduleBuilder struct {
	imports     []any
	providers   []Provider
	controllers []Controller
}

func ModuleBuilder() *moduleBuilder {
	return &moduleBuilder{
		imports:     []any{},
		providers:   []Provider{},
		controllers: []Controller{},
	}
}

// fix later
// change to private fields
// handle for devtool
type WSMiddlewareLayer struct {
	// controllerPath string
	// handlerName    string
	EventName string
	Handlers  []func(*ctx.WSContext)
}

type WSCommonLayer struct {
	// controllerPath string
	// name           string
	EventName string
	Handler   any
}

func (m *moduleBuilder) Imports(modules ...any) *moduleBuilder {
	m.imports = append(m.imports, modules...)
	return m
}

func (m *moduleBuilder) getModuleType() ([]*Module, []any) {
	staticModules := make([]*Module, 0, len(m.imports))
	var dynamicModules []any
	var errors []string

	for _, arg := range m.imports {
		switch module := arg.(type) {
		case *Module:
			staticModules = append(staticModules, module)
		default:
			moduleType := reflect.TypeOf(module)
			if isDynamicModule(moduleType.String()) {
				dynamicModules = append(dynamicModules, module)
			} else {
				errors = append(errors, fmt.Sprintf("can't pass '%v' type as module", moduleType))
			}
		}
	}

	if len(errors) > 0 {
		panic(color.FmtRed("invalid module: %s", strings.Join(errors, "\n       ")))
	}

	return staticModules, dynamicModules
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

	module := &Module{
		Mutex:          &sync.Mutex{},
		staticModules:  staticModules,
		dynamicModules: dynamicModules,
		providers:      m.providers,
		controllers:    m.controllers,

		RESTExceptionFilters: []common.RESTLayer{},
		RESTMiddlewares:      []common.RESTLayer{},
		RESTGuards:           []common.RESTLayer{},
		RESTInterceptors:     []common.RESTLayer{},
		RESTMainHandlers:     []common.RESTLayer{},

		WSGuards:           []common.WSLayer{},
		WSInterceptors:     []common.WSLayer{},
		WSExceptionFilters: []common.WSLayer{},
		WSMainHandlers:     []common.WSLayer{},
	}

	module.id = strconv.FormatUint(uint64(reflect.ValueOf(module).Pointer()), 10)
	return module
}
