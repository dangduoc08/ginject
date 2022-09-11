package gooh

import (
	"fmt"
	"log"
	"testing"

	"github.com/dangduoc08/gooh/ctx"
)

func middleware1(ctx *ctx.Context) {
	fmt.Println("middleware1")
}
func middleware2(ctx *ctx.Context) {
	fmt.Println("middleware2")
}
func handler1(ctx *ctx.Context) {
	fmt.Println("handler1")
}
func handler2(ctx *ctx.Context) {
	fmt.Println("handler2")
}
func middleware3(ctx *ctx.Context) {
	fmt.Println("middleware3")
}
func middleware4(ctx *ctx.Context) {
	fmt.Println("middleware4")
}

func TestApplication(test *testing.T) {
	app := Default()

	userRouter := Router()
	userRouter.Put("/users/{userId}", handler1)

	v1 := Router()
	v1.Group("/v1/", userRouter)

	app.Group("/", v1)

	for _, el := range app.router.RouteMapDataArr {
		for route, _ := range el {
			log.SetPrefix("RouteExplorer")
			log.Default().Println(route)
		}
	}

	log.Fatal(app.ListenAndServe(":3000", nil))
}
