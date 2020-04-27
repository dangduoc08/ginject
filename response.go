package gogo

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type response struct {
	http.ResponseWriter
}

// ResponseExtender embed
// http.ResponseWriter interface
// extend
// more build-in methods
type ResponseExtender interface {
	http.ResponseWriter
	Set(fields map[string]string) ResponseExtender                                  // Set HTTP header field to value
	Type(contentType string) ResponseExtender                                       // Set HTTP header content-type
	Send(statusCode int, content string, arguments ...interface{}) ResponseExtender // Response text or HTML
	JSON(statusCode int, datas ...interface{}) ResponseExtender                     // Response JSON
	Error(callback func(interface{}))                                               // Response Error
}

// Set response HTTP headers
// by implement Header.().Add() method of http.ResponseWriter interface
func (res *response) Set(fields map[string]string) ResponseExtender {
	for field, value := range fields {
		res.Header().Add(field, value)
	}
	return res
}

// Set content-type header
// by implement Header.().Set() method of http.ResponseWriter interface
func (res *response) Type(contentType string) ResponseExtender {
	res.Header().Set("Content-Type", contentType)
	return res
}

// Send string or HTML string
// second params are variable value
func (res *response) Send(statusCode int, content string, arguments ...interface{}) ResponseExtender {
	res.WriteHeader(statusCode)
	fmt.Fprintf(res.ResponseWriter, content, arguments...)
	return res
}

// JSON to reponse JSON type
func (res *response) JSON(statusCode int, datas ...interface{}) ResponseExtender {

	// Datas can be string, struct or map[string]interface{}
	data := datas[0]
	switch data.(type) {

	// Handle case datas are string
	case string:

		// Format string with params
		fStr := fmt.Sprintf(data.(string), datas[1:]...)

		// Parse string to raw JSON
		data = json.RawMessage(fStr)
	default:

		// If datas are not string
		// only accept one argument
		if len(datas) > 1 {
			panic("JSON use map or struct type only accepts a agrument")
		}
	}

	// Parse to JSON
	buffer, err := json.Marshal(&data)
	if err != nil {
		panic(err.Error())
	}
	res.Type("application/json")
	res.WriteHeader(statusCode)
	res.Write(buffer)
	return res
}

// Error invoke recover function
// Should be use with defer, place at the begin of handler
// ex: defer res.Error
func (res *response) Error(callback func(interface{})) {
	if rec := recover(); rec != nil {
		callback(rec)
	}
}
