package exception

import "testing"

func BenchmarkBadRequestException(b *testing.B) {
	for range b.N {
		_ = BadRequestException("validation error")
	}
}

func BenchmarkNotFoundException(b *testing.B) {
	for range b.N {
		_ = NotFoundException("not found")
	}
}

func BenchmarkInternalServerErrorException(b *testing.B) {
	for range b.N {
		_ = InternalServerErrorException("internal error")
	}
}
