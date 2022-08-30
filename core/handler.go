package core

import (
	"github.com/dangduoc08/gooh/context"
)

type Handler = func(ctx *context.Context)

type IController interface {
	Get(route string, handlers ...Handler) IController
	Post(route string, handlers ...Handler) IController
	Put(route string, handlers ...Handler) IController
	Delete(route string, handlers ...Handler) IController
	Patch(route string, handlers ...Handler) IController
	Head(route string, handlers ...Handler) IController
	Options(route string, handlers ...Handler) IController
	Group(args ...interface{}) IController
	Use(args ...interface{}) IController
}
