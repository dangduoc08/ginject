package express

import "net/http"

type Request struct {
	*http.Request
	Params     map[string]string
	Middleware map[string]interface{}
}
