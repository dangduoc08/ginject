package ctx

type Context interface {
	Reset()
	SetID()
	GetID() string
}

type (
	Map         map[string]any
	ErrFunc     func(error)
	HTTPHandler = func(*HTTPContext)
	WSHandler   = func(*WSContext)
	Next        = func()
	Redirect    = func(string)
)

const (
	CatchException = "CatchException"
)
