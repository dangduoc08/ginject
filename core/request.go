package core

import (
	"net/http"
)

type Request struct {
	*http.Request
	Vars Var[interface{}]
}
