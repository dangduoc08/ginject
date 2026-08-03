package pattern

import "strings"

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
	if raw == "*" {
		return Pattern{raw: raw, kind: KindGlobal}
	}

	segs := strings.Split(raw, ".")

	hasWild := false
	for _, s := range segs {
		if s == "*" {
			hasWild = true
			break
		}
	}

	if !hasWild {
		return Pattern{raw: raw, segments: segs, kind: KindExact}
	}

	if len(segs) >= 2 && segs[len(segs)-1] == "*" {
		simple := true
		for _, s := range segs[:len(segs)-1] {
			if s == "*" {
				simple = false
				break
			}
		}
		if simple {
			return Pattern{
				raw:          raw,
				segments:     segs,
				kind:         KindSuffixWildcard,
				simplePrefix: strings.Join(segs[:len(segs)-1], "."),
			}
		}
	}

	return Pattern{raw: raw, segments: segs, kind: KindComplex}
}

func (p Pattern) Match(topic string) bool {
	switch p.kind {
	case KindGlobal:
		return true
	case KindExact:
		return p.raw == topic
	case KindSuffixWildcard:
		if !strings.HasPrefix(topic, p.simplePrefix+".") {
			return false
		}
		return len(topic) > len(p.simplePrefix)+1
	case KindComplex:
		return p.matchSegments(strings.Split(topic, "."))
	}
	return false
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

func (p Pattern) IsGlobal() bool {
	return p.kind == KindGlobal
}

func (p Pattern) matchSegments(topic []string) bool {
	pi, ti := 0, 0
	for pi < len(p.segments) {
		if p.segments[pi] == "*" {
			if pi == len(p.segments)-1 {
				return ti < len(topic)
			}
			if ti >= len(topic) {
				return false
			}
			pi++
			ti++
			continue
		}
		if ti >= len(topic) || p.segments[pi] != topic[ti] {
			return false
		}
		pi++
		ti++
	}
	return pi == len(p.segments) && ti == len(topic)
}
