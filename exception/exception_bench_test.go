package exception

import (
	"errors"
	"net/http"
	"testing"
)

func BenchmarkNewException_NoOpts(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_ = NewException("body", http.StatusBadRequest)
	}
}

func BenchmarkNewException_StringOpt(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_ = NewException("body", http.StatusBadRequest, "custom message")
	}
}

func BenchmarkNewException_ErrorOpt(b *testing.B) {
	cause := errors.New("underlying error")
	b.ResetTimer()
	for range b.N {
		_ = NewException("body", http.StatusBadRequest, cause)
	}
}

func BenchmarkNewException_ExceptionOptions_Cause(b *testing.B) {
	cause := errors.New("db error")
	b.ResetTimer()
	for range b.N {
		_ = NewException("body", http.StatusBadRequest, ExceptionOptions{Cause: cause})
	}
}

func BenchmarkException_GetStatusText(b *testing.B) {
	e := NewException("body", http.StatusBadRequest)
	b.ResetTimer()
	for range b.N {
		_ = e.GetStatusText()
	}
}

func BenchmarkException_Error(b *testing.B) {
	e := NewException("body", http.StatusBadRequest)
	b.ResetTimer()
	for range b.N {
		_ = e.Error()
	}
}

func BenchmarkException_ErrorsIs(b *testing.B) {
	cause := errors.New("db error")
	e := NewException("body", http.StatusInternalServerError, ExceptionOptions{Cause: cause})
	b.ResetTimer()
	for range b.N {
		_ = errors.Is(e, cause)
	}
}
