package express

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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

func (res *response) Set(fields map[string]string) ResponseExtender {
	for field, value := range fields {
		ch := make(chan bool)
		go func(field, value string, ch chan<- bool) {
			res.Header().Add(field, value)
			ch <- true
		}(field, value, ch)
		<-ch
		defer close(ch)
	}
	return res
}

func (res *response) Type(contentType string) ResponseExtender {
	res.Header().Set("Content-Type", contentType)
	return res
}

func (res *response) Send(statusCode int, content string, arguments ...interface{}) ResponseExtender {
	res.WriteHeader(statusCode)
	fmt.Fprintf(res.ResponseWriter, content, arguments...)
	return res
}

func (res *response) JSON(statusCode int, datas ...interface{}) ResponseExtender {
	data := datas[0]
	switch data.(type) {
	case string:
		fStr := fmt.Sprintf(data.(string), datas[1:]...)
		data = json.RawMessage(fStr)
	default:
		if len(datas) > 1 {
			panic("Error: Too many arguments")
		}
	}

	buffer, err := json.Marshal(&data)
	if err != nil {
		panic(err.Error())
	}
	res.Type("application/json")
	res.WriteHeader(statusCode)
	res.Write(buffer)
	return res
}

func (res *response) Error(callback func(interface{})) {
	if rec := recover(); rec != nil {
		callback(rec)
	}
}
