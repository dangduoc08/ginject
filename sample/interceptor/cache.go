package interceptor

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/modules/config"
)

type CacheInterceptor struct {
	ConfigService config.ConfigService
}

func (cacheInterceptor CacheInterceptor) NewInterceptor() common.Interceptable {
	fmt.Println("CacheInterceptor NewInterceptor")
	return cacheInterceptor
}

func (cacheInterceptor CacheInterceptor) Intercept(c gooh.Context, next any) any {
	fmt.Println("CacheInterceptor invoke Intercept")
	return 10
}
