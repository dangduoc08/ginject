package httpclient

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"
)

type Client interface {
	Get(path string) RequestBuilder
	Post(path string) RequestBuilder
	Put(path string) RequestBuilder
	Patch(path string) RequestBuilder
	Delete(path string) RequestBuilder
	Head(path string) RequestBuilder
	Options(path string) RequestBuilder

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

type RequestBuilder interface {
	Context(ctx context.Context) RequestBuilder
	Header(key, value string) RequestBuilder
	Headers(headers map[string]string) RequestBuilder
	Query(key string, value any) RequestBuilder
	JSON(v any) RequestBuilder
	Form(v any) RequestBuilder
	Body(r io.Reader) RequestBuilder

	File(field, filename string, r io.Reader) RequestBuilder

	Field(key, value string) RequestBuilder
	Timeout(d time.Duration) RequestBuilder
	Retry(count int) RequestBuilder
	RetryBackoff(initial, max time.Duration) RequestBuilder

	Stream() RequestBuilder

	SSE() RequestBuilder
	OnProgress(fn func(Progress)) RequestBuilder
	Send() (*Response, error)
}
