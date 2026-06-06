package cache

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/memcache"
)

type CacheOnInitFn = func()

type CacheModuleOptions struct {
	IsGlobal bool
	OnInit   CacheOnInitFn
	Backend  Cache
}

func Register(opts *CacheModuleOptions) *core.Module {
	if opts == nil {
		opts = &CacheModuleOptions{}
	}

	backend := opts.Backend
	if backend == nil {
		backend = memcache.NewMemoryCache()
	}

	svc := CacheService{
		Backend: backend,
	}

	module := core.ModuleBuilder().
		Providers(svc).
		Build()

	module.IsGlobal = opts.IsGlobal
	module.OnInit = opts.OnInit
	return module
}
