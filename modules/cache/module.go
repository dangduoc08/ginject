package cache

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/memorycache"
)

type CacheOnInitFn = func()

type CacheModuleOptions struct {
	IsGlobal bool
	OnInit   CacheOnInitFn
	Backend  Cache

	// MaxEntries caps the default in-memory backend. 0 uses
	// memorycache.DefaultMaxEntries; negative means unlimited. Ignored when
	// Backend is supplied.
	MaxEntries int
}

func Register(opts *CacheModuleOptions) *core.Module {
	if opts == nil {
		opts = &CacheModuleOptions{}
	}

	backend := opts.Backend
	var ownedBackend *memorycache.MemoryCache
	if backend == nil {
		var cacheOpts []memorycache.Option
		if opts.MaxEntries != 0 {
			cacheOpts = append(cacheOpts, memorycache.WithMaxEntries(opts.MaxEntries))
		}
		ownedBackend = memorycache.NewMemoryCache(cacheOpts...)
		backend = ownedBackend
	}

	svc := CacheService{
		Backend: backend,
	}

	module := core.ModuleBuilder().
		Providers(svc).
		Build()

	module.IsGlobal = opts.IsGlobal
	module.OnInit = opts.OnInit
	if ownedBackend != nil {
		module.OnShutdown = ownedBackend.Stop
	}

	return module
}
