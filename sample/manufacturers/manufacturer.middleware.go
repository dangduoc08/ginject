package manufacturers

import (
	"fmt"

	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
)

type ManufacturerMiddleware struct {
	common.Logger
}

func (instance ManufacturerMiddleware) Use(c gogo.Context, next gogo.Next) {
	fmt.Println("[Module] Manufacturer middleware")
	instance.Info("test")

	next()
}
