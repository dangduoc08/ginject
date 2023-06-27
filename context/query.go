package context

import "net/url"

type Query = url.Values

func (c *Context) Query() Query {
	return c.Request.URL.Query()
}
