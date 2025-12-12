package manufacturers

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
)

type ManufacturerMiddleware struct {
	common.Logger
}

func (instance ManufacturerMiddleware) Use(c ginject.Context, next ginject.Next) {
	fmt.Println("[Module] Manufacturer middleware")
	instance.Info("test")

	next()
}
