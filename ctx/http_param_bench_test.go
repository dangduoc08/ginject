package ctx

import "testing"

func BenchmarkHTTPContext_Param(b *testing.B) {
	c := newTestHTTPContext()
	c.ParamKeys = map[string][]int{"id": {0}, "name": {1}}
	c.ParamValues = []string{"42", "joe"}
	var n int
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.param = nil
		n += len(c.Param())
	}
	b.StopTimer()
	if n == 0 {
		b.Fatal("Param should not be empty")
	}
}

func BenchmarkParam_Bind(b *testing.B) {
	p := Param{"id": {"42"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Bind(paramBindDTO{})
	}
}
