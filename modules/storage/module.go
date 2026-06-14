package storage

import "github.com/dangduoc08/ginject/core"

type OnInitFn = func()

// StoreModuleOptions configures the store module.
type StoreModuleOptions struct {
	IsGlobal bool
	Path     string  // directory where data files are stored; required
	OnInit   OnInitFn

	// DisableGitignore turns off the default behavior of adding Path to the
	// project's .gitignore (creating the file if it doesn't exist yet).
	DisableGitignore bool
}

// Register creates and returns a configured store module.
// Panics if Path is empty or the database cannot be opened.
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
