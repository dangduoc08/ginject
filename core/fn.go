package core

import (
	"fmt"
	"go/token"
	"reflect"
	"regexp"
	"strings"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/utils"
)

func isDynamicModule(moduleType string) (bool, error) {
	return regexp.Match(`^func\(.*\*core.Module$`, []byte(moduleType))
}

// function were re-use at
// dynamic module
// isInjectable handler
// checking pipe
// due to all this patterns inject dependencies as function arguments
func getFnArgs(f any, injectedProviders map[string]Provider, cb func(string, int, reflect.Value)) {
	injectableFnType := reflect.TypeOf(f)
	for i := 0; i < injectableFnType.NumIn(); i++ {
		argType := injectableFnType.In(i)
		arg := argType.PkgPath() + "/" + argType.String()
		newArg := reflect.New(argType).Elem()
		argAnyValue := newArg.Interface()

		if bodyPipeable, isImplBodyPipeable := argAnyValue.(common.BodyPipeable); isImplBodyPipeable {
			newArg = injectDependencies(bodyPipeable, "pipe", injectedProviders)
			cb(BODY_PIPEABLE, i, newArg)
		} else if queryPipeable, isImplQueryPipeable := argAnyValue.(common.QueryPipeable); isImplQueryPipeable {
			newArg = injectDependencies(queryPipeable, "pipe", injectedProviders)
			cb(QUERY_PIPEABLE, i, newArg)
		} else if headerPipeable, isImplHeaderPipeable := argAnyValue.(common.HeaderPipeable); isImplHeaderPipeable {
			newArg = injectDependencies(headerPipeable, "pipe", injectedProviders)
			cb(HEADER_PIPEABLE, i, newArg)
		} else if paramPipeable, isImplParamPipeable := argAnyValue.(common.ParamPipeable); isImplParamPipeable {
			newArg = injectDependencies(paramPipeable, "pipe", injectedProviders)
			cb(PARAM_PIPEABLE, i, newArg)
		} else {
			cb(arg, i, newArg)
		}
	}
}

func isInjectableHandler(handler any, injectedProviders map[string]Provider) error {
	var e error

	getFnArgs(handler, injectedProviders, func(arg string, i int, pipeValue reflect.Value) {
		if _, ok := dependencies[arg]; !ok {
			e = fmt.Errorf(
				"can't resolve dependencies of '%v'. Please make sure that the argument dependency at index [%v] is available in the handler",
				reflect.TypeOf(handler).String(),
				i,
			)
		}
	})

	return e
}

func isInjectedProvider(providerFieldType reflect.Type) bool {
	instance := reflect.New(providerFieldType)
	_, ok := instance.Interface().(Provider)
	return ok
}

func genProviderKey(p Provider) string {
	return genFieldKey(reflect.TypeOf(p))
}

func genFieldKey(t reflect.Type) string {
	return t.PkgPath() + "/" + t.String()
}

func createStaticModuleFromDynamicModule(dynamicModule any, injectedProviders map[string]Provider) *Module {
	dynamicModuleType := reflect.TypeOf(dynamicModule)
	localArgsIndex := make(map[string]int)
	localArgs := []reflect.Value{}
	globalArgs := []reflect.Value{}
	args := []reflect.Value{}

	genError := func(dynamicModuleType reflect.Type, dynamicArgKey string, index int) error {
		return fmt.Errorf(
			utils.FmtRed(
				"can't resolve argument of '%v'. Please make sure that the argument '%v' at index [%v] is available in the injected providers",
				strings.Replace(dynamicModuleType.String(), ") *core.Module", ")", 1),
				dynamicArgKey,
				index,
			),
		)
	}

	getFnArgs(dynamicModule, globalProviders, func(dynamicArgKey string, i int, pipeValue reflect.Value) {
		// inject provider priorities
		// local inject
		// global inject
		// inner packages

		// check if an injected provider with the same type exists
		if injectedProviders[dynamicArgKey] != nil {

			// if an injected provider exists, append it to the list of arguments
			localArgs = append(localArgs, reflect.ValueOf(injectedProviders[dynamicArgKey]))
			localArgsIndex[dynamicArgKey] = i
		} else if globalProviders[dynamicArgKey] != nil {

			// if an injected provider doesn't exist, check if a global provider with the same type exists
			// if a global provider exists, append it to the list of arguments
			globalArgs = append(globalArgs, reflect.ValueOf(globalProviders[dynamicArgKey]))
		} else {
			panic(genError(dynamicModuleType, dynamicArgKey, i))
		}
	})

	args = append(args, append(localArgs, globalArgs...)...)

	// call the dynamic module with the list of arguments and convert the result to a static module
	staticModule := reflect.ValueOf(dynamicModule).Call(args)[0].Interface().(*Module)

	// recursion injection
	injectModule := staticModule.NewModule()

	// only import providers which exported
	if len(injectModule.exports) > 0 {
		staticModule.providers = append(staticModule.providers, injectModule.exports...)
	}

	// recheck inject dependencies
	actualLocalProviderMap := make(map[string]bool)
	for _, provider := range staticModule.providers {
		actualLocalProviderMap[genProviderKey(provider)] = true
	}

	for _, localProviders := range localArgs {
		dynamicArgKey := genProviderKey(localProviders.Interface().(Provider))

		if !actualLocalProviderMap[dynamicArgKey] && globalProviders[dynamicArgKey] == nil {
			panic(genError(dynamicModuleType, dynamicArgKey, localArgsIndex[dynamicArgKey]))
		}
	}

	return staticModule
}

func injectDependencies(component any, kind string, dependencies map[string]Provider) reflect.Value {
	componentType := reflect.TypeOf(component)
	componentValue := reflect.ValueOf(component)
	newComponent := reflect.New(componentType)

	// injected providers into components
	// can be injected through global modules
	// or through imported modules
	for j := 0; j < componentType.NumField(); j++ {
		componentField := componentType.Field(j)
		componentFieldType := componentField.Type
		componentFieldKey := genFieldKey(componentFieldType)
		componentFieldName := componentField.Name

		if !token.IsExported(componentFieldName) {
			panic(fmt.Errorf(
				utils.FmtRed(
					"can't set value to unexported '%v' field of the %v %v",
					componentFieldName,
					componentType.Name(),
					kind,
				),
			))
		}

		// inject provider priorities
		// local inject
		// global inject
		// inner packages
		// resolve dependencies error
		if componentFieldKey != "" && dependencies[componentFieldKey] != nil {
			newComponent.Elem().Field(j).Set(reflect.ValueOf(dependencies[componentFieldKey]))
		} else if componentFieldKey != "" && globalProviders[componentFieldKey] != nil {
			newComponent.Elem().Field(j).Set(reflect.ValueOf(globalProviders[componentFieldKey]))
		} else if !isInjectedProvider(componentFieldType) {

			// if module set state to provider
			// this line will set state again to provider
			// other wise state = nil
			newComponent.Elem().Field(j).Set(componentValue.Field(j))
		} else {
			panic(fmt.Errorf(
				utils.FmtRed(
					"can't resolve dependencies of the '%v' %v. Please make sure that the argument dependency at index [%v] is available in the '%v' %v",
					componentFieldType.String(),
					kind,
					j,
					componentType.Name(),
					kind,
				),
			))
		}
	}

	return newComponent
}
