package infra

import (
	"os"
	"path/filepath"

	"github.com/dangduoc08/ginject/modules/cache"
	"github.com/dangduoc08/ginject/modules/storage"
)

var cwd, _ = os.Getwd()

// StorageModule persists every shop table — users, stores, categories and
// products — to a single database under <cwd>/data/shop. Feature modules
// import it directly rather than relying on a global module, so the
// dependency on shared storage stays explicit.
var StorageModule = storage.Register(&storage.StoreModuleOptions{
	Path: filepath.Join(cwd, "data", "shop"),
})

// CacheModule backs session tokens with a TTL so logins expire automatically.
var CacheModule = cache.Register(&cache.CacheModuleOptions{})
