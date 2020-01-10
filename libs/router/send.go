package router

import "fmt"

func (res *response) Send(content string, arguments ...interface{}) {
	fmt.Fprintf(res.ResponseWriter, content, arguments...)
}
