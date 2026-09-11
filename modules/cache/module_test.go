package cache

import "testing"

func TestRegister_OwnedBackend_StoppedOnShutdown(t *testing.T) {
	module := Register(nil)

	if module.OnShutdown == nil {
		t.Fatal("a cache module that owns its backend must stop it on shutdown, otherwise the sweeper goroutine leaks")
	}

	module.OnShutdown()
	module.OnShutdown()
}

type zzNoopCache struct{ Cache }

func TestRegister_InjectedBackend_NotStopped(t *testing.T) {
	module := Register(&CacheModuleOptions{Backend: zzNoopCache{}})

	if module.OnShutdown != nil {
		t.Error("a caller-supplied backend must be left to the caller to stop")
	}
}
