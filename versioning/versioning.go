package versioning

import (
	"strings"

	"github.com/dangduoc08/ginject/ctx"
)

const (
	QueryVersion = iota + 1
	HeaderVersion
	CustomVersion
	MediaType
)

const NeutralVersion = "NEUTRAL"

type ExtractorHandler = func(*ctx.Context) string

type Versioning struct {
	Type           int
	Key            string
	DefaultVersion string
	Extractor      ExtractorHandler
}

func (versioning *Versioning) GetTypeString() string {
	switch versioning.Type {
	case QueryVersion:
		return "query"
	case HeaderVersion:
		return "header"
	case CustomVersion:
		return "custom"
	case MediaType:
		return "media_type"
	default:
		return ""
	}
}

func (versioning *Versioning) GetVersion(c *ctx.Context) string {
	switch versioning.Type {
	case QueryVersion:
		if v := c.Query().Get(versioning.Key); v != "" {
			return v
		}
	case HeaderVersion:
		if v := c.Header().Get(versioning.Key); v != "" {
			return v
		}
	case CustomVersion:
		if versioning.Extractor != nil {
			return versioning.Extractor(c)
		}
	case MediaType:
		if _, after, found := strings.Cut(c.Header().Get("Accept"), versioning.Key+"="); found {
			if v, _, _ := strings.Cut(strings.TrimSpace(after), ";"); v != "" {
				return v
			}
		}
	}

	return versioning.DefaultVersion
}
