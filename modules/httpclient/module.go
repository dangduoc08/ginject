package httpclient

import (
	"time"

	"github.com/dangduoc08/ginject/core"
)

type HttpClientOnInitFn = func()

// HttpClientModuleOptions configures the default HTTP client for the module.
type HttpClientModuleOptions struct {
	IsGlobal bool
	// BaseURL is prepended to every relative request path.
	BaseURL string
	// Headers are sent on every request unless overridden per-request.
	Headers map[string]string
	// Timeout is the default client-level timeout for all requests.
	Timeout time.Duration
	OnInit  HttpClientOnInitFn
}

// Register creates a Ginject module that provides an injectable ClientService.
func Register(opts *HttpClientModuleOptions) *core.Module {
	if opts == nil {
		opts = &HttpClientModuleOptions{}
	}

	svc := ClientService{Backend: newHTTPClient(opts)}

	module := core.ModuleBuilder().
		Providers(svc).
		Build()

	module.IsGlobal = opts.IsGlobal
	module.OnInit = opts.OnInit
	return module
}
