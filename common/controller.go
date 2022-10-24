package common

import (
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
)

type Controller interface {
	NewController() Controller
}

type Control struct {
	prefixes []string
	routers  map[string][]ctx.Handler
}

func (c *Control) Prefix(prefix string) *Control {
	c.prefixes = append(c.prefixes, prefix)
	return c
}

func (c *Control) Get(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodGet, handlers...)
	return c
}

func (c *Control) Head(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodHead, handlers...)
	return c
}

func (c *Control) Post(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodPost, handlers...)
	return c
}

func (c *Control) Put(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodPut, handlers...)
	return c
}

func (c *Control) Patch(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodPatch, handlers...)
	return c
}

func (c *Control) Delete(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodDelete, handlers...)
	return c
}

func (c *Control) Connect(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodDelete, handlers...)
	return c
}

func (c *Control) Options(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodOptions, handlers...)
	return c
}

func (c *Control) Trace(path string, handlers ...ctx.Handler) *Control {
	c.addToRouters(path, http.MethodTrace, handlers...)
	return c
}
