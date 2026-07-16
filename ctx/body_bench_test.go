package ctx

import "testing"

func BenchmarkBodyGet_Shallow(b *testing.B) {
	body := Body{"key": "value"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body.Get("key")
	}
}

func BenchmarkBodyGet_Deep(b *testing.B) {
	body := Body{
		"a": map[string]any{
			"b": map[string]any{
				"c": map[string]any{
					"d": "value",
				},
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body.Get("a.b.c.d")
	}
}
