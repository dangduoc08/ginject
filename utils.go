package gogo

// Format route types to "/<route_wildcard>"
// no need to care about trailing slashes
func formatRoute(route string) string {
	var lastIndex int = len(route) - 1

	// Remove "/" at last route
	if route != slash && string(route[lastIndex]) == slash {
		route = route[0:lastIndex]
	}

	// Add "/" at first route
	if string(route[0]) != slash {
		route = slash + route
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
