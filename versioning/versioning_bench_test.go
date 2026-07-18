package versioning

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

func benchCtxQuery(key, val string) *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	u := &url.URL{RawQuery: url.Values{key: {val}}.Encode()}
	c.Request = &http.Request{URL: u}
	return c
}

func benchCtxHeader(key, val string) *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	c.Request = &http.Request{Header: http.Header{key: {val}}}
	return c
}

func benchCtxEmpty() *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	c.Request = &http.Request{URL: &url.URL{}, Header: http.Header{}}
	return c
}

func BenchmarkGetVersion_Query(b *testing.B) {
	v := &Versioning{Type: QueryVersion, Key: "version", DefaultVersion: "v1"}
	c := benchCtxQuery("version", "v2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.GetVersion(c)
	}
}

func BenchmarkGetVersion_Header(b *testing.B) {
	v := &Versioning{Type: HeaderVersion, Key: "X-Api-Version", DefaultVersion: "v1"}
	c := benchCtxHeader("X-Api-Version", "v2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.GetVersion(c)
	}
}

func BenchmarkGetVersion_Custom(b *testing.B) {
	v := &Versioning{
		Type:      CustomVersion,
		Extractor: func(c *ctx.HTTPContext) string { return "v9" },
	}
	c := benchCtxEmpty()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.GetVersion(c)
	}
}

func BenchmarkGetVersion_MediaType(b *testing.B) {
	v := &Versioning{Type: MediaType, Key: "v", DefaultVersion: "v1"}
	c := benchCtxHeader("Accept", "application/json;v=v2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.GetVersion(c)
	}
}
