package gooh

// func TestHttp(test *testing.T) {
// 	app := Default()

// 	// categoryRouter := Router()
// 	app.For("/")()
// 	app.Use(nil)
// 	app.Get("/")
// 	app.Get("/{categoryId}")
// 	app.Post("/")
// 	app.Put("/{categoryId}")
// 	app.Delete("/{categoryId}")

// 	userRouter := Router()
// 	userRouter.For("/users")()
// 	userRouter.Use(nil)
// 	userRouter.Get("/users")
// 	userRouter.Get("/users/{userId}")
// 	userRouter.Post("/users")
// 	userRouter.Put("/users/{userId}")
// 	userRouter.Delete("/users/{userId}")

// 	// productRouter := Router()
// 	// productRouter.For("/products")()
// 	// productRouter.Use(nil)
// 	// productRouter.Get("/products")
// 	// productRouter.Get("/products/{productId}")
// 	// productRouter.Post("/products")
// 	// productRouter.Put("/products/{productId}")
// 	// productRouter.Delete("/products/{productId}")

// 	// categoryRouter.Group("/categories", categoryRouter)

// 	app.Group("/v1", userRouter)

// 	log.Fatal(app.ListenAndServe(":8080", nil))
// }
