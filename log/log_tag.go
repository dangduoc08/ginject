package log

import (
	"log/slog"
	"reflect"
	"unsafe"
)

const logTagKey = "log"

var logValuerType = reflect.TypeFor[slog.LogValuer]()

func isTaggableStruct(rv reflect.Value) bool {
	if rv.Kind() != reflect.Struct {
		return false
	}
	t := rv.Type()
	return !t.Implements(stringerType) && !t.Implements(errorType) && !t.Implements(logValuerType)
}

func taggedFields(rv reflect.Value) []kv {
	if !rv.CanAddr() {
		addr := reflect.New(rv.Type()).Elem()
		addr.Set(rv)
		rv = addr
	}

	t := rv.Type()
	var out []kv
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag, hasTag := f.Tag.Lookup(logTagKey)
		if !hasTag {
			if f.PkgPath != "" {
				continue
			}
			out = append(out, kv{f.Name, rv.Field(i)})
			continue
		}
		if tag == "-" {
			continue
		}

		fv := rv.Field(i)
		if !fv.CanInterface() {
			fv = reflect.NewAt(fv.Type(), unsafe.Pointer(fv.UnsafeAddr())).Elem()
		}
		out = append(out, kv{tag, fv})
	}
	return out
}
