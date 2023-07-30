package global

import (
	"time"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/modules/config"
)

type AllExceptionsFilter struct {
	ConfigService config.ConfigService
}

func (ex AllExceptionsFilter) Catch(c gooh.Context, e gooh.HTTPException) {
	httpCode, _ := e.GetHTTPStatus()
	c.Status(httpCode).JSON(
		gooh.Map{
			"code":      e.GetCode(),
			"error":     e.Error(),
			"message":   e.GetResponse(),
			"timestamp": time.Now(),
			"path":      c.URL.Path,
		},
	)
}
