package color

import "testing"

func BenchmarkFmtRed_NoArgs(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		FmtRed("GET /users 200")
	}
}

func BenchmarkFmtRed_NoArgsPercent(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		FmtRed("cpu 100%%")
	}
}

func BenchmarkFmtRed_Args(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		FmtRed("%s %s %d", "GET", "/users", 200)
	}
}

func BenchmarkFmtBGRed_Args(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		FmtBGRed("%s %s %d", "GET", "/users", 200)
	}
}

func BenchmarkFmtPurple_Args(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		FmtPurple("%s %s %d", "GET", "/users", 200)
	}
}

func BenchmarkFmtBold_Nested(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		FmtBold("%s", "a "+FmtRed("b")+" c")
	}
}
