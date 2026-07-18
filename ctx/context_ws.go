package ctx

import (
	"golang.org/x/net/websocket"
)

func (c *HTTPContext) SetWSConfig(wsCfg *websocket.Config) {
	c.wsCfg = wsCfg
}

func (c *HTTPContext) GetWSConfig() *websocket.Config {
	return c.wsCfg
}

func (c *HTTPContext) SetWSConn(conn *websocket.Conn) *HTTPContext {
	c.wsConn = conn
	return c
}

func (c *HTTPContext) WSConn() *websocket.Conn {
	return c.wsConn
}

func (c *HTTPContext) SetWSPayload(p WSPayload) *HTTPContext {
	c.wsPayload = p
	return c
}

func (c *HTTPContext) WSPayload() WSPayload {
	return c.wsPayload
}
