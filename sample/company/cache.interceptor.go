package company

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/modules/config"
)

type CacheGetCompaniesInterceptor struct {
	ConfigService config.ConfigService
}

func (i CacheGetCompaniesInterceptor) Intercept(c gooh.Context, a gooh.Aggregation) any {
	if c.Query().Get("isCache") == "true" {
		return "Data cached!"
	}

	return a.Pipe(
		a.Consume(func(data any) any {
			return data
		}),
	)
}
