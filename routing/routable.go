package routing

import (
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/ds"
)

type Routable interface {
	Get(route string, handlers ...ctx.Handler) Routable
	Head(route string, handlers ...ctx.Handler) Routable
	Post(route string, handlers ...ctx.Handler) Routable
	Put(route string, handlers ...ctx.Handler) Routable
	Patch(route string, handlers ...ctx.Handler) Routable
	Delete(route string, handlers ...ctx.Handler) Routable
	Connect(route string, handlers ...ctx.Handler) Routable
	Options(route string, handlers ...ctx.Handler) Routable
	Trace(route string, handlers ...ctx.Handler) Routable
	Group(prefixRoute string, subRouters ...*Router) Routable
	Use(handlers ...ctx.Handler) Routable
	For(route string) func(handlers ...ctx.Handler) Routable
}

func (routerInstance *Router) Get(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodGet, route), handlers...)
}

func (routerInstance *Router) Head(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodHead, route), handlers...)
}

func (routerInstance *Router) Post(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodPost, route), handlers...)
}

func (routerInstance *Router) Put(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodPut, route), handlers...)
}

func (routerInstance *Router) Patch(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodPatch, route), handlers...)
}

func (routerInstance *Router) Delete(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodDelete, route), handlers...)
}

func (routerInstance *Router) Connect(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodConnect, route), handlers...)
}

func (routerInstance *Router) Options(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodOptions, route), handlers...)
}

func (routerInstance *Router) Trace(route string, handlers ...ctx.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodTrace, route), handlers...)
}

func (routerInstance *Router) Group(prefixRoute string, subRouters ...*Router) Routable {
	if prefixRoute == "" {
		prefixRoute = ds.SLASH
	}

	// prevent add prefix include slash at last
	prefixRoute = ds.RemoveAtEnd(prefixRoute, ds.SLASH)
	for _, subRouter := range subRouters {
		routerAdapter := adapter{
			routerInstance,
		}

		// Add sub middlewares to main router
		// incase sub routers has no handlers
		if len(subRouter.RouteMapDataArr) <= 0 {
			for _, subMiddlewaresMap := range subRouter.middlewares {
				for subRoute, subMiddlewares := range subMiddlewaresMap {
					if subRoute == ds.WILDCARD {
						routerInstance.middlewares.cache(subRoute, *subMiddlewares...)
					} else {
						method := matchMethodReg.FindString(subRoute)
						subRouteWithoutMethod := matchMethodReg.ReplaceAllString(subRoute, "")
						routerInstance.middlewares.cache(method+prefixRoute+subRouteWithoutMethod, *subMiddlewares...)
					}
				}
			}
		}

		for _, subRouterDataMappedByRoute := range subRouter.RouteMapDataArr {
			for subRoute, subRouterData := range subRouterDataMappedByRoute {
				method := matchMethodReg.FindString(subRoute)
				subRouteWithoutMethod := matchMethodReg.ReplaceAllString(subRoute, "")
				routerAdapter.insert(method+prefixRoute+subRouteWithoutMethod, *subRouterData.Handlers...)
				routerAdapter.serve(method+prefixRoute+subRouteWithoutMethod, ADD)
			}
		}
	}

	return routerInstance
}

func (routerInstance *Router) Use(handlers ...ctx.Handler) Routable {
	routerInstance.middlewares.cache(ds.WILDCARD, handlers...)
	routerAdapter := adapter{
		routerInstance,
	}
	routerAdapter.serve(ds.WILDCARD, USE, handlers...)

	return routerInstance
}

func (routerInstance *Router) For(route string) func(handlers ...ctx.Handler) Routable {
	routerAdapter := adapter{
		routerInstance,
	}

	return func(handlers ...ctx.Handler) Routable {
		for _, method := range HTTP_METHODS {
			routeWithMethod := handleMethodWithRoute(method, route)
			routerInstance.middlewares.cache(handlePath(routeWithMethod), handlers...)
			routerAdapter.serve(routeWithMethod, USE, handlers...)
		}

		return routerInstance
	}
}

func (routerInstance *Router) Match(method, route string) (bool, string, *routerData) {
	routeWithMethod := handleMethodWithRoute(method, route)
	return routerInstance.match(routeWithMethod)
}
