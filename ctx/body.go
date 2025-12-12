package ctx

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/dangduoc08/ginject/utils"
)

type Body map[string]any

const (
	applicationJSON = "application/json"
)

func (c *Context) Body() Body {
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
	keys = utils.ArrFilter[string](keys, func(el string, i int) bool {
		return strings.TrimSpace(el) != ""
	})
	obj := b
	if len(keys) == 0 {
		return
	}

	for i, key := range keys {
		if val, ok := obj[key]; ok {
			if deeperObj, ok := val.(map[string]any); ok {
				obj = deeperObj
				if i == len(keys)-1 {
					obj[key] = v
					return
				}
			} else if i == len(keys)-1 {
				obj[key] = v
				return
			}
		} else {
			if i == len(keys)-1 {
				obj[key] = v
				return
			} else {
				obj[key] = make(map[string]any)
				obj = obj[key].(map[string]any)
			}
		}
	}
}

func (b Body) Get(k string) any {
	keys := strings.Split(k, ".")
	keys = utils.ArrFilter(keys, func(el string, i int) bool {
		return strings.TrimSpace(el) != ""
	})
	obj := b
	if len(keys) == 0 {
		return obj
	}

	for i, key := range keys {
		if val, ok := obj[key]; ok {
			if deeperObj, ok := val.(map[string]any); ok {
				obj = deeperObj
				if i == len(keys)-1 {
					return obj
				}
			} else if i == len(keys)-1 {
				return obj[key]
			} else {
				return nil
			}
		}
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
