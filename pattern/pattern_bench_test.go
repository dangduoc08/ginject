package pattern_test

import (
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/pattern"
)

func BenchmarkNewPattern_Exact(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = pattern.NewPattern("user.created")
	}
}

func BenchmarkNewPattern_SuffixWildcard(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = pattern.NewPattern("user.**")
	}
}

func BenchmarkNewPattern_Complex(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = pattern.NewPattern("tenant.*.user.**")
	}
}

func BenchmarkMatch_Exact(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("user.created")
	for b.Loop() {
		p.Match("user.created")
	}
}

func BenchmarkMatch_Global(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("**")
	for b.Loop() {
		p.Match("user.created")
	}
}

func BenchmarkMatch_SuffixWildcard_OneSegment(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("user.**")
	for b.Loop() {
		p.Match("user.created")
	}
}

func BenchmarkMatch_SuffixWildcard_ManySegments(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("user.**")
	for b.Loop() {
		p.Match("user.profile.settings.avatar.updated")
	}
}

func BenchmarkMatch_SuffixWildcard_Miss(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("user.**")
	for b.Loop() {
		p.Match("order.created")
	}
}

func BenchmarkMatch_Complex_3Seg(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("tenant.*.user.created")
	for b.Loop() {
		p.Match("tenant.abc.user.created")
	}
}

func BenchmarkMatch_Complex_MiddleAndTrailing(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("tenant.*.user.**")
	for b.Loop() {
		p.Match("tenant.abc.user.profile.updated")
	}
}

func BenchmarkMatch_SuffixWildcard_DeepTopic(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("a.b.c.d.**")
	for b.Loop() {
		p.Match("a.b.c.d.e.f.g.h")
	}
}

func BenchmarkMatch_Complex_LongHostileTopic(b *testing.B) {
	b.ReportAllocs()
	p := pattern.NewPattern("a.*.z")
	topic := strings.Repeat("seg.", 2500) + "end"
	for b.Loop() {
		p.Match(topic)
	}
}
