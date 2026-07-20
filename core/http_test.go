package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/routing"
)

func newTestHTTPContextForServe(urlPath string) (*ctx.HTTPContext, *httptest.ResponseRecorder) {
	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, urlPath, nil)
	w := httptest.NewRecorder()
	c.ResponseWriter = w
	return c, w
}

func TestServeContent_NonStringDirReturnsNotFound(t *testing.T) {
	h := newHTTP()
	c, w := newTestHTTPContextForServe("/files/a.txt")

	h.serveContent(c, 0, 42)

	if w.Code != http.StatusNotFound {
		t.Error(test.DiffMessage(w.Code, http.StatusNotFound, "serveContent with a non-string dir should return 404"))
	}
}

func TestServeContent_NoWildcard_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newHTTP()
	c, w := newTestHTTPContextForServe("/hello.txt")

	var fired bool
	c.Event.On(ctx.RequestFinished, func(args ...any) { fired = true })

	h.serveContent(c, 0, file)

	if w.Code != http.StatusOK {
		t.Error(test.DiffMessage(w.Code, http.StatusOK, "serveContent should serve an existing file with 200"))
	}
	if w.Body.String() != "hello" {
		t.Error(test.DiffMessage(w.Body.String(), "hello", "serveContent should write the file contents"))
	}
	if !fired {
		t.Error(test.DiffMessage(fired, true, "serveContent should emit RequestFinished after serving a file"))
	}
}

func TestServeContent_NoWildcard_MissingFileReturnsNotFound(t *testing.T) {
	h := newHTTP()
	c, w := newTestHTTPContextForServe("/missing.html")

	h.serveContent(c, 0, filepath.Join(t.TempDir(), "missing.html"))

	if w.Code != http.StatusNotFound {
		t.Error(test.DiffMessage(w.Code, http.StatusNotFound, "serveContent should 404 when the file does not exist"))
	}
}

func TestServeContent_Wildcard_ServesFileUnderDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "css", "app.css")
	if err := os.WriteFile(file, []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newHTTP()
	c, w := newTestHTTPContextForServe("/static/css/app.css")

	// route was registered as /static/*/, so the suffix starts at segment 2
	h.serveContent(c, 2, dir)

	if w.Code != http.StatusOK {
		t.Error(test.DiffMessage(w.Code, http.StatusOK, "serveContent should serve a file found under dir+suffix"))
	}
	if w.Body.String() != "body{}" {
		t.Error(test.DiffMessage(w.Body.String(), "body{}", "serveContent should write the resolved file's contents"))
	}
}

func TestServeContent_Wildcard_PathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("do-not-leak"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newHTTP()
	c, w := newTestHTTPContextForServe("/static/../secret.txt")

	h.serveContent(c, 2, subDir)

	if w.Code != http.StatusBadRequest {
		t.Error(test.DiffMessage(w.Code, http.StatusBadRequest, "serveContent must reject a path that escapes the served directory"))
	}
	if strings.Contains(w.Body.String(), "do-not-leak") {
		t.Error(test.DiffMessage(w.Body.String(), "<no leaked file contents>", "the traversal attempt must not leak the escaped file's contents"))
	}
}

func TestReturnNotFound(t *testing.T) {
	h := newHTTP()
	c, w := newTestHTTPContextForServe("/nope")

	h.returnNotFound(c)

	if w.Code != http.StatusNotFound {
		t.Error(test.DiffMessage(w.Code, http.StatusNotFound, "returnNotFound should respond with 404"))
	}
}

func TestReturnInvalidURL(t *testing.T) {
	h := newHTTP()
	c, w := newTestHTTPContextForServe("/bad")

	h.returnInvalidURL(c)

	if w.Code != http.StatusBadRequest {
		t.Error(test.DiffMessage(w.Code, http.StatusBadRequest, "returnInvalidURL should respond with 400"))
	}
}

func TestReturnDeprecatedURL(t *testing.T) {
	h := newHTTP()
	c, w := newTestHTTPContextForServe("/old")

	h.returnDeprecatedURL(c)

	if w.Code != http.StatusGone {
		t.Error(test.DiffMessage(w.Code, http.StatusGone, "returnDeprecatedURL should respond with 410"))
	}
}

func TestAddMainHandler_ServeRouteComputesWildcardIndex(t *testing.T) {
	h := newHTTP()
	h.addMainHandler(common.RESTLayer{
		Method:  routing.SERVE,
		Route:   "/static/*/",
		Handler: func() {},
	})

	pattern := routing.MethodRouteVersionToPattern(http.MethodGet, "/static/*/", "")
	idx, ok := h.lastWildcardSlashIndexByRoute[pattern]
	if !ok {
		t.Fatal("expected a wildcard slash index to be recorded for the SERVE route")
	}
	if idx != 2 {
		t.Error(test.DiffMessage(idx, 2, "wildcard slash index should be strings.Count(route, \"/\") - 1"))
	}
}

func TestAddMainHandler_NonWildcardServeRouteHasZeroIndex(t *testing.T) {
	h := newHTTP()
	h.addMainHandler(common.RESTLayer{
		Method:  routing.SERVE,
		Route:   "/static/logo.png",
		Handler: func() {},
	})

	pattern := routing.MethodRouteVersionToPattern(http.MethodGet, "/static/logo.png", "")
	idx, ok := h.lastWildcardSlashIndexByRoute[pattern]
	if !ok {
		t.Fatal("expected a wildcard slash index to be recorded for the SERVE route")
	}
	if idx != 0 {
		t.Error(test.DiffMessage(idx, 0, "a non-wildcard SERVE route should use config dir (index 0)"))
	}
}
