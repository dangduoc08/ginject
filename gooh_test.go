package gooh

import (
	"log"
	"net/http"
	"testing"

	"github.com/dangduoc08/gooh/ctx"
)

func handler1(c *ctx.Context) {
	c.Status(http.StatusCreated).JSON(Map{
		"name": "Hello World!",
	})
}

func TestApplication(test *testing.T) {
	app := Default()
	// app.Use(middlewares.RequestLogger)

	// userRouter := Router()
	// userRouter.Get("/{userId}/all", handler1)

	app.Get("/users/{userId}/all", handler1)
	// app.Get("/users/{userId}/*", handler1)

	log.Fatal(app.ListenAndServe(":8080", nil))
}
