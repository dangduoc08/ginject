package core

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

func isDynamicModule(moduleType string) (bool, error) {
	return regexp.Match(`^func\(.*\*core.Module$`, []byte(moduleType))
}

func genProviderKey(p Provider) string {
	return reflect.TypeOf(p).String()
}

func createStaticModuleFromDynamicModule(dynamicModule any, injectedProviders map[string]Provider) *Module {
	dynamicModuleType := reflect.TypeOf(dynamicModule)
	args := []reflect.Value{}

	// loop through each input parameter of the dynamic module
	for i := 0; i < dynamicModuleType.NumIn(); i++ {

		// get the type of the current input parameter
		dynamicArgKey := dynamicModuleType.In(i).String()

		// inject provider priorities
		// local inject
		// global inject
		// inner packages

		// check if an injected provider with the same type exists
		if injectedProviders[dynamicArgKey] != nil {

			// if an injected provider exists, append it to the list of arguments
			args = append(args, reflect.ValueOf(injectedProviders[dynamicArgKey]))
		} else if globalProviders[dynamicArgKey] != nil {

			// if an injected provider doesn't exist, check if a global provider with the same type exists
			// if a global provider exists, append it to the list of arguments
			args = append(args, reflect.ValueOf(globalProviders[dynamicArgKey]))
		} else {
			panic(fmt.Errorf(
				"can't resolve argument of %v. Please make sure that the argument %v at index [%v] is available in the injected providers",
				strings.Replace(dynamicModuleType.String(), ") *core.Module", ")", 1),
				dynamicArgKey,
				i,
			))
		}
	}

	// call the dynamic module with the list of arguments and convert the result to a static module
	return reflect.ValueOf(dynamicModule).Call(args)[0].Interface().(*Module)
}
