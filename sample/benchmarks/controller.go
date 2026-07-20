package benchmarks

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/modules/httpclient"
)

type Controller struct {
	common.REST
	common.WS
	common.Middleware
	common.Guard
	common.Interceptor
	common.ExceptionFilter
	httpclient.ClientService
}

func (instance Controller) NewController() core.Controller {
	instance.BindMiddleware(LogModuleMiddleware{})
	instance.BindGuard(LogModuleHTTPGuard{})
	instance.BindGuard(LogModuleWSGuard{})
	instance.BindInterceptor(LogModuleHTTPInterceptor{})
	instance.BindInterceptor(LogModuleWSInterceptor{})
	instance.BindExceptionFilter(LogModuleHTTPExceptionFilter{})
	instance.BindExceptionFilter(LogModuleWSExceptionFilter{})

	return instance
}

func (instance Controller) READ_ping(query ginject.Query) ginject.Map {
	fmt.Println("[REST] READ_ping triggered")

	if query.Get("error") == "true" {
		panic(exception.InternalServerErrorException("READ_ping error triggered"))
	}

	return ginject.Map{
		"message": "Hello, World!",
	}
}

func (instance Controller) SUBSCRIBE_chat_PERSON_ANY() ginject.Map {
	return ginject.Map{
		"message": "Hello, World!",
	}
}
