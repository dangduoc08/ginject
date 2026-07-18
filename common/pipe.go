package common

import (
	"github.com/dangduoc08/ginject/ctx"
)

const (
	ContextPipeableKey    = "context"
	BodyPipeableKey       = "body"
	FormPipeableKey       = "form"
	QueryPipeableKey      = "query"
	HeaderPipeableKey     = "header"
	ParamPipeableKey      = "param"
	FilePipeableKey       = "file"
	WSPayloadPipeableKey = "wsPayload"
)

type ContextPipeable interface {
	Transform(*ctx.HTTPContext, ArgumentMetadata) any
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
