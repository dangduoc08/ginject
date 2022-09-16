package routing

import "github.com/dangduoc08/gooh/ctx"

type cache struct {
	matchedRoute string
	paramKeys    map[string][]int
	paramVals    []string
	handlers     []ctx.Handler
}
