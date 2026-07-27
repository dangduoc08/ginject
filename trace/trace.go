package trace

import "time"

const EventName = "trace"

const (
	StageMiddleware      = "middleware"
	StageGuard           = "guard"
	StagePreInterceptor  = "pre_interceptor"
	StagePostInterceptor = "post_interceptor"
	StageExceptionFilter = "exception_filter"
	StageHandler         = "handler"
	StageComplete        = "complete"
)

const (
	TransportHTTP = "HTTP"
	TransportWS   = "WS"
	TransportGQL  = "GQL"
	TransportRPC  = "RPC"
)

type Event struct {
	ID        string
	Stage     string
	Name      string
	Transport string
	Duration  time.Duration
	Operation string
	Target    string
	Code      int
}
