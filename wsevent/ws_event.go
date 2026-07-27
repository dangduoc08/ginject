package wsevent

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/matcher"
)

type WSEventItem struct {
	Middlewares []ctx.WSHandler
	Handler     any
}

type WSEvent struct {
	wsEventItemByPattern map[string]WSEventItem
	prefixByPrefix       map[string]string
	complexPatterns      []matcher.Pattern
	globalPattern        string
}

func NewWSEvent() *WSEvent {
	return &WSEvent{
		wsEventItemByPattern: make(map[string]WSEventItem),
		prefixByPrefix:       make(map[string]string),
	}
}

func (m *WSEvent) index(pattern string) {
	pat := matcher.Parse(pattern)
	switch pat.Kind() {
	case matcher.KindGlobal:
		m.globalPattern = pattern
	case matcher.KindSuffixWildcard:
		m.prefixByPrefix[pat.SimplePrefix()] = pattern
	case matcher.KindComplex:
		m.complexPatterns = append(m.complexPatterns, pat)
	}
}

func (m *WSEvent) Add(pattern string, value WSEventItem) {
	if _, existed := m.wsEventItemByPattern[pattern]; !existed {
		m.index(pattern)
	}
	m.wsEventItemByPattern[pattern] = value
}

func (m *WSEvent) AddMiddlewares(pattern string, middlewares ...ctx.WSHandler) {
	if _, existed := m.wsEventItemByPattern[pattern]; !existed {
		m.index(pattern)
	}
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

	if _, existed := m.wsEventItemByPattern[pattern]; !existed {
		m.index(pattern)
	}
	item := m.wsEventItemByPattern[pattern]
	item.Handler = handler
	m.wsEventItemByPattern[pattern] = item
}

func (m *WSEvent) Match(topic string) (WSEventItem, string, bool) {
	if item, ok := m.wsEventItemByPattern[topic]; ok && item.Handler != nil {
		return item, topic, true
	}

	for i := len(topic) - 1; i >= 0; i-- {
		if topic[i] != '.' {
			continue
		}
		if pattern, ok := m.prefixByPrefix[topic[:i]]; ok {
			if item := m.wsEventItemByPattern[pattern]; item.Handler != nil {
				return item, pattern, true
			}
		}
	}

	for _, pat := range m.complexPatterns {
		if matcher.Match(pat, topic) {
			if item := m.wsEventItemByPattern[pat.Raw()]; item.Handler != nil {
				return item, pat.Raw(), true
			}
		}
	}

	if m.globalPattern != "" {
		if item := m.wsEventItemByPattern[m.globalPattern]; item.Handler != nil {
			return item, m.globalPattern, true
		}
	}

	return WSEventItem{}, "", false
}
