package common

import "testing"

func BenchmarkArgumentMetadata_Init(b *testing.B) {
	var sink ArgumentMetadata
	for range b.N {
		sink = ArgumentMetadata{
			ContextType: "http",
			ParamType:   BodyPipeableKey,
		}
	}
	_ = sink
}

func BenchmarkArgumentMetadata_ZeroValue(b *testing.B) {
	var sink ArgumentMetadata
	for range b.N {
		sink = ArgumentMetadata{}
	}
	_ = sink
}

func BenchmarkPipeableInterfaceAssertion(b *testing.B) {
	var iface any = mockBodyPipeable{}
	var sink bool
	for range b.N {
		_, sink = iface.(BodyPipeable)
	}
	_ = sink
}
