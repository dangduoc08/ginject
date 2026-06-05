package cache

import "github.com/dangduoc08/ginject/core"

type CacheOnInitFn = func()

type CacheModuleOptions struct {
	IsGlobal bool
	OnInit   CacheOnInitFn
	Engine   Cache
}

func Register(opts *CacheModuleOptions) *core.Module {
	if opts == nil {
		opts = &CacheModuleOptions{}
	}

	engine := opts.Engine
	if engine == nil {
		engine = newMemoryCache()
	}

	svc := CacheService{
		Engine: engine,
	}

	module := core.ModuleBuilder().
		Providers(svc).
		Build()

	module.IsGlobal = opts.IsGlobal
	module.OnInit = opts.OnInit
	return module
}
