package cache

import "github.com/dangduoc08/ginject/core"

type CacheOnInitFn = func()

type CacheModuleOptions struct {
	IsGlobal bool
	OnInit   CacheOnInitFn
}

func Register(opts *CacheModuleOptions) *core.Module {
	if opts == nil {
		opts = &CacheModuleOptions{}
	}

	svc := CacheService{Backend: newMemoryCache()}

	module := core.ModuleBuilder().
		Providers(svc).
		Build()

	module.IsGlobal = opts.IsGlobal
	module.OnInit = opts.OnInit
	return module
}
