package common

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

type fnTestController struct{}

func (fnTestController) READ_users()    {}
func (fnTestController) CREATE_orders() {}

func TestGetFnName(t *testing.T) {
	cases := []struct {
		handler any
		want    string
	}{
		{fnTestController{}.READ_users, "READ_users"},
		{fnTestController{}.CREATE_orders, "CREATE_orders"},
	}
	for _, c := range cases {
		got := GetFuncName(c.handler)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "GetFuncName"))
		}
	}
}

func TestToWSEventName(t *testing.T) {
	cases := []struct {
		s, want string
	}{
		{"/message/", "message"},
		{"/room/events/", "room/events"},
		{"status", "status"},
		{"/nested/deep/", "nested/deep"},
		{"/event/", "event"},
	}
	for _, c := range cases {
		got := ToWSEventName(c.s)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "ToWSEventName"))
		}
	}
}

// TestParseFnNameToURL_AllHTTPMethods verifies every REST operation maps to
// the correct HTTP method.
func TestParseFnNameToURL_AllHTTPMethods(t *testing.T) {
	cases := []struct {
		fn, wantMethod, wantRoute string
	}{
		{"READ_health", "GET", "/health/"},
		{"CREATE_users", "POST", "/users/"},
		{"UPDATE_users", "PUT", "/users/"},
		{"MODIFY_users", "PATCH", "/users/"},
		{"DELETE_users", "DELETE", "/users/"},
		{"PREFLIGHT_health", "OPTIONS", "/health/"},
	}
	for _, c := range cases {
		method, route, version := ParseFuncNameToURL(c.fn)
		if method != c.wantMethod {
			t.Error(test.DiffMessage(method, c.wantMethod, c.fn+" method"))
		}
		if route != c.wantRoute {
			t.Error(test.DiffMessage(route, c.wantRoute, c.fn+" route"))
		}
		if version != "" {
			t.Error(test.DiffMessage(version, "", c.fn+" version"))
		}
	}
}

// TestParseFnNameToURL_InvalidInput verifies that unrecognised or empty inputs
// do not produce method output and do not panic.
func TestParseFnNameToURL_InvalidInput(t *testing.T) {
	cases := []struct {
		fn, wantMethod string
	}{
		{"INVALID_users", ""},
		{"lowercase_users", ""},
		{"", ""},
	}
	for _, c := range cases {
		method, _, _ := ParseFuncNameToURL(c.fn)
		if method != c.wantMethod {
			t.Error(test.DiffMessage(method, c.wantMethod, c.fn+" method should be empty"))
		}
	}
}

// TestParseFnNameToURL_VersionExtraction verifies version tokens are captured correctly.
func TestParseFnNameToURL_VersionExtraction(t *testing.T) {
	cases := []struct {
		fn, wantRoute, wantVersion string
	}{
		{"READ_users_VERSION_v1", "/users/", "v1"},
		{"READ_users_VERSION_V_12", "/users/", "V_12"},
		// trailing underscores in VERSION are filtered as empty segments
		{"READ_users_VERSION_", "/users/", ""},
		// version with no tokens after it
		{"READ_users_VERSION", "/users/", ""},
	}
	for _, c := range cases {
		_, route, version := ParseFuncNameToURL(c.fn)
		if route != c.wantRoute {
			t.Error(test.DiffMessage(route, c.wantRoute, c.fn+" route"))
		}
		if version != c.wantVersion {
			t.Error(test.DiffMessage(version, c.wantVersion, c.fn+" version"))
		}
	}
}

// TestParseFnNameToURL_BareOperation verifies that a bare operation with no
// path tokens produces a clean single-slash root route.
func TestParseFnNameToURL_BareOperation(t *testing.T) {
	cases := []struct {
		fn, wantRoute string
	}{
		{"READ", "/"},
		{"CREATE", "/"},
		{"READ_VERSION_v1", "/"},
	}
	for _, c := range cases {
		_, route, _ := ParseFuncNameToURL(c.fn)
		if route != c.wantRoute {
			t.Error(test.DiffMessage(route, c.wantRoute, c.fn+" route"))
		}
	}
}

// TestParseFnNameToURL_ParamWithoutPath verifies that BY immediately after an
// operation (no resource name) produces a clean route with no double slash.
func TestParseFnNameToURL_ParamWithoutPath(t *testing.T) {
	_, route, _ := ParseFuncNameToURL("READ_BY_id")
	want := "/{id}/"
	if route != want {
		t.Error(test.DiffMessage(route, want, "READ_BY_id route"))
	}
}

