package httpclient

import (
	"crypto/tls"
	"errors"
	"net/http"
)

type secureRoundTripper struct {
	base         http.RoundTripper
	isHTTPSRequired bool
	validateHost func(string) bool
}

func (t *secureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.isHTTPSRequired && req.URL.Scheme != "https" {
		return nil, errors.New("httpclient: HTTPS required but scheme is " + req.URL.Scheme)
	}
	if t.validateHost != nil && !t.validateHost(req.URL.Hostname()) {
		return nil, errors.New("httpclient: host not allowed: " + req.URL.Hostname())
	}
	return t.base.RoundTrip(req)
}

func buildTransport(tlsCfg *tls.Config, isHTTPSRequired bool, validateHost func(string) bool) http.RoundTripper {
	var base http.RoundTripper
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		base = t.Clone()
	} else {
		base = http.DefaultTransport
	}
	if tlsCfg != nil {
		if t, ok := base.(*http.Transport); ok {
			t.TLSClientConfig = tlsCfg
		}
	}
	if isHTTPSRequired || validateHost != nil {
		return &secureRoundTripper{
			base:         base,
			isHTTPSRequired: isHTTPSRequired,
			validateHost: validateHost,
		}
	}
	return base
}
