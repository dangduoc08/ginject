package exception

import (
	"errors"
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestException_Error_NilError(t *testing.T) {
	var e Exception
	if e.Error() != "" {
		t.Error(test.DiffMessage(e.Error(), "", "Error on zero-value Exception must not panic"))
	}
}

func TestException_Error_WithError(t *testing.T) {
	e := NewException("body", http.StatusBadRequest, "custom message")
	if e.Error() != "custom message" {
		t.Error(test.DiffMessage(e.Error(), "custom message", "Error returns wrapped error text"))
	}
}

func TestException_Unwrap_Nil(t *testing.T) {
	var e Exception
	if e.Unwrap() != nil {
		t.Error(test.DiffMessage(e.Unwrap(), nil, "Unwrap on zero-value Exception"))
	}
}

func TestException_Unwrap_ReturnsDirectError(t *testing.T) {
	cause := errors.New("root cause")
	e := NewException("body", http.StatusBadRequest, ExceptionOptions{Cause: cause})
	if !errors.Is(e.Unwrap(), cause) && e.Unwrap() != cause {
		t.Error(test.DiffMessage(e.Unwrap(), cause, "Unwrap must return the directly wrapped error"))
	}
}

func TestException_ErrorsIs_ChainWorks(t *testing.T) {
	cause := errors.New("db connection refused")
	e := NewException("body", http.StatusInternalServerError, ExceptionOptions{Cause: cause})
	if !errors.Is(e, cause) {
		t.Error(test.DiffMessage(errors.Is(e, cause), true, "errors.Is must walk through Exception to find cause"))
	}
}

func TestException_ErrorsAs_ChainWorks(t *testing.T) {
	type customErr struct{ error }
	cause := &customErr{errors.New("wrapped")}
	e := NewException("body", http.StatusInternalServerError, ExceptionOptions{Cause: cause})

	var target *customErr
	if !errors.As(e, &target) {
		t.Error(test.DiffMessage(errors.As(e, &target), true, "errors.As must walk through Exception to find cause"))
	}
}

func TestException_GetCode(t *testing.T) {
	e := NewException("body", 999)
	if e.GetCode() != 999 {
		t.Error(test.DiffMessage(e.GetCode(), 999, "GetCode returns raw code"))
	}
}

func TestException_GetMessage(t *testing.T) {
	e := NewException("hello", http.StatusBadRequest)
	if e.GetMessage() != "hello" {
		t.Error(test.DiffMessage(e.GetMessage(), "hello", "GetMessage"))
	}
}

func TestException_GetStatusText_HTTPValid(t *testing.T) {
	e := NewException("body", http.StatusNotFound)
	if text := e.GetStatusText(); text != "Not Found" {
		t.Error(test.DiffMessage(text, "Not Found", "GetStatusText valid HTTP code"))
	}
}

func TestException_GetStatusText_WSValid(t *testing.T) {
	e := NewException("body", 1008)
	if text := e.GetStatusText(); text != "Policy Violation" {
		t.Error(test.DiffMessage(text, "Policy Violation", "GetStatusText valid WS code"))
	}
}

func TestException_GetStatusText_Unknown(t *testing.T) {
	e := NewException("body", 999)
	if text := e.GetStatusText(); text != "" {
		t.Error(test.DiffMessage(text, "", "GetStatusText unknown code"))
	}
}

func TestNewException_DefaultErrorFromHTTPCode(t *testing.T) {
	e := NewException("body", http.StatusBadRequest)
	if e.Error() != "Bad Request" {
		t.Error(test.DiffMessage(e.Error(), "Bad Request", "default error from HTTP status"))
	}
}

func TestNewException_DefaultErrorFromWSCode(t *testing.T) {
	e := NewException("body", 1008)
	if e.Error() != "Policy Violation" {
		t.Error(test.DiffMessage(e.Error(), "Policy Violation", "default error from WS close status"))
	}
}

func TestNewException_UnknownCodeHasNoDefaultError(t *testing.T) {
	e := NewException("body", 999)
	if e.Error() != "" {
		t.Error(test.DiffMessage(e.Error(), "", "unknown code has no default error text"))
	}
}

func TestNewException_StringOpt(t *testing.T) {
	e := NewException("body", http.StatusBadRequest, "custom message")
	if e.Error() != "custom message" {
		t.Error(test.DiffMessage(e.Error(), "custom message", "string opt overrides error"))
	}
}

func TestNewException_ErrorOpt(t *testing.T) {
	cause := errors.New("underlying error")
	e := NewException("body", http.StatusBadRequest, cause)
	if e.Error() != "underlying error" {
		t.Error(test.DiffMessage(e.Error(), "underlying error", "error opt sets error"))
	}
}

func TestNewException_ExceptionOptions_Description(t *testing.T) {
	e := NewException("body", http.StatusBadRequest, ExceptionOptions{
		Description: "validation failed",
	})
	if e.Error() != "validation failed" {
		t.Error(test.DiffMessage(e.Error(), "validation failed", "ExceptionOptions description"))
	}
}

func TestNewException_ExceptionOptions_Cause(t *testing.T) {
	cause := errors.New("db error")
	e := NewException("body", http.StatusInternalServerError, ExceptionOptions{
		Cause: cause,
	})
	if !errors.Is(e, cause) {
		t.Error(test.DiffMessage(errors.Is(e, cause), true, "cause is wrapped"))
	}
}

func TestNewException_ExceptionOptions_DescriptionAndCause(t *testing.T) {
	cause := errors.New("root cause")
	e := NewException("body", http.StatusBadRequest, ExceptionOptions{
		Description: "custom",
		Cause:       cause,
	})
	if !errors.Is(e, cause) {
		t.Error(test.DiffMessage(errors.Is(e, cause), true, "cause wrapped under description"))
	}
	if e.Error() == "" {
		t.Error(test.DiffMessage(e.Error(), "non-empty", "error string not empty"))
	}
}

func TestNewException_ExceptionOptions_NilCauseKeepsDefault(t *testing.T) {
	e := NewException("body", http.StatusBadRequest, ExceptionOptions{})
	if e.Error() != "Bad Request" {
		t.Error(test.DiffMessage(e.Error(), "Bad Request", "nil cause keeps default status text"))
	}
}

func TestNewException_ExceptionOptions_CauseNoDefaultText(t *testing.T) {
	cause := errors.New("root cause")
	e := NewException("body", 999, ExceptionOptions{Cause: cause})
	if e.Error() != "root cause" {
		t.Error(test.DiffMessage(e.Error(), "root cause", "cause becomes error when no default text exists"))
	}
}
