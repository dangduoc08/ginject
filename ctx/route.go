package ctx

type Layer struct {
	Handler *Handler
}

type Route struct {
	Path string
	// Stack []*Layer
}
