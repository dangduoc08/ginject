package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/dangduoc08/ginject/broker"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/utils"
)

var pkgFromControllerKeyReg = regexp.MustCompile(`\[.*?\]`)

func isDynamicModule(moduleType string) bool {
	return strings.HasPrefix(moduleType, "func(") && strings.HasSuffix(moduleType, "*core.Module")
}

func getFnArgs(f any, injectedProviders map[string]Provider, cb func(string, int, reflect.Value)) {
	getFnArgsByType(reflect.TypeOf(f), injectedProviders, cb)
}

func getFnArgsByType(injectableFnType reflect.Type, injectedProviders map[string]Provider, cb func(string, int, reflect.Value)) {
	for i := 0; i < injectableFnType.NumIn(); i++ {
		argType := injectableFnType.In(i)
		arg := argType.PkgPath() + "/" + argType.String()
		newArg := reflect.New(argType).Elem()
		argAnyValue := newArg.Interface()

		if contextPipeable, isImplContextPipeable := argAnyValue.(common.ContextPipeable); isImplContextPipeable {
			newArg, err := injectDependencies(contextPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.CONTEXT_PIPEABLE, i, newArg)
		} else if bodyPipeable, isImplBodyPipeable := argAnyValue.(common.BodyPipeable); isImplBodyPipeable {
			newArg, err := injectDependencies(bodyPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.BODY_PIPEABLE, i, newArg)
		} else if formPipeable, isImplFormPipeable := argAnyValue.(common.FormPipeable); isImplFormPipeable {
			newArg, err := injectDependencies(formPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.FORM_PIPEABLE, i, newArg)
		} else if queryPipeable, isImplQueryPipeable := argAnyValue.(common.QueryPipeable); isImplQueryPipeable {
			newArg, err := injectDependencies(queryPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.QUERY_PIPEABLE, i, newArg)
		} else if headerPipeable, isImplHeaderPipeable := argAnyValue.(common.HeaderPipeable); isImplHeaderPipeable {
			newArg, err := injectDependencies(headerPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.HEADER_PIPEABLE, i, newArg)
		} else if paramPipeable, isImplParamPipeable := argAnyValue.(common.ParamPipeable); isImplParamPipeable {
			newArg, err := injectDependencies(paramPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.PARAM_PIPEABLE, i, newArg)
		} else if filePipeable, isImplFilePipeable := argAnyValue.(common.FilePipeable); isImplFilePipeable {
			newArg, err := injectDependencies(filePipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.FILE_PIPEABLE, i, newArg)
		} else if wsPayloadPipeable, isImplWSPayloadPipeable := argAnyValue.(common.WSPayloadPipeable); isImplWSPayloadPipeable {
			newArg, err := injectDependencies(wsPayloadPipeable, "pipe", injectedProviders)
			if err != nil {
				panic(err)
			}

			cb(common.WS_PAYLOAD_PIPEABLE, i, newArg)
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

func genControllerKey(m *Module, c Controller) string {
	return "[" + m.ID() + "]" + genFieldKey(reflect.TypeOf(c))
}

func getPkgFromControllerKey(k string) string {
	return pkgFromControllerKeyReg.ReplaceAllString(k, "")
}

func genFieldKey(t reflect.Type) string {
	return t.PkgPath() + "/" + t.String()
}

func createStaticModuleFromDynamicModule(dynamicModule any) *Module {
	dynamicModuleType := reflect.TypeOf(dynamicModule)
	var args []reflect.Value

	genError := func(dynamicModuleType reflect.Type, dynamicArgKey string, index int) error {
		return errors.New(
			utils.FmtRed(
				"can't resolve argument of '%v'. Please make sure that the argument '%v' at index [%v] is available in the injected providers",
				strings.Replace(dynamicModuleType.String(), ") *core.Module", ")", 1),
				dynamicArgKey,
				index,
			),
		)
	}

	getFnArgs(dynamicModule, globalProviders, func(dynamicArgKey string, i int, pipeValue reflect.Value) {
		if globalProviders[dynamicArgKey] != nil {
			args = append(args, reflect.ValueOf(globalProviders[dynamicArgKey]))
		} else if globalInterfaces[dynamicArgKey] != nil {
			args = append(args, reflect.ValueOf(globalInterfaces[dynamicArgKey]))
		} else {
			panic(genError(dynamicModuleType, dynamicArgKey, i))
		}
	})

	return reflect.ValueOf(dynamicModule).Call(args)[0].Interface().(*Module)
}

func injectDependencies(component any, kind string, dependencies map[string]Provider) (reflect.Value, error) {
	componentType := reflect.TypeOf(component)
	componentValue := reflect.ValueOf(component)
	newComponent := reflect.New(componentType)

	// injected providers into components
	// can be injected through global modules
	// or through imported modules
	componentName := path.Base(componentType.PkgPath()) + "." + componentType.Name()
	for j := 0; j < componentType.NumField(); j++ {
		componentField := componentType.Field(j)
		componentFieldType := componentField.Type
		componentFieldKey := genFieldKey(componentFieldType)
		componentFieldName := componentField.Name

		if !token.IsExported(componentFieldName) {
			panic(errors.New(
				utils.FmtRed(
					"can't set value to unexported '%v' field of the %v %v",
					componentFieldName,
					componentName,
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
		} else if componentFieldKey != "" && globalInterfaces[componentFieldKey] != nil {
			newComponent.Elem().Field(j).Set(reflect.ValueOf(globalInterfaces[componentFieldKey]))
		} else if !isInjectedProvider(componentFieldType) {

			// if module set state to provider
			// this line will set state again to provider
			// other wise state = nil
			newComponent.Elem().Field(j).Set(componentValue.Field(j))
		} else {
			return reflect.ValueOf(nil), errors.New(
				utils.FmtRed(
					"can't resolve dependency '%v' of the %v. Please make sure that the argument dependency at index [%v] is available in the '%v' %v",
					componentFieldType.String(),
					kind,
					j,
					componentName,
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
	accessURLs := utils.FmtBold("%s", utils.FmtBGYellow(" GG! Here Are Your Access URLs: ")) + "\n"
	divider := utils.FmtDim("--------------------------------------------") + "\n"
	host := utils.FmtBold("%s", utils.FmtWhite("Localhost: ")) + utils.FmtMagenta("%v:%v", "localhost", port) + "\n"
	lan := utils.FmtBold("%s", utils.FmtWhite("      LAN: ")) + utils.FmtMagenta("%v:%v", getLocalIP(), port) + "\n"
	close := utils.FmtItalic("%s", utils.FmtGreen("Press CTRL+C to stop")) + "\n"

	_, _ = fmt.Fprint(os.Stdout, "\n"+accessURLs+divider+host+lan+divider+close)
}

func getDependency(k string, c *ctx.Context, pipeValue reflect.Value) any {
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
	case FILE:
		return c.File()
	case WS_PAYLOAD:
		return c.WS.Message.Payload
	case NEXT:
		return c.Next
	case REDIRECT:
		return c.Redirect
	case common.CONTEXT_PIPEABLE:
		return pipeValue.
			Interface().(common.ContextPipeable).
			Transform(c, common.ArgumentMetadata{
				ParamType:   common.CONTEXT_PIPEABLE,
				ContextType: c.GetType(),
			})
	case common.BODY_PIPEABLE:
		return pipeValue.
			Interface().(common.BodyPipeable).
			Transform(c.Body(), common.ArgumentMetadata{
				ParamType:   common.BODY_PIPEABLE,
				ContextType: c.GetType(),
			})
	case common.FORM_PIPEABLE:
		return pipeValue.
			Interface().(common.FormPipeable).
			Transform(c.Form(), common.ArgumentMetadata{
				ParamType:   common.FORM_PIPEABLE,
				ContextType: c.GetType(),
			})
	case common.QUERY_PIPEABLE:
		return pipeValue.
			Interface().(common.QueryPipeable).
			Transform(c.Query(), common.ArgumentMetadata{
				ParamType:   common.QUERY_PIPEABLE,
				ContextType: c.GetType(),
			})
	case common.HEADER_PIPEABLE:
		return pipeValue.
			Interface().(common.HeaderPipeable).
			Transform(c.Header(), common.ArgumentMetadata{
				ParamType:   common.HEADER_PIPEABLE,
				ContextType: c.GetType(),
			})
	case common.PARAM_PIPEABLE:
		return pipeValue.
			Interface().(common.ParamPipeable).
			Transform(c.Param(), common.ArgumentMetadata{
				ParamType:   common.PARAM_PIPEABLE,
				ContextType: c.GetType(),
			})
	case common.FILE_PIPEABLE:
		return pipeValue.
			Interface().(common.FilePipeable).
			Transform(c.File(), common.ArgumentMetadata{
				ParamType:   common.FILE_PIPEABLE,
				ContextType: c.GetType(),
			})
	case common.WS_PAYLOAD_PIPEABLE:
		return pipeValue.
			Interface().(common.WSPayloadPipeable).
			Transform(c.WS.Message.Payload, common.ArgumentMetadata{
				ParamType:   common.WS_PAYLOAD_PIPEABLE,
				ContextType: c.GetType(),
			})
	}

	return dependencies
}

func returnREST(c *ctx.Context, data reflect.Value) {
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

func setStatusCode(c *ctx.Context, statusCode reflect.Value) {
	switch statusCode.Type().Kind() {
	case reflect.Int:
		status := int(statusCode.Int())
		if http.StatusText(status) != "" {
			c.Status(status)
		}
	case reflect.Interface:
		if status, ok := statusCode.Interface().(int); ok &&
			http.StatusText(status) != "" {
			c.Status(status)
		}
	}
}

func toUniqueControllers(module *Module, controllers *[]Controller) {
	seen := make(map[string]struct{}, len(*controllers))
	uniqueControllers := make([]Controller, 0, len(*controllers))
	for _, controller := range *controllers {
		controllerKey := genControllerKey(module, controller)
		if _, ok := seen[controllerKey]; !ok {
			seen[controllerKey] = struct{}{}
			uniqueControllers = append(uniqueControllers, controller)
		}
	}

	*controllers = uniqueControllers
}

func invokeHandlerByProviders(f any, injectedProviders map[string]Provider, c *ctx.Context) []reflect.Value {
	fType := reflect.TypeOf(f)
	args := make([]reflect.Value, 0, fType.NumIn())
	getFnArgsByType(fType, injectedProviders, func(dynamicArgKey string, i int, pipeValue reflect.Value) {
		if _, ok := dependencies[dynamicArgKey]; ok {
			args = append(args, reflect.ValueOf(getDependency(dynamicArgKey, c, pipeValue)))
		} else {
			panic(fmt.Errorf(
				"can't resolve dependencies of the %v. Please make sure that the argument dependency at index [%v] is available in the handler",
				fType.String(),
				i,
			))
		}
	})

	return reflect.ValueOf(f).Call(args)
}

type catchEventPayload struct {
	reqCtx    *ctx.Context
	recovered any
	index     int
}

func buildCatchMiddleware(catchEvent string, catchFns []common.Catch) ctx.Handler {
	return func(c *ctx.Context) {
		_, _ = c.Broker.Once(catchEvent, func(m *broker.Message) {
			p := m.Payload.(catchEventPayload)
			catchFnIndex := p.index

			defer func() {
				if rec := recover(); rec != nil {
					_ = c.Broker.Publish(catchEvent, catchEventPayload{reqCtx: c, recovered: rec, index: catchFnIndex + 1})
				}
			}()

			newC := p.reqCtx
			catchFn := catchFns[catchFnIndex]

			response := http.StatusText(http.StatusInternalServerError)

			switch arg := p.recovered.(type) {
			case exception.Exception:
				catchFn(newC, &arg)
				return
			case error:
				response = arg.Error()
			case string:
				response = arg
			case int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64,
				float32, float64, complex64, complex128, uintptr:
				_ = arg
			}
			ex := exception.InternalServerErrorException(response, map[string]any{
				"description": "Unknown exception",
			})
			catchFn(newC, &ex)
		})

		c.Next()
	}
}

func buildInterceptMiddleware(key string, interceptFn func(*ctx.Context, *aggregation.Aggregation) any) ctx.Handler {
	return func(c *ctx.Context) {
		aggregationInstance := aggregation.NewAggregation()

		if aggregations, ok := c.Context().Value(WithValueKey(key)).([]*aggregation.Aggregation); ok {
			aggregations = append(aggregations, aggregationInstance)
			c.Request = c.WithContext(context.WithValue(c.Context(), WithValueKey(key), aggregations))
		} else {
			c.Request = c.WithContext(context.WithValue(c.Context(), WithValueKey(key), []*aggregation.Aggregation{aggregationInstance}))
		}

		aggregationInstance.IsMainHandlerCalled = false
		aggregationInstance.SetMainData(nil)

		aggregationInstance.InterceptorData = interceptFn(c, aggregationInstance)
		setErrorAggregationOperators(c, aggregationInstance)

		c.Next()
	}
}

func buildUseMiddleware(useFn common.Use) ctx.Handler {
	return func(c *ctx.Context) { useFn(c, c.Next) }
}

func buildGuardMiddleware(canActiveFn common.CanActivate) ctx.Handler {
	return func(c *ctx.Context) { common.HandleGuard(c, canActiveFn(c)) }
}

func setErrorAggregationOperators(c *ctx.Context, aggregationInstance *aggregation.Aggregation) {
	errorOps := aggregationInstance.GetAggregationOperators(aggregation.OPERATOR_ERROR)
	if len(errorOps) == 0 {
		return
	}
	var existing []aggregation.AggregationOperator
	if v := c.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY)); v != nil {
		existing = v.([]aggregation.AggregationOperator)
	}
	merged := make([]aggregation.AggregationOperator, len(existing), len(existing)+len(errorOps))
	copy(merged, existing)
	for _, op := range errorOps {
		merged = append(merged, op.Aggregation)
	}
	c.Request = c.WithContext(context.WithValue(c.Context(), WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY), merged))
}

func getWSEventKeys() []string {
	keys := make([]string, 0, len(common.InsertedEvents))
	for k := range common.InsertedEvents {
		keys = append(keys, k)
	}
	return keys
}

func getContextID(c *ctx.Context) string {
	reqID := c.Header().Get(ctx.REQUEST_ID)
	if reqID == "" {
		uuid, _ := utils.StrUUID()
		return uuid
	}
	return reqID
}
