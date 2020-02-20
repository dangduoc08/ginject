package gogo

import (
	"context"
	"errors"
	"net/http"
)

// Request embed
// *http.Request struct
// extend
// Params to hold params from URL
// ctx to create context.WithValue
type Request struct {
	*http.Request
	Params map[string]string
	ctx    context.Context
}

// WithMiddleware helps pass params between middlewares
func (r *Request) WithMiddleware(args ...interface{}) (interface{}, error) {
	var argsLength int = len(args)

	if argsLength == 0 {
		return nil, errors.New("Nil argument")
	}

	if argsLength > 2 {
		return nil, errors.New("Too many arguments")
	}

	key := args[0]

	if argsLength == 2 {
		value := args[1]
		r.ctx = context.WithValue(r.ctx, key, value)
		return nil, nil
	}

	return r.ctx.Value(key), nil
}
