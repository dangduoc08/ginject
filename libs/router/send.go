package router

import "fmt"

// Implements ResponseExtender interface
func (res *response) Send(content string, arguments ...interface{}) {
	fmt.Fprintf(res.ResponseWriter, content, arguments...)
}
