package manufacturers

import (
	"fmt"

	"github.com/dangduoc08/gogo/ctx"
)

func ManufacturerMiddleware(c *ctx.Context) {
	fmt.Println("[Module] Manufacturer middleware")

	c.Next()
}
