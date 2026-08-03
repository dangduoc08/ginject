package pattern_test

import (
	"testing"

	"github.com/dangduoc08/ginject/pattern"
)

var sink bool

func BenchmarkNewPattern_Exact(b *testing.B) {
	for b.Loop() {
		_ = pattern.NewPattern("user.created")
	}
}

func BenchmarkNewPattern_SuffixWildcard(b *testing.B) {
	for b.Loop() {
		_ = pattern.NewPattern("user.*")
	}
}

func BenchmarkNewPattern_Complex(b *testing.B) {
	for b.Loop() {
		_ = pattern.NewPattern("tenant.*.user.*")
	}
}

func BenchmarkMatch_Exact(b *testing.B) {
	p := pattern.NewPattern("user.created")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("user.created")
	}
}

func BenchmarkMatch_Global(b *testing.B) {
	p := pattern.NewPattern("*")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("user.created")
	}
}

func BenchmarkMatch_SuffixWildcard_OneSegment(b *testing.B) {
	p := pattern.NewPattern("user.*")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("user.created")
	}
}

func BenchmarkMatch_SuffixWildcard_ManySegments(b *testing.B) {
	p := pattern.NewPattern("user.*")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("user.profile.settings.avatar.updated")
	}
}

func BenchmarkMatch_SuffixWildcard_Miss(b *testing.B) {
	p := pattern.NewPattern("user.*")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("order.created")
	}
}

func BenchmarkMatch_Complex_3Seg(b *testing.B) {
	p := pattern.NewPattern("tenant.*.user.created")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("tenant.abc.user.created")
	}
}

func BenchmarkMatch_Complex_MiddleAndTrailing(b *testing.B) {
	p := pattern.NewPattern("tenant.*.user.*")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("tenant.abc.user.profile.updated")
	}
}

func BenchmarkMatch_SuffixWildcard_DeepTopic(b *testing.B) {
	p := pattern.NewPattern("a.b.c.d.*")
	b.ResetTimer()
	for b.Loop() {
		sink = p.Match("a.b.c.d.e.f.g.h")
	}
}
