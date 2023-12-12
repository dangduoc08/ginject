package ctx

import (
	"fmt"
	"net/http"
)

type DataWriter interface {
	WriteData(int)
}

type JSON struct {
	responseWriter http.ResponseWriter
	data           any
}

type JSONP struct {
	callback       string
	responseWriter http.ResponseWriter
	data           any
}

type Text struct {
	responseWriter http.ResponseWriter
	data           string
	args           any
}

func (json *JSON) WriteData(statusCode int) {
	jsonBuf, err := toJSONBuffer(json.data.([]any)...)
	if err != nil {
		panic(err.Error())
	}

	json.responseWriter.Header().Set("Content-Type", "application/json")
	json.responseWriter.WriteHeader(statusCode)
	json.responseWriter.Write(jsonBuf)
}

func (jsonp *JSONP) WriteData(statusCode int) {
	jsonBuf, err := toJSONBuffer(jsonp.data.([]any)...)
	if err != nil {
		panic(err.Error())
	}

	jsonp.responseWriter.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	jsonp.responseWriter.WriteHeader(statusCode)
	fmt.Fprint(jsonp.responseWriter, toJSONP(string(jsonBuf), jsonp.callback))
}

func (text *Text) WriteData(statusCode int) {
	text.responseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	text.responseWriter.WriteHeader(statusCode)
	fmt.Fprintf(text.responseWriter, text.data, text.args.([]any)...)
}
