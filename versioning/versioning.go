package versioning

import (
	"strings"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/utils"
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
	defaultKey := "v"
	key := versioning.Key
	if key == "" {
		key = defaultKey
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
		if c.Header().Has("Accept") {
			headerVals := strings.Split(c.Header().Get("Accept"), ";")
			kv := utils.ArrFind(headerVals, func(val string, i int) bool {
				return strings.Contains(val, key+"=")
			})
			if len(kv) > 0 {
				kv = strings.TrimSpace(kv)
				i := strings.Index(kv, "=")
				if i > -1 {
					v = kv[i+1:]
				}
			} else {
				v = versioning.DefaultVersion
			}
		} else {
			v = versioning.DefaultVersion
		}
	}

	return v
}
