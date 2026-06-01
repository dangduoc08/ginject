package matcher_test

import (
	"testing"

	"github.com/dangduoc08/ginject/matcher"
)

var sink bool

func BenchmarkParse_Exact(b *testing.B) {
	for b.Loop() {
		_ = matcher.Parse("user.created")
	}
}

func BenchmarkParse_SingleSuffix(b *testing.B) {
	for b.Loop() {
		_ = matcher.Parse("user.*")
	}
}

func BenchmarkParse_Complex(b *testing.B) {
	for b.Loop() {
		_ = matcher.Parse("tenant.*.user.>")
	}
}

func BenchmarkMatch_Exact(b *testing.B) {
	p := matcher.Parse("user.created")
	b.ResetTimer()
	for b.Loop() {
		sink = matcher.Match(p, "user.created")
	}
}

func BenchmarkMatch_Global(b *testing.B) {
	p := matcher.Parse("*")
	b.ResetTimer()
	for b.Loop() {
		sink = matcher.Match(p, "user.created")
	}
}

func BenchmarkMatch_SingleSuffix_Hit(b *testing.B) {
	p := matcher.Parse("user.*")
	b.ResetTimer()
	for b.Loop() {
		sink = matcher.Match(p, "user.created")
	}
}

func BenchmarkMatch_SingleSuffix_Miss(b *testing.B) {
	p := matcher.Parse("user.*")
	b.ResetTimer()
	for b.Loop() {
		sink = matcher.Match(p, "user.profile.updated")
	}
}

func BenchmarkMatch_Complex_3Seg(b *testing.B) {
	p := matcher.Parse("tenant.*.user.created")
	b.ResetTimer()
	for b.Loop() {
		sink = matcher.Match(p, "tenant.abc.user.created")
	}
}

func BenchmarkMatch_Complex_Multi(b *testing.B) {
	p := matcher.Parse("tenant.*.user.>")
	b.ResetTimer()
	for b.Loop() {
		sink = matcher.Match(p, "tenant.abc.user.profile.updated")
	}
}

func BenchmarkMatch_Complex_DeepTopic(b *testing.B) {
	p := matcher.Parse("a.b.c.d.>")
	b.ResetTimer()
	for b.Loop() {
		sink = matcher.Match(p, "a.b.c.d.e.f.g.h")
	}
}
