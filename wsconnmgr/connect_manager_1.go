package wsconnmgr

import (
	"github.com/dangduoc08/ginject/modules/cache"
)

type Options struct {
	Engine cache.Cache
}

type WSConnectionManager struct {
	store cache.Cache
}

func New(opts *Options) *WSConnectionManager {
	var engine cache.Cache

	if opts != nil {
		engine = opts.Engine
	}

	if engine == nil {
		engine = cache.NewMemoryCache()
	}

	return &WSConnectionManager{
		store: engine,
	}
}
