package httpclient

import (
	"encoding/json"
	"io"
	"net/http"
)

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	BodyStream io.ReadCloser
	Raw        *http.Response
	Timing     *TimingInfo
}

func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

func (r *Response) Text() string { return string(r.Body) }

func (r *Response) Bytes() []byte { return r.Body }
