package exception

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

type Exception struct {
	response any
	error    error
	code     string
}

type ExceptionOptions struct {
	Description string
	Cause       error
}

func (exception Exception) Error() string {
	return exception.error.Error()
}

func (exception Exception) Unwrap() error {
	return errors.Unwrap(exception.error)
}

func (exception Exception) GetCode() string {
	return exception.code
}

func (exception Exception) GetResponse() any {
	return exception.response
}

func (exception Exception) GetHTTPStatus() (int, string) {
	codeInt, err := strconv.Atoi(exception.code)
	if err == nil {
		statusText := http.StatusText(codeInt)
		if statusText != "" {
			return codeInt, statusText
		}
	}
	return 0, ""
}

func (exception Exception) errorBuilder(opts ...any) Exception {

	// By default error will be HTTP statuses
	_, text := exception.GetHTTPStatus()
	if text != "" {
		exception.error = errors.New(text)
	}

	if len(opts) > 0 {
		switch option := opts[0].(type) {
		case string:
			exception.error = errors.New(option)
		case Exception:
			exception.error = option.error
		case error:
			exception.error = option
		case ExceptionOptions:
			if option.Description != "" {
				exception.error = errors.New(option.Description)
			}

			if option.Cause != nil {
				exception.error = fmt.Errorf("%v: %w", exception.Error(), option.Cause)
			}
		}
	}

	return exception
}

func NewException(response any, code string, opts ...any) Exception {
	return Exception{
		response: response,
		code:     code,
	}.
		errorBuilder(opts...)
}
