package gooh

// type application struct {
// 	router *routing.Router
// }

// func Default() *application {
// 	appInstance := application{
// 		routing.NewRouter(),
// 	}

// 	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
// 		isMatched, matchedRoute, routerData := appInstance.match(req.Method, req.URL.Path)
// 		if isMatched {
// 			fmt.Println(matchedRoute)
// 			fmt.Println(routerData.Params)
// 		} else {
// 			http.NotFound(res, req)
// 		}
// 	})

// 	return &appInstance
// }

// func (appInstance *application) ListenAndServe(addr string, handler http.Handler) error {
// 	return http.ListenAndServe(addr, handler)
// }

// func (appInstance *application) add(method, route string, handlers ...context.Handler) IController {
// 	handledRoute := handleRoute(method, route)
// 	appInstance.router.Add(handledRoute, handlers...)
// 	return appInstance
// }

// func (appInstance *application) Get(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodGet, route, handlers...)
// }

// func (appInstance *application) Head(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodHead, route, handlers...)
// }

// func (appInstance *application) Post(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodPost, route, handlers...)
// }

// func (appInstance *application) Put(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodPut, route, handlers...)
// }

// func (appInstance *application) Patch(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodPatch, route, handlers...)
// }

// func (appInstance *application) Delete(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodDelete, route, handlers...)
// }

// func (appInstance *application) Connect(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodConnect, route, handlers...)
// }

// func (appInstance *application) Options(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodOptions, route, handlers...)
// }

// func (appInstance *application) Trace(route string, handlers ...context.Handler) IController {
// 	return appInstance.add(http.MethodTrace, route, handlers...)
// }

// func (appInstance *application) Group(prefixRoute string, subHttpRouters ...*httpRouter) IController {
// 	if prefixRoute == "" {
// 		prefixRoute = ds.SLASH
// 	}

// 	for _, httpRouter := range subHttpRouters {
// 		appInstance.router.Group(addSlash(prefixRoute), httpRouter.router)
// 	}

// 	return appInstance
// }

// func (appInstance *application) Use(handlers ...context.Handler) IController {
// 	appInstance.router.Use(handlers...)
// 	return appInstance
// }

// func (appInstance *application) For(route string) func(handlers ...context.Handler) IController {
// 	return func(handlers ...context.Handler) IController {
// 		for _, METHOD := range HTTP_METHODS {
// 			appInstance.router.For(handleRoute(METHOD, route))(handlers...)
// 		}
// 		return appInstance
// 	}
// }

// func (appInstance *application) match(method, route string) (bool, string, *routing.RouterData) {
// 	return appInstance.router.Match(handleRoute(method, route))
// }
