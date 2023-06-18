package core

import (
	"fmt"
	"go/token"
	"reflect"
	"regexp"
	"strings"

	"github.com/dangduoc08/gooh/utils"
)

func isDynamicModule(moduleType string) (bool, error) {
	return regexp.Match(`^func\(.*\*core.Module$`, []byte(moduleType))
}

func getFnArgs(f any, cb func(string, int)) {
	injectableFnType := reflect.TypeOf(f)
	for i := 0; i < injectableFnType.NumIn(); i++ {
		arg := injectableFnType.In(i).PkgPath() + "/" + injectableFnType.In(i).String()
		cb(arg, i)
	}
}

func isInjectableHandler(handler any) error {
	var e error

	getFnArgs(handler, func(arg string, i int) {
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
	providerType := reflect.TypeOf(p)
	return providerType.PkgPath() + "/" + providerType.String()
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

	getFnArgs(dynamicModule, func(dynamicArgKey string, i int) {
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
		componentFieldKey := componentFieldType.PkgPath() + "/" + componentFieldType.String()
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
