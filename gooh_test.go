package gooh

import (
	"log"
	"testing"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/middlewares"
)

func handler1(ctx *ctx.Context) {
	ctx.JSON(201, "{\"problems\":[{\"Diabetes\":[{\"medications\":[{\"medicationsClasses\":[{\"className\":[{\"associatedDrug\":[{\"name\":\"asprin\",\"dose\":\"\",\"strength\":\"500 mg\"}],\"associatedDrug#2\":[{\"name\":\"somethingElse\",\"dose\":\"\",\"strength\":\"500 mg\"}]}],\"className2\":[{\"associatedDrug\":[{\"name\":\"asprin\",\"dose\":\"\",\"strength\":\"500 mg\"}],\"associatedDrug#2\":[{\"name\":\"somethingElse\",\"dose\":\"\",\"strength\":\"500 mg\"}]}]}]}],\"labs\":[{\"missing_field\":\"missing_value\"}]}],\"Asthma\":[{}]}]}")
}

func TestApplication(test *testing.T) {
	app := Default()
	app.Use(middlewares.RequestLogger)

	userRouter := Router()
	userRouter.Get("/{userId}", handler1)

	app.Group("/users", userRouter)
	app.Use(middlewares.RequestLogger)

	log.Fatal(app.ListenAndServe(":3000", nil))
}
