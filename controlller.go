package gogo

// Handler handle request and response
// with third param is a next function,
// we can use as a middleware function
// by pass many handler arguments
// and invoke next function
type Handler = func(req *Request, res ResponseExtender, next func())

// Controller defines application and router abstract layers
// Get, Post, Put, Delete, Patch, Head, Options
// are the same, only diff http method
// UseRouter to merge router with routers or with application
// UseMiddleware
type Controller interface {
	Get(route string, handlers ...Handler) Controller
	Post(route string, handlers ...Handler) Controller
	Put(route string, handlers ...Handler) Controller
	Delete(route string, handlers ...Handler) Controller
	Patch(route string, handlers ...Handler) Controller
	Head(route string, handlers ...Handler) Controller
	Options(route string, handlers ...Handler) Controller
	Group(args ...interface{}) Controller
	Use(args ...interface{}) Controller
}
