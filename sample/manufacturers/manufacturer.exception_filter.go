package manufacturers

import (
	"fmt"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

type ManufacturerExceptionFilter struct{}

func (g ManufacturerExceptionFilter) Catch(c *ctx.Context, ex *exception.Exception) {
	fmt.Println("[Module] Manufacturer exception filter")
	internalServerErrorException := exception.InternalServerErrorException("Unhandled exception has occurred")

	code := ex.GetCode()
	if code == "" {
		code = internalServerErrorException.GetCode()
	}

	err := ex.Error()
	if err == "" {
		err = internalServerErrorException.Error()
	}
	data := ctx.Map{
		"module": "Manufacturer",
		"code":   code,
		"error":  err,
	}

	message := ex.GetResponse()
	switch reflect.TypeOf(message).Kind() {
	case reflect.String, reflect.Map, reflect.Slice, reflect.Struct:
		data["message"] = message
	default:
		data["message"] = internalServerErrorException.GetResponse()
	}

	httpCode, httpText := ex.GetHTTPStatus()
	if httpText == "" {
		httpCode, _ = internalServerErrorException.GetHTTPStatus()
	}

	c.Status(httpCode).JSON(data)
}
