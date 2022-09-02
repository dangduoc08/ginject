package routing

import (
	"net/http"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/ds"
)

type Routable interface {
	Get(route string, handlers ...context.Handler) Routable
	Head(route string, handlers ...context.Handler) Routable
	Post(route string, handlers ...context.Handler) Routable
	Put(route string, handlers ...context.Handler) Routable
	Patch(route string, handlers ...context.Handler) Routable
	Delete(route string, handlers ...context.Handler) Routable
	Connect(route string, handlers ...context.Handler) Routable
	Options(route string, handlers ...context.Handler) Routable
	Trace(route string, handlers ...context.Handler) Routable
	Group(prefixRoute string, subRouters ...*Router) Routable
	Use(handlers ...context.Handler) Routable
	For(route string) func(handlers ...context.Handler) Routable
}

func (routerInstance *Router) Get(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodGet, route), handlers...)
}

func (routerInstance *Router) Head(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodHead, route), handlers...)
}

func (routerInstance *Router) Post(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodPost, route), handlers...)
}

func (routerInstance *Router) Put(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodPut, route), handlers...)
}

func (routerInstance *Router) Patch(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodPatch, route), handlers...)
}

func (routerInstance *Router) Delete(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodDelete, route), handlers...)
}

func (routerInstance *Router) Connect(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodConnect, route), handlers...)
}

func (routerInstance *Router) Options(route string, handlers ...context.Handler) Routable {
	return routerInstance.add(handleMethodWithRoute(http.MethodOptions, route), handlers...)
}

func (routerInstance *Router) Trace(route string, handlers ...context.Handler) Routable {
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
		for _, subRouterDataMappedByRoute := range subRouter.array {
			for subRoute, subRouterData := range subRouterDataMappedByRoute {
				method := matchMethodReg.FindString(subRoute)
				subRouteWithoutMethod := matchMethodReg.ReplaceAllString(subRoute, "")
				routerAdapter.insert(method+prefixRoute+subRouteWithoutMethod, *subRouterData.Handlers...)
			}
		}
	}

	return routerInstance
}

func (routerInstance *Router) Use(handlers ...context.Handler) Routable {
	routerInstance.middlewares.cache(ds.WILDCARD, handlers...)
	routerAdapter := adapter{
		routerInstance,
	}
	routerAdapter.serve(ds.WILDCARD, USE, handlers...)

	return routerInstance
}

func (routerInstance *Router) For(route string) func(handlers ...context.Handler) Routable {
	routerAdapter := adapter{
		routerInstance,
	}

	return func(handlers ...context.Handler) Routable {
		for _, method := range HTTP_METHODS {
			routeWithMethod := handleMethodWithRoute(method, route)
			routerInstance.middlewares.cache(handlePath(routeWithMethod), handlers...)
			routerAdapter.serve(routeWithMethod, USE, handlers...)
		}

		return routerInstance
	}
}
