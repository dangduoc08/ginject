package exception

import (
	"errors"
	"fmt"
	"net/http"
)

type Exception struct {
	message string
	error   error
	code    int
}

type ExceptionOptions struct {
	Description string
	Cause       error
}

func (e Exception) Error() string {
	if e.error == nil {
		return ""
	}
	return e.error.Error()
}

func (e Exception) Unwrap() error {
	return e.error
}

func (e Exception) GetCode() int {
	return e.code
}

func (e Exception) GetMessage() string {
	return e.message
}

func (e Exception) GetStatusText() string {
	statusText := http.StatusText(e.code)
	if statusText == "" {
		statusText = wsCloseStatusText[e.code]
	}
	return statusText
}

func (e Exception) errorBuilder(opts ...any) Exception {
	if len(opts) == 0 {
		if text := e.GetStatusText(); text != "" {
			e.error = errors.New(text)
		}
		return e
	}

	switch option := opts[0].(type) {
	case string:
		e.error = errors.New(option)
	case error:
		e.error = option
	case ExceptionOptions:
		if option.Description != "" {
			e.error = errors.New(option.Description)
		} else if text := e.GetStatusText(); text != "" {
			e.error = errors.New(text)
		}

		if option.Cause == nil {
			break
		}

		if e.error == nil {
			e.error = option.Cause
			break
		}

		e.error = fmt.Errorf("%v: %w", e.error, option.Cause)
	}

	return e
}

func NewException(message string, code int, opts ...any) Exception {
	return Exception{message: message, code: code}.errorBuilder(opts...)
}
