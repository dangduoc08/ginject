package storage

import "github.com/dangduoc08/ginject/core"

type OnInitFn = func()

type StoreModuleOptions struct {
	IsGlobal bool
	Path     string
	OnInit   OnInitFn

	DisableGitignore bool
}

func Register(opts *StoreModuleOptions) *core.Module {
	if opts == nil {
		opts = &StoreModuleOptions{}
	}
	if opts.Path == "" {
		panic("store: StoreModuleOptions.Path must not be empty")
	}

	if !opts.DisableGitignore {
		ensureGitignoreEntry(opts.Path)
	}

	db, err := Open(opts.Path)
	if err != nil {
		panic("store: failed to open database: " + err.Error())
	}

	svc := StoreService{DB: db}
	module := core.ModuleBuilder().
		Providers(svc).
		Build()

	module.IsGlobal = opts.IsGlobal
	module.OnInit = opts.OnInit
	return module
}
