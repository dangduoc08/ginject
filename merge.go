package gogo

// Merge helps group all routers together
func Merge(parentRoute string, r *R) *R {
	parentRoute = handleSlash(parentRoute)
	router := *r

	for httpMethod := range router {
		var routeAndHandlerArray []map[string]interface{} = router[httpMethod]
		if len(routeAndHandlerArray) > 0 {
			for _, routeAndHandlerMap := range routeAndHandlerArray {
				routeAndHandlerMap[routeKey] = parentRoute + routeAndHandlerMap[routeKey].(string)
			}
		}
	}

	return r
}
