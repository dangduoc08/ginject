package ctx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func newTestHTTPContext() *HTTPContext {
	return NewHTTPContext()
}

func TestNewHTTPContext_DefaultCode(t *testing.T) {
	c := NewHTTPContext()
	if c.Code != http.StatusOK {
		t.Error(test.DiffMessage(c.Code, http.StatusOK, "NewHTTPContext default Code"))
	}
}

func TestSetID_IgnoresRequestIDHeader(t *testing.T) {
	c := newTestHTTPContext()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(RequestID, "test-request-id")
	c.Init(httptest.NewRecorder(), r)
	if c.id == "test-request-id" {
		t.Error(test.DiffMessage(c.id, "<generated UUID, not the header value>", "SetID must always generate its own ID and ignore the X-Request-Id header"))
	}
	if c.id == "" {
		t.Error(test.DiffMessage(c.id, "<non-empty UUID>", "SetID must still generate an ID when a header is present"))
	}
}

func TestSetID_GeneratedWhenNoHeader(t *testing.T) {
	c := newTestHTTPContext()
	r := httptest.NewRequest("GET", "/", nil)
	c.Init(httptest.NewRecorder(), r)
	if c.id == "" {
		t.Error(test.DiffMessage(c.id, "<non-empty UUID>", "SetID generates UUID"))
	}
}

func TestSetID_Idempotent(t *testing.T) {
	c := newTestHTTPContext()
	r := httptest.NewRequest("GET", "/", nil)
	c.Init(httptest.NewRecorder(), r)
	first := c.id
	c.SetID()
	if c.id != first {
		t.Error(test.DiffMessage(c.id, first, "SetID idempotent"))
	}
}

func TestGetID(t *testing.T) {
	c := newTestHTTPContext()
	r := httptest.NewRequest("GET", "/", nil)
	c.Init(httptest.NewRecorder(), r)
	if c.GetID() == "" {
		t.Error(test.DiffMessage(c.GetID(), "<non-empty UUID>", "GetID"))
	}
	if c.GetID() != c.id {
		t.Error(test.DiffMessage(c.GetID(), c.id, "GetID must return the same ID stored on the context"))
	}
}

func TestHTTPContext_Status(t *testing.T) {
	c := newTestHTTPContext()
	ret := c.Status(http.StatusNotFound)
	if c.Code != http.StatusNotFound {
		t.Error(test.DiffMessage(c.Code, http.StatusNotFound, "Status sets Code"))
	}
	if ret != c {
		t.Error(test.DiffMessage(ret, c, "Status returns self"))
	}
}

func TestReset_ClearsAllFields(t *testing.T) {
	c := newTestHTTPContext()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set(RequestID, "some-id")
	w := httptest.NewRecorder()
	c.Init(w, r)
	c.Status(http.StatusNotFound)
	_ = c.Query()
	_ = c.Header()
	c.ParamKeys = map[string][]int{"id": {0}}
	c.ParamValues = []string{"1"}
	_ = c.Param()
	c.Next = func() {}

	c.Reset()

	if c.Code != http.StatusOK {
		t.Error(test.DiffMessage(c.Code, http.StatusOK, "Reset Code"))
	}
	if c.id != "" {
		t.Error(test.DiffMessage(c.id, "", "Reset ID"))
	}
	if c.Request != nil {
		t.Error(test.DiffMessage(c.Request, nil, "Reset Request"))
	}
	if c.ResponseWriter != nil {
		t.Error(test.DiffMessage(c.ResponseWriter, nil, "Reset ResponseWriter"))
	}
	if c.body != nil {
		t.Error(test.DiffMessage(c.body, nil, "Reset body"))
	}
	if c.query != nil {
		t.Error(test.DiffMessage(c.query, nil, "Reset query"))
	}
	if c.header != nil {
		t.Error(test.DiffMessage(c.header, nil, "Reset header"))
	}
	if c.param != nil {
		t.Error(test.DiffMessage(c.param, nil, "Reset param"))
	}
	if c.ParamKeys != nil {
		t.Error(test.DiffMessage(c.ParamKeys, nil, "Reset ParamKeys"))
	}
	if c.ParamValues != nil {
		t.Error(test.DiffMessage(c.ParamValues, nil, "Reset ParamValues"))
	}
	if c.Next != nil {
		t.Error(test.DiffMessage(c.Next, nil, "Reset Next"))
	}
}

func TestHTTPContext_Text(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w

	c.Text("hello %s", "joe")

	if w.Body.String() != "hello joe" {
		t.Error(test.DiffMessage(w.Body.String(), "hello joe", "Text should write the formatted text to the response"))
	}
	if w.Code != http.StatusOK {
		t.Error(test.DiffMessage(w.Code, http.StatusOK, "Text should write the context's status code"))
	}
}

func TestHTTPContext_JSON(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w

	c.JSON(map[string]any{"foo": "bar"})

	if !strings.Contains(w.Body.String(), `"foo":"bar"`) {
		t.Error(test.DiffMessage(w.Body.String(), `{"foo":"bar"}`, "JSON should write the marshaled data to the response"))
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Error(test.DiffMessage(ct, "application/json", "JSON should set the Content-Type header"))
	}
}

func TestHTTPContext_JSONP(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/?callback=cb", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w

	c.JSONP(map[string]any{"foo": "bar"})

	if !strings.Contains(w.Body.String(), "cb(") {
		t.Error(test.DiffMessage(w.Body.String(), "cb({...});", "JSONP should wrap the JSON body in the callback"))
	}
}

func TestHTTPContext_JSONPFallsBackToJSONWithoutCallback(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w

	c.JSONP(map[string]any{"foo": "bar"})

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Error(test.DiffMessage(ct, "application/json", "JSONP without a callback query param should fall back to plain JSON"))
	}
}

func TestHTTPContext_Redirect(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w

	c.Redirect("/new-location")

	if w.Code != http.StatusMovedPermanently {
		t.Error(test.DiffMessage(w.Code, http.StatusMovedPermanently, "Redirect should respond with 301"))
	}
	if loc := w.Header().Get("Location"); loc != "/new-location" {
		t.Error(test.DiffMessage(loc, "/new-location", "Redirect should set the Location header"))
	}
}
