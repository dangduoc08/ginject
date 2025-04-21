package core

type RESTLayer struct {
	handler         any
	controllerPath  string
	name            string
	route           string
	version         string
	method          string
	pattern         string
	mainHandlerName string
}

func generateLayersByPattern(layers []RESTLayer) map[string][]*RESTLayer {
	result := map[string][]*RESTLayer{}

	for _, layer := range layers {
		if _, ok := result[layer.pattern]; !ok {
			result[layer.pattern] = []*RESTLayer{}
		}

		result[layer.pattern] = append(
			result[layer.pattern],
			&layer,
		)
	}

	return result
}
