package gogo

// Merge source router groups into target router group
func mergeRouterWithRouter(
	parentRoute string,
	targetRouterGroup *routerGroup,
	sourceRouterGroups []*routerGroup,
) {
	for _, sourceRouterGroup := range sourceRouterGroups {
		var sourceMiddlewares []Handler = sourceRouterGroup.middlewares
		var sourceRouter router = sourceRouterGroup.router

		// Push each source middleware
		// into target router group middlewares
		targetRouterGroup.middlewares = append(targetRouterGroup.middlewares, sourceMiddlewares...)

		// Iterable source router map
		sourceRouter.forEach(func(httpMethod string, routeAndHandlerMap map[string]interface{}) {

			// Clone router and handler map
			// from source router
			// to avoid effect on source router
			routeAndHandlerMapClone := make(map[string]interface{})
			routeAndHandlerMapClone[routeKey] = parentRoute + routeAndHandlerMap[routeKey].(string)
			routeAndHandlerMapClone[handlersKey] = routeAndHandlerMap[handlersKey]

			targetRouterGroup.router[httpMethod] = append(targetRouterGroup.router[httpMethod], routeAndHandlerMapClone)
		})
	}
}

// Merge source router groups routers into application
func mergeRouterWithApp(
	parentRoute string,
	gg *app,
	sourceRouterGroups []*routerGroup,
) {
	for _, sourceRouterGroup := range sourceRouterGroups {
		var sourceMiddlewares []Handler = sourceRouterGroup.middlewares
		var sourceRouter router = sourceRouterGroup.router

		// Push each source middleware
		// into global application middlewares
		gg.middlewares = append(gg.middlewares, sourceMiddlewares...)

		// Iterable source router map
		sourceRouter.forEach(func(httpMethod string, routeAndHandlerMap map[string]interface{}) {
			var route string = parentRoute + routeAndHandlerMap[routeKey].(string)

			// Format route before insert trie
			route = httpMethod + formatRoute(route)
			var handlers []Handler = routeAndHandlerMap[handlersKey].([]Handler)

			// Insert each route and handlers to trie
			gg.routerTree.insert(route, httpMethod, handlers...)
		})
	}
}
