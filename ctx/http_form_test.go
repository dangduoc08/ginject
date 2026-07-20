package ctx

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestHTTPContext_FormURLEncoded(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader("foo=bar"))
	c.Request.Header.Set("Content-Type", applicationXWWWFormUrlencoded)

	f := c.Form()
	if f.Get("foo") != "bar" {
		t.Error(test.DiffMessage(f.Get("foo"), "bar", "Form should parse a urlencoded body"))
	}
}

func TestHTTPContext_FormMultipart(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("foo", "bar"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", &buf)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())

	f := c.Form()
	if f.Get("foo") != "bar" {
		t.Error(test.DiffMessage(f.Get("foo"), "bar", "Form should parse a multipart body"))
	}
}

func TestHTTPContext_FormNoContentType(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)

	f := c.Form()
	if len(f) != 0 {
		t.Error(test.DiffMessage(f, Form{}, "Form should be empty when the content type is neither urlencoded nor multipart"))
	}
}

func TestHTTPContext_FormIsCached(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader("foo=bar"))
	c.Request.Header.Set("Content-Type", applicationXWWWFormUrlencoded)

	f1 := c.Form()
	f1.Set("foo", "mutated")
	f2 := c.Form()

	if f2.Get("foo") != "mutated" {
		t.Error(test.DiffMessage(f2.Get("foo"), "mutated", "Form should be cached across calls on the same context"))
	}
}

func TestForm_GetMissingKey(t *testing.T) {
	f := Form{}
	if f.Get("missing") != "" {
		t.Error(test.DiffMessage(f.Get("missing"), "", "Get on missing key should return empty string"))
	}
}

func TestForm_SetAddDelHas(t *testing.T) {
	f := Form{}
	f.Set("a", "1")
	if f.Get("a") != "1" {
		t.Error(test.DiffMessage(f.Get("a"), "1", "Set should store the value"))
	}

	f.Add("a", "2")
	if got := f["a"]; len(got) != 2 || got[1] != "2" {
		t.Error(test.DiffMessage(got, []string{"1", "2"}, "Add should append to the existing values"))
	}

	if !f.Has("a") {
		t.Error(test.DiffMessage(f.Has("a"), true, "Has should be true after Set"))
	}

	f.Del("a")
	if f.Has("a") {
		t.Error(test.DiffMessage(f.Has("a"), false, "Has should be false after Del"))
	}
}

type formBindDTO struct {
	Name string `bind:"name"`
}

func TestForm_Bind(t *testing.T) {
	f := Form{"name": {"joe"}}
	result, fls := f.Bind(formBindDTO{})

	dto, ok := result.(formBindDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, formBindDTO{}, "Bind should return a formBindDTO"))
	}
	if dto.Name != "joe" {
		t.Error(test.DiffMessage(dto.Name, "joe", "Bind should populate fields from form values"))
	}
	if len(fls) != 1 {
		t.Error(test.DiffMessage(len(fls), 1, "Bind should report a FieldLevel per bound field"))
	}
}
