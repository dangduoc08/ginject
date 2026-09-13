package infra

import (
	"os"
	"path/filepath"

	"github.com/dangduoc08/ginject/modules/cache"
	"github.com/dangduoc08/ginject/modules/storage"
)

var cwd, _ = os.Getwd()

var StorageModule = storage.Register(&storage.StoreModuleOptions{
	Path: filepath.Join(cwd, "data", "shop"),
})

var CacheModule = cache.Register(&cache.CacheModuleOptions{})
