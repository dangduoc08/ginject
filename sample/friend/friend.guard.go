package friend

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/modules/config"
)

type APIKeyGuard struct {
	ConfigService config.ConfigService
}

func (g APIKeyGuard) CanActivate(c gooh.Context) bool {
	fmt.Println("module APIKeyGuard", c.GetType())
	// return c.Header().Get("X-API-KEY") == g.ConfigService.Get("API_KEY")
	return true
}
