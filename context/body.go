package context

import (
	"encoding/json"
	"io"
	"strings"
)

type Body map[string]any

const (
	applicationJSON               = "application/json"
	multipartFormData             = "multipart/form-data"
	applicationXWWWFormUrlencoded = "application/x-www-form-urlencoded"
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
	} else if strings.Contains(contentType, applicationXWWWFormUrlencoded) {
		c.Request.ParseForm()
		for key, values := range c.Request.Form {
			if len(values) == 1 {
				c.body[key] = values[0]
			} else {
				c.body[key] = values
			}
		}
	} else if strings.Contains(contentType, multipartFormData) {
		c.Request.ParseMultipartForm(32 << 20)
		for key, values := range c.Request.Form {
			if len(values) == 1 {
				c.body[key] = values[0]
			} else {
				c.body[key] = values
			}
		}
	}

	return c.body
}
