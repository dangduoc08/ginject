package routing

import (
	"fmt"
	"reflect"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/utils"
)

type Route struct {
	*Trie
	Hash               map[string][]context.Handler
	List               []string
	Middlewares        []context.Handler
	InjectableHandlers map[string]any
}

func NewRoute() *Route {
	return &Route{
		Trie:               NewTrie(),
		Hash:               make(map[string][]context.Handler),
		List:               []string{},
		Middlewares:        []context.Handler{},
		InjectableHandlers: make(map[string]any),
	}
}

func (r *Route) Add(route string, handlers ...context.Handler) *Route {
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
	if isStaticRoute(parsedRoute) {
		_, _, _, r.Hash[parsedRoute] = r.Trie.find(parsedRoute, '/')
	}

	return r
}

func (r *Route) AddInjectableHandler(route string, handler any) *Route {
	handlerKind := reflect.TypeOf(handler).Kind()
	if handler == nil || handlerKind != reflect.Func {
		panic(fmt.Errorf(
			utils.FmtRed(
				"%v is not a handler",
				handlerKind,
			),
		))
	}

	// push to trie
	// and mark as nil handler
	r.Add(route, nil)
	r.InjectableHandlers[route] = handler

	return r
}

func (r *Route) match(route string) (bool, string, map[string][]int, []string, []context.Handler) {
	if handlers, ok := r.Hash[route]; ok {
		return ok, route, nil, nil, handlers
	}

	i, paramKeys, paramVals, handlers := r.Trie.find(ToEndpoint(route), '/')
	matchedRoute := ""
	isMatched := false
	if i > -1 {
		isMatched = true
		matchedRoute = r.List[i]
	}

	return isMatched, matchedRoute, paramKeys, paramVals, handlers
}
