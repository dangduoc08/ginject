package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

func TestNew(t *testing.T) {
	app := New()
	if app == nil {
		t.Fatal(testutils.DiffMessage(nil, "*App", "New should not return nil"))
		return
	}
	if app.http.route == nil {
		t.Error(testutils.DiffMessage(nil, "router", "route not initialized"))
	}
	if app.http.catchRESTFnsMap == nil {
		t.Error(testutils.DiffMessage(nil, "map", "catchRESTFnsMap not initialized"))
	}
	if app.Logger != nil {
		t.Error(testutils.DiffMessage(app.Logger, nil, "Logger should be nil before Create"))
	}
}

func TestGetContextIDFromHeader(t *testing.T) {
	app := New()
	c := ctx.NewContext()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(ctx.REQUEST_ID, "test-id-123")
	c.Request = r

	got := app.http.getContextID(c)
	if got != "test-id-123" {
		t.Error(testutils.DiffMessage(got, "test-id-123", "getContextID should use X-Request-Id header"))
	}
}

func TestGetContextIDGeneratesUUID(t *testing.T) {
	app := New()
	c1 := ctx.NewContext()
	c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c2 := ctx.NewContext()
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	id1 := app.http.getContextID(c1)
	id2 := app.http.getContextID(c2)

	if id1 == "" {
		t.Error(testutils.DiffMessage(id1, "non-empty UUID", "should generate UUID when header absent"))
	}
	if id1 == id2 {
		t.Error(testutils.DiffMessage(id1, "different UUID", "each call should produce a unique ID"))
	}
}
