package ctx

import (
	"strings"
)

type Form map[string][]string

const (
	multipartFormData             = "multipart/form-data"
	applicationXWWWFormUrlencoded = "application/x-www-form-urlencoded"
	defaultMaxMemory              = 32 << 20
)

func (c *Context) Form() Form {
	if c.form != nil {
		return c.form
	}

	var e error
	contentType := c.Header().Get("Content-Type")

	if strings.Contains(contentType, multipartFormData) {
		e = c.Request.ParseMultipartForm(defaultMaxMemory)
	} else if strings.Contains(contentType, applicationXWWWFormUrlencoded) {
		e = c.Request.ParseForm()
	}

	if e != nil {
		panic(e)
	}

	c.form = Form(c.Request.Form)
	return c.form
}

func (f Form) Get(k string) string {
	fs := f[k]
	if len(fs) == 0 {
		return ""
	}
	return fs[0]
}

func (f Form) Set(k, v string) {
	f[k] = []string{v}
}

func (f Form) Add(k, v string) {
	f[k] = append(f[k], v)
}

func (f Form) Del(k string) {
	delete(f, k)
}

func (f Form) Has(k string) bool {
	_, ok := f[k]
	return ok
}

func (f Form) Bind(s any) (any, []FieldLevel) {
	return BindStrArr(f, &[]FieldLevel{}, s)
}
