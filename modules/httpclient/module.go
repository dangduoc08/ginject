package httpclient

import (
	"time"

	"github.com/dangduoc08/ginject/core"
)

type HTTPClientOnInitFn = func()

// HTTPClientModuleOptions configures the default HTTP client for the module.
type HTTPClientModuleOptions struct {
	IsGlobal bool
	// BaseURL is prepended to every relative request path.
	BaseURL string
	// Headers are sent on every request unless overridden per-request.
	Headers map[string]string
	// Timeout is the default client-level timeout for all requests.
	Timeout time.Duration
	OnInit  HTTPClientOnInitFn
}

// Register creates a Ginject module that provides an injectable ClientService.
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
