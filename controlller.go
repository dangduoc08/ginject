package gogo

// Handler handle request and response
// with third param is a next function,
// we can use as a middleware function
// by pass many handler arguments
// and invoke next function
type Handler func(req *Request, res ResponseExtender, next func())

// Controller defines application and router abstract layers
// GET, POST, PUT, DELETE are the same, only diff http method
// UseRouter to merge router with routers or with application
// UseMiddleware
type Controller interface {
	GET(route string, handlers ...Handler) Controller
	POST(route string, handlers ...Handler) Controller
	PUT(route string, handlers ...Handler) Controller
	PATCH(route string, handlers ...Handler) Controller
	HEAD(route string, handlers ...Handler) Controller
	OPTIONS(route string, handlers ...Handler) Controller
	DELETE(route string, handlers ...Handler) Controller
	UseRouter(args ...interface{}) Controller
	UseMiddleware(args ...interface{}) Controller
}
