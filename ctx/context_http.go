package ctx

import (
	"net/http"

	"github.com/dangduoc08/ginject/internal/str"
)

func (c *HTTPContext) Status(code int) *HTTPContext {
	c.Code = code
	return c
}

func (c *HTTPContext) Text(data string, args ...any) {
	c.dataWriter = &Text{
		data:           data,
		args:           args,
		responseWriter: c.ResponseWriter,
	}
	c.dataWriter.WriteData(c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}

func (c *HTTPContext) JSON(data ...any) {
	c.dataWriter = &JSON{
		data:           data,
		responseWriter: c.ResponseWriter,
	}
	c.dataWriter.WriteData(c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}

func (c *HTTPContext) JSONP(data ...any) {
	callback := str.RemoveSpace(c.URL.Query().Get("callback"))
	if callback == "" {
		c.JSON(data...)
		return
	}

	c.dataWriter = &JSONP{
		data:           data,
		responseWriter: c.ResponseWriter,
		callback:       callback,
	}
	c.dataWriter.WriteData(c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}

func (c *HTTPContext) Redirect(url string) {
	c.Status(http.StatusMovedPermanently)
	http.Redirect(c.ResponseWriter, c.Request, url, c.Code)
	_ = c.Broker.Publish(RequestFinished, c)
}
