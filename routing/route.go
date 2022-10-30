package routing

import (
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/utils"
)

type Route struct {
	*Trie
	List        []string
	Middlewares []ctx.Handler
}

func NewRoute() *Route {
	return &Route{
		Trie:        NewTrie(),
		List:        []string{},
		Middlewares: []ctx.Handler{},
	}
}

func (r *Route) Add(route string, handlers ...ctx.Handler) *Route {
	endpoint := utils.StrRemoveDup(ToEndpoint(route), "*")
	i := utils.ArrFindIndex(r.List, func(route string, i int) bool {
		return route == endpoint
	})
	if i < 0 {
		r.List = append(r.List, endpoint)
		i = len(r.List) - 1

		// add global middleware to node
		handlers = append(r.Middlewares, handlers...)
	}
	parsedRoute, paramKey := parseToParamKey(endpoint)

	r.Trie.insert(parsedRoute, '/', i, paramKey, handlers)

	return r
}

func (r *Route) match(route string) (bool, string, map[string][]int, []string, []ctx.Handler) {
	i, paramKeys, paramVals, handlers := r.Trie.find(ToEndpoint(route), '/')
	matchedRoute := ""
	isMatched := false
	if i > -1 {
		isMatched = true
		matchedRoute = r.List[i]
	}

	return isMatched, matchedRoute, paramKeys, paramVals, handlers
}
