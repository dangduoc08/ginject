package httpclient

import (
	"time"

	"github.com/dangduoc08/ginject/core"
)

type HTTPClientOnInitFn = func()

type HTTPClientModuleOptions struct {
	IsGlobal bool

	BaseURL string

	Headers map[string]string

	Timeout time.Duration
	OnInit  HTTPClientOnInitFn
}

func Register(opts *HTTPClientModuleOptions) *core.Module {
	if opts == nil {
		opts = &HTTPClientModuleOptions{}
	}

	svc := ClientService{Backend: newHTTPClient(opts)}

	module := core.ModuleBuilder().
		Providers(svc).
		Build()

	module.IsGlobal = opts.IsGlobal
	module.OnInit = opts.OnInit
	return module
}
