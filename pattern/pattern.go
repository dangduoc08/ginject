package pattern

import (
	"slices"
	"strings"
)

const (
	singleWildcard = "*"
	multiWildcard  = "**"
)

type Kind uint8

const (
	KindExact Kind = iota
	KindGlobal
	KindSuffixWildcard
	KindComplex
)

type Pattern struct {
	raw          string
	segments     []string
	kind         Kind
	simplePrefix string
}

func NewPattern(raw string) Pattern {
	if raw == multiWildcard {
		return Pattern{raw: raw, kind: KindGlobal}
	}

	segs := strings.Split(raw, ".")
	last := len(segs) - 1

	hasWild := false
	for i, s := range segs {
		if s == singleWildcard || (s == multiWildcard && i == last) {
			hasWild = true
			break
		}
	}

	if !hasWild {
		return Pattern{raw: raw, segments: segs, kind: KindExact}
	}

	if last >= 1 && segs[last] == multiWildcard && !slices.ContainsFunc(segs[:last], isWildcard) {
		return Pattern{
			raw:          raw,
			segments:     segs,
			kind:         KindSuffixWildcard,
			simplePrefix: strings.Join(segs[:last], "."),
		}
	}

	return Pattern{raw: raw, segments: segs, kind: KindComplex}
}

func isWildcard(segment string) bool {
	return segment == singleWildcard || segment == multiWildcard
}

func (p Pattern) Match(topic string) bool {
	switch p.kind {
	case KindGlobal:
		return true
	case KindExact:
		return p.raw == topic
	case KindSuffixWildcard:
		prefixLen := len(p.simplePrefix)
		return len(topic) > prefixLen+1 && topic[prefixLen] == '.' && topic[:prefixLen] == p.simplePrefix
	default:
		return p.matchSegments(topic)
	}
}

func (p Pattern) Raw() string {
	return p.raw
}

func (p Pattern) Kind() Kind {
	return p.kind
}

func (p Pattern) SimplePrefix() string {
	return p.simplePrefix
}

func (p Pattern) IsExact() bool {
	return p.kind == KindExact
}

func (p Pattern) matchSegments(topic string) bool {
	rest := topic
	exhausted := false
	last := len(p.segments) - 1
	for pi, seg := range p.segments {
		if exhausted {
			return false
		}
		if seg == multiWildcard && pi == last {
			return true
		}
		current, next, found := strings.Cut(rest, ".")
		if seg != singleWildcard && seg != current {
			return false
		}
		rest = next
		exhausted = !found
	}
	return exhausted
}
