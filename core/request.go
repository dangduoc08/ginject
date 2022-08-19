package core

import (
	"net/http"
)

type Request struct {
	*http.Request
	Params Param[interface{}]
}
