package httpclient

import (
	"fmt"
	"net/http"
)

// Error wraps a failed HTTP request with the originating request and response.
type Error struct {
	Request  *http.Request
	Response *Response
	Cause    error
}

func (e *Error) Error() string {
	if e.Response != nil && e.Request != nil {
		return fmt.Sprintf("httpclient: %s %s returned status %d",
			e.Request.Method, e.Request.URL.String(), e.Response.StatusCode)
	}
	if e.Cause != nil {
		return "httpclient: " + e.Cause.Error()
	}
	return "httpclient: unknown error"
}

func (e *Error) Unwrap() error { return e.Cause }
