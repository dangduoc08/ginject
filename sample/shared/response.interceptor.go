package shared

import (
	"fmt"
	"net/http"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type ResponseInterceptor struct {
	common.Logger
}

func (instance ResponseInterceptor) Intercept(c ginject.Context, aggregation ginject.Aggregation) any {
	return aggregation.Pipe(
		aggregation.Transform(func(c ginject.Context, data any) any {
			transformedData := ginject.Map{
				"data": data,
			}
			return transformedData
		}),
		aggregation.Error(func(c ginject.Context, e any) any {

			// The framework pre-sets c.Code (e.g. 201 for POST) before the
			// handler runs, so the exception's own status must be applied
			// explicitly here — otherwise c.JSON would write the error body
			// with whatever success status was assumed beforehand.
			ex, ok := e.(exception.Exception)
			if !ok {
				c.Status(http.StatusInternalServerError).JSON(ginject.Map{
					"error": fmt.Sprint(e),
				})
				return nil
			}

			httpCode, _ := ex.GetHTTPStatus()
			if httpCode == 0 {
				httpCode = http.StatusInternalServerError
			}

			// ex.Error() is the generic HTTP status text; the descriptive
			// message passed to e.g. exception.BadRequestException(...) is
			// carried separately in ex.GetResponse().
			c.Status(httpCode).JSON(ginject.Map{
				"code":    ex.GetCode(),
				"error":   ex.Error(),
				"message": ex.GetResponse(),
			})

			return nil
		}),
	)
}
