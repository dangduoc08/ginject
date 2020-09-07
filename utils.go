package gogo

// Format route types to "/<route_wildcard>"
// no need to care about trailing slashes
func handleSlash(route string) string {
	if route != empty {
		var lastIndex int = len(route) - 1

		// Remove "/" at last route
		if route != slash && string(route[lastIndex]) == slash {
			route = route[0:lastIndex]
		}

		// Add "/" at first route
		if string(route[0]) != slash {
			route = slash + route
		}
	}
	return route
}

// Common UseRouter function for app and router
// resolve arguments to URL path and source router group
func resolveRouterGroup(args ...interface{}) (string, []*routerGroup) {
	var totalArg int = len(args)

	if totalArg == 0 {
		panic("UseRouter must pass arguments")
	}

	var parentRoute string
	var sourceRouterGroups []*routerGroup

	for index, arg := range args {
		var isFirstArg bool = index == 0

		switch arg.(type) {
		case string:
			if isFirstArg {
				if totalArg <= 1 {
					panic("UseRouter need atleast a router")
				}
				parentRoute = arg.(string)
			} else {
				panic("UseRouter only accepts string as first argument")
			}
			break

		case *routerGroup:

			// Push all source router group
			// to an router group array
			// prepare to merge
			var sourceRouterGroup *routerGroup = arg.(*routerGroup)
			sourceRouterGroups = append(sourceRouterGroups, sourceRouterGroup)
			break
		}
	}

	return parentRoute, sourceRouterGroups
}

// Common UseMiddleware function for app and router
// resolve arguments to URL path and source handlers
func resolveMiddlewares(args ...interface{}) (string, []Handler) {
	var totalArg int = len(args)

	if totalArg == 0 {
		panic("UseMiddleware must pass arguments")
	}

	var parentRoute string
	var sourceHandlers []Handler

	for index, arg := range args {
		var isFirstArg bool = index == 0

		switch arg.(type) {
		case string:
			if isFirstArg {
				if totalArg <= 1 {
					panic("UseMiddleware need atleast a handler")
				}
				parentRoute = arg.(string)
			} else {
				panic("UseMiddleware only accepts string as first argument")
			}
			break

		case Handler:
			var sourceHandler Handler = arg.(Handler)
			sourceHandlers = append(sourceHandlers, sourceHandler)
			break
		}
	}

	return parentRoute, sourceHandlers
}

// Merge source router groups into target
// can be router group
// or can be app
func mergeRouterGroup(target interface{}, parentRoute string, sourceRouterGroups []*routerGroup) {
	var targetMiddlewares *[]Handler
	var insert func(route, httpMethod string, handlers ...Handler)

	switch target.(type) {
	case *app:
		gg := target.(*app)
		targetMiddlewares = &gg.middlewares
		insert = gg.routerTree.insert
		break
	case *routerGroup:
		gr := target.(*routerGroup)
		targetMiddlewares = &gr.middlewares
		insert = gr.insert
		break
	}

	for _, sourceRouterGroup := range sourceRouterGroups {
		var sourceRouter router = sourceRouterGroup.router              // router of source router group
		var sourceMiddlewares []Handler = sourceRouterGroup.middlewares // global source middlewares

		// Push each source middleware
		// into global target router group middlewares
		if len(sourceMiddlewares) > 0 {
			*targetMiddlewares = append(*targetMiddlewares, sourceMiddlewares...)
		}

		for httpMethod, sourceRouteAndHandlerMapSlice := range sourceRouter {
			for _, sourceRouteAndHandlerMap := range sourceRouteAndHandlerMapSlice {
				var sourceRoute string = sourceRouteAndHandlerMap[routeKey].(string)
				var mergedRoute string = parentRoute

				// To make sure remove slash at the last route
				if sourceRoute != slash {
					mergedRoute += sourceRoute
				}
				var handlers []Handler = sourceRouteAndHandlerMap[handlersKey].([]Handler)

				// Append group route to existed group
				// or create the new once
				// this will be handled inside insert function
				insert(mergedRoute, httpMethod, handlers...)
			}
		}
	}
}

// Merge source middlewares to target
// there are 2 cases when merge
// if no route, will merge to global middleware
// else will merge to matched route
func mergeMiddleware(target interface{}, parentRoute string, sourceHandlers []Handler) {
	var targetMiddlewares *[]Handler
	var insert func(route, httpMethod string, handlers ...Handler)

	switch target.(type) {
	case *app:
		gg := target.(*app)
		targetMiddlewares = &gg.middlewares
		insert = gg.routerTree.insert
		break
	case *routerGroup:
		gr := target.(*routerGroup)
		targetMiddlewares = &gr.middlewares
		insert = gr.insert
		break
	}

	if parentRoute != empty {

		// Append handler to existing route controller
		// or create the new once
		// this will be handled inside insert function
		for _, httpMethod := range httpMethods {
			insert(parentRoute, httpMethod, sourceHandlers...)
		}
	} else {

		// Append source middlewares into global middlewares
		*targetMiddlewares = append(*targetMiddlewares, sourceHandlers...)
	}
}
