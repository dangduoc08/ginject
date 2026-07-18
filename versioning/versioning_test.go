package versioning

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func makeCtxWithQuery(key, val string) *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	u := &url.URL{RawQuery: url.Values{key: {val}}.Encode()}
	c.Request = &http.Request{URL: u}
	return c
}

func makeCtxWithHeader(key, val string) *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	h := http.Header{key: {val}}
	c.Request = &http.Request{Header: h}
	return c
}

func makeCtxEmpty() *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	u := &url.URL{RawQuery: ""}
	c.Request = &http.Request{URL: u, Header: http.Header{}}
	return c
}

func TestGetVersion_Query(t *testing.T) {
	v := &Versioning{Type: QueryVersion, Key: "version", DefaultVersion: "v1"}

	got := v.GetVersion(makeCtxWithQuery("version", "v2"))
	if got != "v2" {
		t.Error(test.DiffMessage(got, "v2", "QueryVersion: key present"))
	}

	got = v.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(test.DiffMessage(got, "v1", "QueryVersion: key absent uses default"))
	}
}

func TestGetVersion_QueryDefaultKey(t *testing.T) {
	v := &Versioning{Type: QueryVersion, Key: "v", DefaultVersion: "v3"}

	got := v.GetVersion(makeCtxWithQuery("v", "v5"))
	if got != "v5" {
		t.Error(test.DiffMessage(got, "v5", "QueryVersion: default key 'v'"))
	}
}

func TestGetVersion_Header(t *testing.T) {
	v := &Versioning{Type: HeaderVersion, Key: "X-Api-Version", DefaultVersion: "v1"}

	got := v.GetVersion(makeCtxWithHeader("X-Api-Version", "v2"))
	if got != "v2" {
		t.Error(test.DiffMessage(got, "v2", "HeaderVersion: key present"))
	}

	got = v.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(test.DiffMessage(got, "v1", "HeaderVersion: key absent uses default"))
	}
}

func TestGetVersion_Custom(t *testing.T) {
	v := &Versioning{
		Type:           CustomVersion,
		DefaultVersion: "v1",
		Extractor:      func(c *ctx.HTTPContext) string { return "v9" },
	}

	got := v.GetVersion(makeCtxEmpty())
	if got != "v9" {
		t.Error(test.DiffMessage(got, "v9", "CustomVersion: extractor called"))
	}

	v2 := &Versioning{Type: CustomVersion, DefaultVersion: "v1"}
	got = v2.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(test.DiffMessage(got, "v1", "CustomVersion: nil extractor uses default"))
	}
}

func TestGetVersion_MediaType(t *testing.T) {
	v := &Versioning{Type: MediaType, Key: "v", DefaultVersion: "v1"}

	got := v.GetVersion(makeCtxWithHeader("Accept", "application/json;v=v2"))
	if got != "v2" {
		t.Error(test.DiffMessage(got, "v2", "MediaType: v= present in Accept"))
	}

	got = v.GetVersion(makeCtxWithHeader("Accept", "application/json"))
	if got != "v1" {
		t.Error(test.DiffMessage(got, "v1", "MediaType: no v= uses default"))
	}

	got = v.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(test.DiffMessage(got, "v1", "MediaType: no Accept header uses default"))
	}
}

func TestGetVersion_Neutral(t *testing.T) {
	v := &Versioning{Type: QueryVersion, Key: "v", DefaultVersion: NeutralVersion}

	got := v.GetVersion(makeCtxEmpty())
	if got != NeutralVersion {
		t.Error(test.DiffMessage(got, NeutralVersion, "default version can be NEUTRAL"))
	}
}
