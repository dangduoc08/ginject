package express

func ExecWorkerPool(worker int, middlewares ...Handler) Handler {
	return func(req *Request, res ResponseExtender, next func()) {
		var totalMiddleware int = len(middlewares)
		var lastExecutorIndex int = len(middlewares) - 1
		queue := make(chan Handler, totalMiddleware)
		receiver := make(chan bool, totalMiddleware)
		defer close(queue)
		defer close(receiver)

		// Run concurency base on the number of worker passed
		// every time when
		for i := 0; i < worker; i++ {
			go func(queue <-chan Handler, receiver chan<- bool) {
				for task := range queue {
					var isNextCalled bool
					task(req, res, func() {
						isNextCalled = true
					})
					receiver <- isNextCalled
				}
			}(queue, receiver)
		}

		// Push middleware functions to queue
		for _, task := range middlewares {
			queue <- task
		}

		var isAllNextCalled bool = true
		// Receive result from queue channel
		for mwIndex, _ := range middlewares {
			isNextCalled := <-receiver
			if !isNextCalled {
				isAllNextCalled = false
			}
			// If all middlewares in pool invoked next function
			// call to the next middleware which out of worker pool
			if mwIndex == lastExecutorIndex && isAllNextCalled {
				next()
			}
		}
	}
}
