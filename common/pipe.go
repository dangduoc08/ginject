package common

import (
	"net/http"
	"net/url"

	"github.com/dangduoc08/gooh/context"
)

type QueryPipeable interface {
	Transform(url.Values, ArgumentMetadata) any
}

type ParamPipeable interface {
	Transform(context.Values, ArgumentMetadata) any
}

type HeaderPipeable interface {
	Transform(http.Header, ArgumentMetadata) any
}

type ArgumentMetadata struct {
	ParamType string
}
