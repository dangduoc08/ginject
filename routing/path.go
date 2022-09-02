package routing

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/dangduoc08/gooh/ds"
)

var HTTP_METHODS = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

var matchMethodReg *regexp.Regexp = regexp.MustCompile(strings.Join(ds.Map(HTTP_METHODS, func(elem string, index int) string {
	return ds.SLASH + ds.BACKSLASH + ds.OPEN_SQUARE_BRACKET + elem + ds.BACKSLASH + ds.CLOSE_SQUARE_BRACKET
}), "|"))

func handlePath(path string) string {
	return ds.AddAtEnd(ds.AddAtBegin(ds.RemoveSpace(path), ds.SLASH), ds.SLASH)
}

func handleMethod(method string) string {
	return ds.OPEN_SQUARE_BRACKET + method + ds.CLOSE_SQUARE_BRACKET
}

func handleMethodWithRoute(method, path string) string {
	return handleMethod(method) + handlePath(path)
}
