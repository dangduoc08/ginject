package wsevent

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/ds"
	"github.com/dangduoc08/ginject/internal/str"
)

type WSEventItem struct {
	Middlewares []ctx.Handler
	Handler     any
}

type WSEvent struct {
	trie                 *ds.Trie
	wsEventItemByPattern map[string]WSEventItem
}

func NewWSEvent() *WSEvent {
	return &WSEvent{
		trie:                 ds.NewTrie(),
		wsEventItemByPattern: make(map[string]WSEventItem),
	}
}

func (m *WSEvent) Add(pattern string, value WSEventItem) {
	m.trie.Insert(pattern, str.Enclose(pattern, '.'), '.')
	m.wsEventItemByPattern[pattern] = value
}

func (m *WSEvent) AddMiddlewares(pattern string, middlewares ...ctx.Handler) {
	item := m.wsEventItemByPattern[pattern]
	item.Middlewares = append(item.Middlewares, middlewares...)
	m.wsEventItemByPattern[pattern] = item
}

func (m *WSEvent) AddInjectableHandler(pattern string, handler any) {
	var handlerKind reflect.Kind
	if handler != nil {
		handlerKind = reflect.TypeOf(handler).Kind()
	}
	if handler == nil || handlerKind != reflect.Func {
		panic(errors.New(
			color.FmtRed(
				"invalid handler: %v is not a handler",
				handlerKind,
			),
		))
	}

	item := m.wsEventItemByPattern[pattern]
	item.Handler = handler
	m.wsEventItemByPattern[pattern] = item

	m.trie.Insert(pattern, str.Enclose(pattern, '.'), '.')
}

func (m *WSEvent) Match(topic string) (value WSEventItem, pattern string, ok bool) {
	matchedRaw, wildcardRaw, _ := m.trie.Find(str.Enclose(topic, '.'), '.', false)

	raw := matchedRaw
	if raw == "" {
		raw = wildcardRaw
	}
	if raw == "" {
		return
	}

	v, found := m.wsEventItemByPattern[raw]
	return v, raw, found
}
