package gogo

// Merge source routers into target router
func mergeRouterWithRouter(
	parentRoute string,
	targetRouter *router,
	sourceRouters ...*router,
) {
	for _, sourceRouter := range sourceRouters {
		sourceRouter.forEach(func(httpMethod string, routeAndHandlerMap map[string]interface{}) {

			// Clone router and handler map
			// from source router
			// to avoid effect on source router
			routeAndHandlerMapClone := make(map[string]interface{})
			routeAndHandlerMapClone[routeKey] = parentRoute + routeAndHandlerMap[routeKey].(string)
			routeAndHandlerMapClone[handlersKey] = routeAndHandlerMap[handlersKey]

			// Push router and handler map
			// to target router
			(*targetRouter)[httpMethod] = append((*targetRouter)[httpMethod], routeAndHandlerMapClone)
		})
	}
}

// Merge source routers into application
func mergeRouterWithApp(
	parentRoute string,
	gg *app,
	sourceRouters ...*router,
) {
	for _, sourceRouter := range sourceRouters {
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
