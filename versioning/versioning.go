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

func (versioning *Versioning) GetTypeString() string {
	switch versioning.Type {
	case QUERY:
		return "query"
	case HEADER:
		return "header"
	case CUSTOM:
		return "custom"
	case MEDIA_TYPE:
		return "media_type"
	default:
		return ""
	}
}

func (versioning *Versioning) GetVersion(c *ctx.Context) string {
	switch versioning.Type {
	case QUERY:
		if v := c.Query().Get(versioning.Key); v != "" {
			return v
		}
	case HEADER:
		if v := c.Header().Get(versioning.Key); v != "" {
			return v
		}
	case CUSTOM:
		if versioning.Extractor != nil {
			return versioning.Extractor(c)
		}
	case MEDIA_TYPE:
		if _, after, found := strings.Cut(c.Header().Get("Accept"), versioning.Key+"="); found {
			if v, _, _ := strings.Cut(strings.TrimSpace(after), ";"); v != "" {
				return v
			}
		}
	}

	return versioning.DefaultVersion
}
