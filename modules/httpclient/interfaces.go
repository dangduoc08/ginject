package httpclient

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"
)

// Client builds and executes HTTP requests with a shared configuration.
type Client interface {
	Get(path string) RequestBuilder
	Post(path string) RequestBuilder
	Put(path string) RequestBuilder
	Patch(path string) RequestBuilder
	Delete(path string) RequestBuilder
	Head(path string) RequestBuilder
	Options(path string) RequestBuilder

	// Use appends middleware to the client's global chain.
	Use(middlewares ...Middleware)

	SetBaseURL(u string)
	SetHeader(key, value string)
	SetHeaders(headers map[string]string)

	SetTimeout(d time.Duration)
	SetRetry(count int)
	SetRetryBackoff(initial, max time.Duration)

	EnableDebug()
	EnableCookies()
	RequireHTTPS(v bool)
	SetTLSConfig(cfg *tls.Config)
	SetMaxResponseSize(n int64)
	SetValidateHost(fn func(host string) bool)
	SetValidateStatus(fn func(code int) bool)

	OnBeforeRequest(fn func(*http.Request) error)
	OnAfterResponse(fn func(*Response) error)
	OnError(fn func(error))

	Download(rawURL, filepath string) error
	DownloadWithProgress(rawURL, filepath string, fn func(Progress)) error
}

// RequestBuilder constructs and sends a single HTTP request.
type RequestBuilder interface {
	Context(ctx context.Context) RequestBuilder
	Header(key, value string) RequestBuilder
	Headers(headers map[string]string) RequestBuilder
	Query(key string, value any) RequestBuilder
	JSON(v any) RequestBuilder
	Form(v any) RequestBuilder
	Body(r io.Reader) RequestBuilder
	// File adds a file field to a multipart/form-data request.
	File(field, filename string, r io.Reader) RequestBuilder
	// Field adds a text field to a multipart/form-data request.
	Field(key, value string) RequestBuilder
	Timeout(d time.Duration) RequestBuilder
	Retry(count int) RequestBuilder
	RetryBackoff(initial, max time.Duration) RequestBuilder
	// Stream keeps the response body open; caller must close Response.BodyStream.
	Stream() RequestBuilder
	// SSE is like Stream but signals that the response is a text/event-stream.
	SSE() RequestBuilder
	OnProgress(fn func(Progress)) RequestBuilder
	Send() (*Response, error)
}
