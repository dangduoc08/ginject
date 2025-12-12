package common

import (
	"github.com/dangduoc08/ginject/ctx"
)

const (
	CONTEXT_PIPEABLE    = "context"
	BODY_PIPEABLE       = "body"
	FORM_PIPEABLE       = "form"
	QUERY_PIPEABLE      = "query"
	HEADER_PIPEABLE     = "header"
	PARAM_PIPEABLE      = "param"
	FILE_PIPEABLE       = "file"
	WS_PAYLOAD_PIPEABLE = "wsPayload"
)

type ContextPipeable interface {
	Transform(*ctx.Context, ArgumentMetadata) any
}

type BodyPipeable interface {
	Transform(ctx.Body, ArgumentMetadata) any
}

type FormPipeable interface {
	Transform(ctx.Form, ArgumentMetadata) any
}

type QueryPipeable interface {
	Transform(ctx.Query, ArgumentMetadata) any
}

type HeaderPipeable interface {
	Transform(ctx.Header, ArgumentMetadata) any
}

type ParamPipeable interface {
	Transform(ctx.Param, ArgumentMetadata) any
}

type FilePipeable interface {
	Transform(ctx.File, ArgumentMetadata) any
}

type WSPayloadPipeable interface {
	Transform(ctx.WSPayload, ArgumentMetadata) any
}

type ArgumentMetadata struct {
	ContextType string
	ParamType   string
}
