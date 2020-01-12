package router

import "fmt"

// Implements ResponseExtender interface
func (res *response) Send(content string, arguments ...interface{}) ResponseExtender {
	fmt.Fprintf(res.ResponseWriter, content, arguments...)
	return res
}

func (res *response) Status(statusCode int) ResponseExtender {
	res.WriteHeader(statusCode)
	return res
}
