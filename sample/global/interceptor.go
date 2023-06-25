package global

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/modules/config"
)

type Interceptor struct {
	ConfigService config.ConfigService
}

func (cacheInterceptor Interceptor) NewInterceptor() common.Interceptable {
	fmt.Println("Global Interceptor NewInterceptor")
	return cacheInterceptor
}

func (cacheInterceptor Interceptor) Intercept(c gooh.Context, next any) any {
	fmt.Println("Global Interceptor invoke Intercept")
	return 10
}
