package core

import "net/http"

type response struct {
	http.ResponseWriter
}

type ResponseExtender interface {
	http.ResponseWriter
	Set(fields map[string]string) ResponseExtender                                  // Set HTTP header field to value
	Type(contentType string) ResponseExtender                                       // Set HTTP header content-type
	Send(statusCode int, content string, arguments ...interface{}) ResponseExtender // Response text or HTML
	JSON(statusCode int, datas ...interface{}) ResponseExtender                     // Response JSON
	Error(callback func(interface{}))                                               // Response Error
}
