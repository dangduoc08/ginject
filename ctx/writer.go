package ctx

import (
	"fmt"
	"io"
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
	_, _ = json.responseWriter.Write(jsonBuf)
}

func (jsonp *JSONP) WriteData(statusCode int) {
	jsonBuf, err := toJSONBuffer(jsonp.data.([]any)...)
	if err != nil {
		panic(err.Error())
	}

	jsonp.responseWriter.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	jsonp.responseWriter.WriteHeader(statusCode)
	_, _ = fmt.Fprint(jsonp.responseWriter, toJSONP(string(jsonBuf), jsonp.callback))
}

func (text *Text) WriteData(statusCode int) {
	text.responseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	text.responseWriter.WriteHeader(statusCode)
	args := text.args.([]any)
	if len(args) == 0 {
		_, _ = io.WriteString(text.responseWriter, text.data)
	} else {
		_, _ = fmt.Fprintf(text.responseWriter, text.data, args...)
	}
}
