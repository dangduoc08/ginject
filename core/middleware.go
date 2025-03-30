package core

import (
	"fmt"
	"reflect"

	"github.com/dangduoc08/gogo/common"
	"github.com/dangduoc08/gogo/ctx"
	"github.com/dangduoc08/gogo/utils"
)

type Middleware struct {
	middlewares map[string][]ctx.Handler
}

func (mw *Middleware) add(k string, middleware ctx.Handler) {
	if mw.middlewares == nil {
		mw.middlewares = make(map[string][]func(*ctx.Context))
	}

	if appliedMiddlewares, ok := mw.middlewares[k]; ok {
		mw.middlewares[k] = append(appliedMiddlewares, middleware)
	} else {
		mw.middlewares[k] = []ctx.Handler{middleware}
	}
}

func (mw *Middleware) Apply(middleware ctx.Handler, handlers ...any) *Middleware {
	if len(handlers) > 0 {
		for _, handler := range handlers {
			handlerKind := reflect.TypeOf(handler).Kind()
			if handler == nil || handlerKind != reflect.Func {
				panic(fmt.Errorf(
					utils.FmtRed(
						"%v is not a handler",
						handlerKind,
					),
				))
			}

			mw.add(common.GetFnName(handler), middleware)
		}
	} else {
		mw.add("*", middleware)
	}

	return mw
}

func (mw *Middleware) addREST(
	controllerPath string,
	restMiddlewares *[]RESTLayer,
) {
	for key, middlewareHandlers := range mw.middlewares {

		for _, middlewareHandler := range middlewareHandlers {
			middlewareName := common.GetFnName(middlewareHandler)
			middlewareLayer := RESTLayer{
				controllerPath: controllerPath,
				route:          key,
				name:           middlewareName,
				handler:        middlewareHandler,
			}

			// apply for all
			if key == "*" {
				middlewareLayer.method = key
			} else if httpMethod, _, _ := common.ParseFnNameToURL(key, common.RESTOperations); httpMethod != "" {
				middlewareLayer.method = httpMethod
			}

			*restMiddlewares = append(*restMiddlewares, middlewareLayer)
		}
	}
}

func (mw *Middleware) addWS(
	controllerName string,
	wsMiddlewares *[]struct {
		controllerName string
		Subprotocol    string
		EventName      string
		Handlers       []ctx.Handler
	},
) {
	for key, middlewareHandlers := range mw.middlewares {

		// apply for all
		if key == "*" {
			middlewareStruct := struct {
				controllerName string
				Subprotocol    string
				EventName      string
				Handlers       []ctx.Handler
			}{
				controllerName: controllerName,
				Subprotocol:    key,
				EventName:      key,
				Handlers:       middlewareHandlers,
			}
			*wsMiddlewares = append(*wsMiddlewares, middlewareStruct)
		} else if opr, eventName, _ := common.ParseFnNameToURL(key, common.WSOperations); opr != "" {
			middlewareStruct := struct {
				controllerName string
				Subprotocol    string
				EventName      string
				Handlers       []ctx.Handler
			}{
				controllerName: controllerName,
				Subprotocol:    "",
				EventName:      eventName,
				Handlers:       middlewareHandlers,
			}
			*wsMiddlewares = append(*wsMiddlewares, middlewareStruct)
		}
	}
}
