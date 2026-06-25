package ds

import (
	"testing"
)

func BenchmarkMatchWildcard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		matchWildcard("index.html", "in*.html")
	}
}
