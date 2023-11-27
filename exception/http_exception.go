package exception

import (
	"net/http"
	"strconv"
)

type HTTPException struct {
	response any
	error    string
	code     string
}

func (httpException HTTPException) Error() string {
	return httpException.error
}

func (httpException HTTPException) GetCode() string {
	return httpException.code
}

func (httpException HTTPException) GetHTTPStatus() (int, string) {
	codeInt, err := strconv.Atoi(httpException.code)
	if err == nil {
		statusText := http.StatusText(codeInt)
		if statusText != "" {
			return codeInt, statusText
		}
	}
	return 0, ""
}

func (httpException HTTPException) GetResponse() any {
	return httpException.response
}

func (httpException HTTPException) errorBuilder(response any, code string, opts ...any) HTTPException {
	httpException.response = response
	httpException.code = code

	if len(opts) > 0 {
		switch option := opts[0].(type) {
		case string:
			httpException.error = option
		case error:
			httpException.error = option.Error()
		case map[string]any:
			if v, ok := option["description"]; ok {
				switch desc := v.(type) {
				case string:
					httpException.error = desc
				}
			} else if v, ok := option["cause"]; ok {
				switch desc := v.(type) {
				case error:
					httpException.error = desc.Error()
				}
			}
		}
	} else {
		_, text := httpException.GetHTTPStatus()
		if text != "" {
			httpException.error = text
		}
	}

	return httpException
}

func NewHTTPException(response any, code string, opts ...any) HTTPException {
	return HTTPException{}.errorBuilder(response, code, opts...)
}
