package core

import (
	"context"
	"net/http"
)

type Request struct {
	*http.Request
	Params map[string]string
	ctx    context.Context
}
