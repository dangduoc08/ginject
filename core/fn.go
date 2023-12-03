package core

import (
	"encoding/json"
	"fmt"
	"go/token"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/context"
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
			newArg, err := injectDependencies(bodyPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(BODY_PIPEABLE, i, newArg)
		} else if formPipeable, isImplFormPipeable := argAnyValue.(common.FormPipeable); isImplFormPipeable {
			newArg, err := injectDependencies(formPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(FORM_PIPEABLE, i, newArg)
		} else if queryPipeable, isImplQueryPipeable := argAnyValue.(common.QueryPipeable); isImplQueryPipeable {
			newArg, err := injectDependencies(queryPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(QUERY_PIPEABLE, i, newArg)
		} else if headerPipeable, isImplHeaderPipeable := argAnyValue.(common.HeaderPipeable); isImplHeaderPipeable {
			newArg, err := injectDependencies(headerPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(HEADER_PIPEABLE, i, newArg)
		} else if paramPipeable, isImplParamPipeable := argAnyValue.(common.ParamPipeable); isImplParamPipeable {
			newArg, err := injectDependencies(paramPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(PARAM_PIPEABLE, i, newArg)
		} else if wsPayloadPipeable, isImplWSPayloadPipeable := argAnyValue.(common.WSPayloadPipeable); isImplWSPayloadPipeable {
			newArg, err := injectDependencies(wsPayloadPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(WS_PAYLOAD_PIPEABLE, i, newArg)
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
				"can't resolve dependencies of the '%v'. Please make sure that the argument dependency at index [%v] is available in the handler",
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

func createStaticModuleFromDynamicModule(dynamicModule any) *Module {
	dynamicModuleType := reflect.TypeOf(dynamicModule)
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
		if globalProviders[dynamicArgKey] != nil {

			// if an injected provider doesn't exist, check if a global provider with the same type exists
			// if a global provider exists, append it to the list of arguments
			globalArgs = append(globalArgs, reflect.ValueOf(globalProviders[dynamicArgKey]))
		} else if globalInterfaces[dynamicArgKey] != nil {
			globalArgs = append(globalArgs, reflect.ValueOf(globalInterfaces[dynamicArgKey]))
		} else {
			panic(genError(dynamicModuleType, dynamicArgKey, i))
		}
	})

	args = append(args, append(localArgs, globalArgs...)...)

	// call the dynamic module with the list of arguments and convert the result to a static module
	staticModule := reflect.ValueOf(dynamicModule).Call(args)[0].Interface().(*Module)

	return staticModule
}

func injectDependencies(component any, kind string, dependencies map[string]Provider) (reflect.Value, error) {
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
		} else if globalInterfaces[componentFieldKey] != nil {
			newComponent.Elem().Field(j).Set(reflect.ValueOf(globalInterfaces[componentFieldKey]))
		} else if !isInjectedProvider(componentFieldType) {

			// if module set state to provider
			// this line will set state again to provider
			// other wise state = nil
			newComponent.Elem().Field(j).Set(componentValue.Field(j))
		} else {
			return reflect.ValueOf(nil), fmt.Errorf(
				utils.FmtRed(
					"can't resolve dependencies of the '%v' %v. Please make sure that the argument dependency at index [%v] is available in the '%v' %v",
					componentFieldType.String(),
					kind,
					j,
					componentType.Name(),
					kind,
				),
			)
		}
	}

	return newComponent, nil
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func logBoostrap(port int) {
	accessURLs := utils.FmtBold(utils.FmtBGYellow(utils.FmtWhite(" GOOH! Here Are Your Access URLs: "))) + "\n"
	divider := utils.FmtDim("--------------------------------------------") + "\n"
	host := utils.FmtBold(utils.FmtWhite("Localhost: ")) + utils.FmtMagenta("%v:%v", "localhost", port) + "\n"
	lan := utils.FmtBold(utils.FmtWhite("      LAN: ")) + utils.FmtMagenta(fmt.Sprintf("%v:%v", getLocalIP(), port)) + "\n"
	close := utils.FmtItalic(utils.FmtGreen("Press CTRL+C to stop")) + "\n\n"

	os.Stdout.Write([]byte(accessURLs))
	os.Stdout.Write([]byte(divider))
	os.Stdout.Write([]byte(host))
	os.Stdout.Write([]byte(lan))
	os.Stdout.Write([]byte(divider))
	os.Stdout.Write([]byte(close))
}

func getDependency(k string, c *context.Context, pipeValue reflect.Value) any {
	switch k {
	case CONTEXT:
		return c
	case WS_CONNECTION:
		return c.WS.Connection
	case REQUEST:
		return c.Request
	case RESPONSE:
		return c.ResponseWriter
	case BODY:
		return c.Body()
	case FORM:
		return c.Form()
	case QUERY:
		return c.Query()
	case HEADER:
		return c.Header()
	case PARAM:
		return c.Param()
	case WS_PAYLOAD:
		return c.WS.Message.Payload
	case NEXT:
		return c.Next
	case REDIRECT:
		return c.Redirect
	case BODY_PIPEABLE:
		return pipeValue.
			Interface().(common.BodyPipeable).
			Transform(c.Body(), common.ArgumentMetadata{
				ParamType: BODY_PIPEABLE,
			})
	case FORM_PIPEABLE:
		return pipeValue.
			Interface().(common.FormPipeable).
			Transform(c.Form(), common.ArgumentMetadata{
				ParamType: FORM_PIPEABLE,
			})
	case QUERY_PIPEABLE:
		return pipeValue.
			Interface().(common.QueryPipeable).
			Transform(c.Query(), common.ArgumentMetadata{
				ParamType: QUERY_PIPEABLE,
			})
	case HEADER_PIPEABLE:
		return pipeValue.
			Interface().(common.HeaderPipeable).
			Transform(c.Header(), common.ArgumentMetadata{
				ParamType: HEADER_PIPEABLE,
			})
	case PARAM_PIPEABLE:
		return pipeValue.
			Interface().(common.ParamPipeable).
			Transform(c.Param(), common.ArgumentMetadata{
				ParamType: PARAM_PIPEABLE,
			})
	case WS_PAYLOAD_PIPEABLE:
		return pipeValue.
			Interface().(common.WSPayloadPipeable).
			Transform(c.WS.Message.Payload, common.ArgumentMetadata{
				ParamType: WS_PAYLOAD_PIPEABLE,
			})
	}

	return dependencies
}

func returnREST(c *context.Context, data reflect.Value) {
	switch data.Type().Kind() {
	case
		reflect.Map,
		reflect.Slice,
		reflect.Struct,
		reflect.Interface:
		c.JSON(data.Interface())
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		c.Text(fmt.Sprint(data))
	case
		reflect.Pointer,
		reflect.UnsafePointer:
		c.Text(fmt.Sprint(data.UnsafePointer()))
	case
		reflect.String:
		c.Text(data.Interface().(string))
	case
		reflect.Func:
		c.Text(data.Type().String())
	}
}

func toWSMessage(data reflect.Value) string {
	switch data.Type().Kind() {
	case
		reflect.Map,
		reflect.Slice,
		reflect.Struct,
		reflect.Interface:
		jsonBuf, _ := json.Marshal(data.Interface())
		return string(jsonBuf)
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		return fmt.Sprint(data)
	case
		reflect.Pointer,
		reflect.UnsafePointer:
		return fmt.Sprint(data.UnsafePointer())
	case
		reflect.String:
		return data.Interface().(string)
	case
		reflect.Func:
		return data.Type().String()
	default:
		return data.String()
	}
}

func setStatusCode(c *context.Context, statusCode reflect.Value) {
	statusCodeKind := statusCode.Type().Kind()

	if statusCodeKind == reflect.Int {
		status := int(statusCode.Int())
		if http.StatusText(status) != "" {
			c.Status(status)
		}
	} else if statusCodeKind == reflect.Interface {
		if status, ok := statusCode.Interface().(int); ok &&
			http.StatusText(status) != "" {
			c.Status(status)
		}
	}
}
