package ctx

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/dangduoc08/ginject/internal/slice"
)

type Body map[string]any

const (
	applicationJSON = "application/json"
)

func (c *HTTPContext) Body() Body {
	if c.body != nil {
		return c.body
	}

	c.body = make(Body)
	contentType := c.Header().Get("Content-Type")
	if strings.Contains(contentType, applicationJSON) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(body, &c.body)
		if err != nil {
			panic(err)
		}
	}

	return c.body
}

func (b Body) Set(k string, v any) {
	keys := strings.Split(k, ".")
	keys = slice.Filter[string](keys, func(el string, i int) bool {
		return strings.TrimSpace(el) != ""
	})
	if len(keys) == 0 {
		return
	}

	obj := b
	for i, key := range keys {
		if i == len(keys)-1 {
			obj[key] = v
			return
		}

		deeperObj, ok := obj[key].(map[string]any)
		if !ok {
			deeperObj = make(map[string]any)
			obj[key] = deeperObj
		}
		obj = deeperObj
	}
}

func (b Body) Get(k string) any {
	keys := strings.Split(k, ".")
	keys = slice.Filter(keys, func(el string, i int) bool {
		return strings.TrimSpace(el) != ""
	})
	if len(keys) == 0 {
		return b
	}

	obj := b
	for i, key := range keys {
		val, ok := obj[key]
		if !ok {
			return nil
		}
		if i == len(keys)-1 {
			return val
		}

		deeperObj, ok := val.(map[string]any)
		if !ok {
			return nil
		}
		obj = deeperObj
	}

	return nil
}

func (b Body) Del(k string) {
	delete(b, k)
}

func (b Body) Has(k string) bool {
	return b.Get(k) != nil
}

func (b Body) Bind(s any) (any, []FieldLevel) {
	return BindStruct(b, &[]FieldLevel{}, s, "", "")
}
