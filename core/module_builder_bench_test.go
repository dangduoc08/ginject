package core

import (
	"testing"
)

func BenchmarkModuleBuilderBuild(b *testing.B) {
	p := &mockProvider{}
	c := &mockController{}
	child := ModuleBuilder().Build()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ModuleBuilder().Imports(child).Providers(p).Controllers(c).Build()
	}
}

func BenchmarkGetModuleType(b *testing.B) {
	child1 := ModuleBuilder().Build()
	child2 := ModuleBuilder().Build()
	child3 := ModuleBuilder().Build()
	builder := ModuleBuilder().Imports(child1, child2, child3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.getModuleType()
	}
}
