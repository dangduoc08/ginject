package exception

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
)

func BenchmarkBadRequestException(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = BadRequestException("validation error")
	}
}

func BenchmarkNotFoundException(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NotFoundException("not found")
	}
}

func BenchmarkInternalServerErrorException(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = InternalServerErrorException("internal error")
	}
}

func BenchmarkNewException_NoOpts(b *testing.B) {
	code := strconv.Itoa(http.StatusBadRequest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewException("body", code)
	}
}

func BenchmarkNewException_StringOpt(b *testing.B) {
	code := strconv.Itoa(http.StatusBadRequest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewException("body", code, "custom message")
	}
}

func BenchmarkNewException_ExceptionOptions_Cause(b *testing.B) {
	code := strconv.Itoa(http.StatusBadRequest)
	cause := errors.New("db error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewException("body", code, ExceptionOptions{Cause: cause})
	}
}

func BenchmarkGetHTTPStatus(b *testing.B) {
	e := BadRequestException("body")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.GetHTTPStatus()
	}
}
