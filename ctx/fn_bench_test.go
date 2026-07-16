package ctx

import "testing"

func BenchmarkGetTagParams_Single(b *testing.B) {
	v := "field_name"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParams(v)
	}
}

func BenchmarkGetTagParams_Multiple(b *testing.B) {
	v := "field_name, required, min=1, max=100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParams(v)
	}
}

func BenchmarkGetTagParamIndex_WithDot(b *testing.B) {
	v := "integers_1.3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParamIndex(v)
	}
}

func BenchmarkGetTagParamIndex_NoDot(b *testing.B) {
	v := "field_name"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParamIndex(v)
	}
}

func BenchmarkResolveWSEventname(b *testing.B) {
	e := "CHAT_HELLO_WORLD"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveWSEventName(e)
	}
}
