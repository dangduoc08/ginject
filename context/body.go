package context

import (
	"encoding/json"
	"io"
	"strings"
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

func (b Body) Bind(s any) any {
	return BindStruct(b, s)
}
