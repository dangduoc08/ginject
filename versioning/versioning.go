package versioning

import (
	"strings"

	"github.com/dangduoc08/ginject/ctx"
)

const (
	QUERY = iota + 1
	HEADER
	CUSTOM
	MEDIA_TYPE
)

const NEUTRAL_VERSION = "NEUTRAL"

type ExtractorHandler = func(*ctx.Context) string

type Versioning struct {
	Type           int
	Key            string
	DefaultVersion string
	Extractor      ExtractorHandler
}

func (versioning *Versioning) GetVersion(c *ctx.Context) string {
	v := ""
	key := versioning.Key
	if key == "" {
		key = "v"
	}

	switch versioning.Type {
	case QUERY:
		if c.Query().Has(key) {
			v = c.Query().Get(key)
		} else {
			v = versioning.DefaultVersion
		}

	case HEADER:
		if c.Header().Has(key) {
			v = c.Header().Get(key)
		} else {
			v = versioning.DefaultVersion
		}

	case CUSTOM:
		if versioning.Extractor != nil {
			v = versioning.Extractor(c)
		} else {
			v = versioning.DefaultVersion
		}

	case MEDIA_TYPE:
		if _, after, found := strings.Cut(c.Header().Get("Accept"), key+"="); found {
			v, _, _ = strings.Cut(strings.TrimSpace(after), ";")
		} else {
			v = versioning.DefaultVersion
		}
	}

	return v
}
