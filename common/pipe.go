package common

type Pipeable interface {
	Transform(any, ArgumentMetadata) any
}

type ArgumentMetadata struct {
	ParamType string
}
