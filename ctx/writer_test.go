package ctx

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestJSON_WriteData(t *testing.T) {
	w := httptest.NewRecorder()
	j := &JSON{responseWriter: w, data: []any{map[string]any{"foo": "bar"}}}
	j.WriteData(200)

	if w.Code != 200 {
		t.Error(test.DiffMessage(w.Code, 200, "JSON WriteData status code"))
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Error(test.DiffMessage(ct, "application/json", "JSON WriteData Content-Type"))
	}
	if body := w.Body.String(); !strings.Contains(body, `"foo":"bar"`) {
		t.Error(test.DiffMessage(body, `{"foo":"bar"}`, "JSON WriteData body"))
	}
}

func TestJSON_WriteDataPanicsOnUnmarshalableData(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when data cannot be marshaled to JSON")
		}
	}()
	w := httptest.NewRecorder()
	j := &JSON{responseWriter: w, data: []any{make(chan int)}}
	j.WriteData(200)
}

func TestJSONP_WriteData(t *testing.T) {
	w := httptest.NewRecorder()
	jp := &JSONP{responseWriter: w, data: []any{map[string]any{"foo": "bar"}}, callback: "cb"}
	jp.WriteData(200)

	if ct := w.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Error(test.DiffMessage(ct, "text/javascript; charset=utf-8", "JSONP WriteData Content-Type"))
	}
	body := w.Body.String()
	if !strings.Contains(body, "cb(") || !strings.Contains(body, `"foo":"bar"`) {
		t.Error(test.DiffMessage(body, "cb({\"foo\":\"bar\"});", "JSONP WriteData body"))
	}
}

func TestJSONP_WriteDataPanicsOnUnmarshalableData(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when data cannot be marshaled to JSON")
		}
	}()
	w := httptest.NewRecorder()
	jp := &JSONP{responseWriter: w, data: []any{make(chan int)}, callback: "cb"}
	jp.WriteData(200)
}

func TestText_WriteDataNoArgs(t *testing.T) {
	w := httptest.NewRecorder()
	tx := &Text{responseWriter: w, data: "hello world", args: []any{}}
	tx.WriteData(200)

	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Error(test.DiffMessage(ct, "text/plain; charset=utf-8", "Text WriteData Content-Type"))
	}
	if body := w.Body.String(); body != "hello world" {
		t.Error(test.DiffMessage(body, "hello world", "Text WriteData body without args"))
	}
}

func TestText_WriteDataWithArgs(t *testing.T) {
	w := httptest.NewRecorder()
	tx := &Text{responseWriter: w, data: "hello %s, you are %d", args: []any{"joe", 30}}
	tx.WriteData(200)

	if body := w.Body.String(); body != "hello joe, you are 30" {
		t.Error(test.DiffMessage(body, "hello joe, you are 30", "Text WriteData body with args"))
	}
}
