package httpclient

import (
	"encoding/json"
	"io"
	"net/http"
)

// Response holds the completed HTTP response with helper methods.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	BodyStream io.ReadCloser
	Raw        *http.Response
	Timing     *TimingInfo
}

// JSON unmarshals the body into v.
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

// Text returns the body as a UTF-8 string.
func (r *Response) Text() string { return string(r.Body) }

// Bytes returns the raw body bytes.
func (r *Response) Bytes() []byte { return r.Body }
