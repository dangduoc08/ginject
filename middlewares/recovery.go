package middlewares

import (
	"net/http"
	"strconv"

	"github.com/dangduoc08/gooh/context"
)

func Recovery(c *context.Context) {
	c.Event.On(context.REQUEST_FAILED, func(args ...any) {
		errorStr := "Unknown Error"
		switch args[0].(type) {
		case error:
			errorStr = args[0].(error).Error()
		case string:
			errorStr = args[0].(string)
		case int:
		case int8:
		case int16:
		case int32:
		case int64:
		case uint:
		case uint8:
		case uint16:
		case uint32:
		case uint64:
		case float32:
		case float64:
		case complex64:
		case complex128:
		case uintptr:
			errorStr = strconv.Itoa(args[0].(int))
		}

		c.Status(http.StatusInternalServerError)
		c.Event.Emit(context.REQUEST_FINISHED)
		http.Error(c.ResponseWriter, errorStr, http.StatusInternalServerError)
	})
	c.Next()
}
