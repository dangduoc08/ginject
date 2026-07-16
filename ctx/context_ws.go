package ctx

import (
	"golang.org/x/net/websocket"
)

func (c *Context) SetWSConfig(wsCfg *websocket.Config) {
	c.wsCfg = wsCfg
}

func (c *Context) GetWSConfig() *websocket.Config {
	return c.wsCfg
}

func (c *Context) SetWSConn(conn *websocket.Conn) *Context {
	c.wsConn = conn
	return c
}

func (c *Context) WSConn() *websocket.Conn {
	return c.wsConn
}

func (c *Context) SetWSPayload(p WSPayload) *Context {
	c.wsPayload = p
	return c
}

func (c *Context) WSPayload() WSPayload {
	return c.wsPayload
}
