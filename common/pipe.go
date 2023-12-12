package common

import (
	"github.com/dangduoc08/gooh/context"
)

type ContextPipeable interface {
	Transform(*context.Context, ArgumentMetadata) any
}

type BodyPipeable interface {
	Transform(context.Body, ArgumentMetadata) any
}

type FormPipeable interface {
	Transform(context.Form, ArgumentMetadata) any
}

type QueryPipeable interface {
	Transform(context.Query, ArgumentMetadata) any
}

type HeaderPipeable interface {
	Transform(context.Header, ArgumentMetadata) any
}

type ParamPipeable interface {
	Transform(context.Param, ArgumentMetadata) any
}

type FilePipeable interface {
	Transform(context.File, ArgumentMetadata) any
}

type WSPayloadPipeable interface {
	Transform(context.WSPayload, ArgumentMetadata) any
}

type ArgumentMetadata struct {
	ContextType string
	ParamType   string
}
