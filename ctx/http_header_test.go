package ctx

import (
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestHTTPContext_Header(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Foo", "bar")

	h := c.Header()
	if h.Get("X-Foo") != "bar" {
		t.Error(test.DiffMessage(h.Get("X-Foo"), "bar", "Header should wrap the request headers"))
	}
}

func TestHTTPContext_HeaderIsCached(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)

	h1 := c.Header()
	h1.Set("X-Foo", "mutated")
	h2 := c.Header()

	if h2.Get("X-Foo") != "mutated" {
		t.Error(test.DiffMessage(h2.Get("X-Foo"), "mutated", "Header should be cached across calls on the same context"))
	}
}

func TestHeader_GetMissingKey(t *testing.T) {
	h := Header{}
	if h.Get("missing") != "" {
		t.Error(test.DiffMessage(h.Get("missing"), "", "Get on missing key should return empty string"))
	}
}

func TestHeader_SetAddDelHas(t *testing.T) {
	h := Header{}
	h.Set("X-Foo", "1")
	if h.Get("X-Foo") != "1" {
		t.Error(test.DiffMessage(h.Get("X-Foo"), "1", "Set should store the value"))
	}

	h.Add("X-Foo", "2")
	if got := h["X-Foo"]; len(got) != 2 || got[1] != "2" {
		t.Error(test.DiffMessage(got, []string{"1", "2"}, "Add should append to the existing values"))
	}

	if !h.Has("X-Foo") {
		t.Error(test.DiffMessage(h.Has("X-Foo"), true, "Has should be true after Set"))
	}

	h.Del("X-Foo")
	if h.Has("X-Foo") {
		t.Error(test.DiffMessage(h.Has("X-Foo"), false, "Has should be false after Del"))
	}
}

func TestHeader_HasCanonicalizesKey(t *testing.T) {
	h := Header{}
	h.Set("x-foo", "1")
	if !h.Has("X-Foo") {
		t.Error(test.DiffMessage(h.Has("X-Foo"), true, "Has should canonicalize the header key before lookup"))
	}
}

type headerBindDTO struct {
	Auth string `bind:"Authorization"`
}

func TestHeader_Bind(t *testing.T) {
	h := Header{}
	h.Set("Authorization", "Bearer token")
	result, fls := h.Bind(headerBindDTO{})

	dto, ok := result.(headerBindDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, headerBindDTO{}, "Bind should return a headerBindDTO"))
	}
	if dto.Auth != "Bearer token" {
		t.Error(test.DiffMessage(dto.Auth, "Bearer token", "Bind should populate fields from header values"))
	}
	if len(fls) != 1 {
		t.Error(test.DiffMessage(len(fls), 1, "Bind should report a FieldLevel per bound field"))
	}
}
