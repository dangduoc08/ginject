package context

import "net/http"

type Header = http.Header

func (c *Context) Header() Header {
	return c.Request.Header
}

func (c *Context) SetHeaders(pair map[string]string) Responser {
	for key, value := range pair {
		c.ResponseWriter.Header().Set(key, value)
	}
	return c
}
