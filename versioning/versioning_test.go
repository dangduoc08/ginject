package versioning

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

func makeCtxWithQuery(key, val string) *ctx.Context {
	c := ctx.NewContext()
	u := &url.URL{RawQuery: url.Values{key: {val}}.Encode()}
	c.Request = &http.Request{URL: u}
	return c
}

func makeCtxWithHeader(key, val string) *ctx.Context {
	c := ctx.NewContext()
	h := http.Header{key: {val}}
	c.Request = &http.Request{Header: h}
	return c
}

func makeCtxEmpty() *ctx.Context {
	c := ctx.NewContext()
	u := &url.URL{RawQuery: ""}
	c.Request = &http.Request{URL: u, Header: http.Header{}}
	return c
}

func TestGetVersion_Query(t *testing.T) {
	v := &Versioning{Type: QUERY, Key: "version", DefaultVersion: "v1"}

	got := v.GetVersion(makeCtxWithQuery("version", "v2"))
	if got != "v2" {
		t.Error(testutils.DiffMessage(got, "v2", "QUERY: key present"))
	}

	got = v.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(testutils.DiffMessage(got, "v1", "QUERY: key absent uses default"))
	}
}

func TestGetVersion_QueryDefaultKey(t *testing.T) {
	v := &Versioning{Type: QUERY, DefaultVersion: "v3"}

	got := v.GetVersion(makeCtxWithQuery("v", "v5"))
	if got != "v5" {
		t.Error(testutils.DiffMessage(got, "v5", "QUERY: default key 'v'"))
	}
}

func TestGetVersion_Header(t *testing.T) {
	v := &Versioning{Type: HEADER, Key: "X-Api-Version", DefaultVersion: "v1"}

	got := v.GetVersion(makeCtxWithHeader("X-Api-Version", "v2"))
	if got != "v2" {
		t.Error(testutils.DiffMessage(got, "v2", "HEADER: key present"))
	}

	got = v.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(testutils.DiffMessage(got, "v1", "HEADER: key absent uses default"))
	}
}

func TestGetVersion_Custom(t *testing.T) {
	v := &Versioning{
		Type:           CUSTOM,
		DefaultVersion: "v1",
		Extractor:      func(c *ctx.Context) string { return "v9" },
	}

	got := v.GetVersion(makeCtxEmpty())
	if got != "v9" {
		t.Error(testutils.DiffMessage(got, "v9", "CUSTOM: extractor called"))
	}

	v2 := &Versioning{Type: CUSTOM, DefaultVersion: "v1"}
	got = v2.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(testutils.DiffMessage(got, "v1", "CUSTOM: nil extractor uses default"))
	}
}

func TestGetVersion_MediaType(t *testing.T) {
	v := &Versioning{Type: MEDIA_TYPE, Key: "v", DefaultVersion: "v1"}

	got := v.GetVersion(makeCtxWithHeader("Accept", "application/json;v=v2"))
	if got != "v2" {
		t.Error(testutils.DiffMessage(got, "v2", "MEDIA_TYPE: v= present in Accept"))
	}

	got = v.GetVersion(makeCtxWithHeader("Accept", "application/json"))
	if got != "v1" {
		t.Error(testutils.DiffMessage(got, "v1", "MEDIA_TYPE: no v= uses default"))
	}

	got = v.GetVersion(makeCtxEmpty())
	if got != "v1" {
		t.Error(testutils.DiffMessage(got, "v1", "MEDIA_TYPE: no Accept header uses default"))
	}
}

func TestGetVersion_Neutral(t *testing.T) {
	v := &Versioning{Type: QUERY, Key: "v", DefaultVersion: NEUTRAL_VERSION}

	got := v.GetVersion(makeCtxEmpty())
	if got != NEUTRAL_VERSION {
		t.Error(testutils.DiffMessage(got, NEUTRAL_VERSION, "default version can be NEUTRAL"))
	}
}
