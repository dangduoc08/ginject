package httpclient

import (
	"fmt"
	"net/http"
	"strings"
)

var maskedHeaders = map[string]bool{
	"authorization": true,
	"cookie":        true,
	"set-cookie":    true,
	"x-api-key":     true,
}

func debugRequest(req *http.Request) {
	fmt.Printf("[httpclient] --> %s %s\n", req.Method, req.URL.String())
	for k, vv := range req.Header {
		val := strings.Join(vv, ", ")
		if maskedHeaders[strings.ToLower(k)] {
			val = "***"
		}
		fmt.Printf("  >  %s: %s\n", k, val)
	}
}

func debugResponse(resp *Response, timing *TimingInfo) {
	fmt.Printf("[httpclient] <-- %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	for k, vv := range resp.Headers {
		val := strings.Join(vv, ", ")
		if maskedHeaders[strings.ToLower(k)] {
			val = "***"
		}
		fmt.Printf("  <  %s: %s\n", k, val)
	}
	if timing != nil {
		fmt.Printf("  timing: total=%v ttfb=%v dns=%v tcp=%v tls=%v\n",
			timing.Total, timing.TTFB, timing.DNS, timing.TCP, timing.TLS)
	}
}
