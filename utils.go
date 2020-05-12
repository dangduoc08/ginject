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

// Common UserRouter function for app and router
func useRouter(args ...interface{}) (string, []*router) {
	var totalArg int = len(args)

	if totalArg == 0 {
		panic("UseRouter must pass arguments")
	}

	var parentRoute string
	var sourceRouters []*router

	for index, arg := range args {
		var isFirstArg bool = index == 0

		switch arg.(type) {
		case string:
			if isFirstArg {
				if totalArg <= 1 {
					panic("UseRouter need atleast a router")
				}
				parentRoute = formatRoute(arg.(string))
			} else {
				panic("UseRouter only accepts string as first argument")
			}
			break

		case *routerGroup:

			// Push all source router to an router array
			// prepare to merge
			var sourceRouter *router = &(arg.(*routerGroup).router)
			sourceRouters = append(sourceRouters, sourceRouter)
			break
		}
	}

	return parentRoute, sourceRouters
}
