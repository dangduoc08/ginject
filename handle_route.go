package gogo

func handleSlash(route string) string {
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
