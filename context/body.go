package context

import (
	"encoding/json"
	"fmt"
	"io"
)

type Body map[string][]string

func (c *Context) Body() Body {
	ch := make(chan map[string][]string)

	go func(ch chan map[string][]string) {
		fmt.Println("form")
		c.Request.ParseMultipartForm(1024)

		ch <- c.Request.Form

	}(ch)

	go func(ch chan map[string][]string) {
		fmt.Println("urlencoded")
		c.Request.ParseForm()

		ch <- c.Request.Form

	}(ch)

	go func(ch chan map[string][]string) {
		fmt.Println("json")
		body, _ := io.ReadAll(c.Request.Body)
		json.Marshal(body)
		fmt.Println("json", string(body))

	}(ch)

	c.body = <-ch

	fmt.Println("c.body", c.body)

	return c.body
}
