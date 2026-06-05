package color

import "testing"

func BenchmarkFmtGreen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FmtGreen("status: %s %d", "ok", 200)
	}
}

func BenchmarkFmtRed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FmtRed("error: %s", "something went wrong")
	}
}

func BenchmarkFmtBGDim(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FmtBGDim("label: %s", "info")
	}
}

func BenchmarkFmtPurple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FmtPurple("value: %d", 42)
	}
}
