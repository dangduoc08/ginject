package log

import (
	"fmt"
	"reflect"
)

const (
	logTagKey           = "log"
	maxDepth            = 16
	maxNodes            = 1000
	maxDepthPlaceholder = "[MAX_DEPTH]"

	labelDebug = "DEBUG"
	labelInfo  = "INFO"
	labelWarn  = "WARN"
	labelError = "ERROR"
	labelFatal = "FATAL"
)

func taggedFields(rv reflect.Value) []kv {
	t := rv.Type()
	var out []kv
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag, hasTag := f.Tag.Lookup(logTagKey)
		if tag == "-" {
			continue
		}
		if !hasTag {
			tag = f.Name
		}
		out = append(out, kv{tag, rv.Field(i)})
	}
	return out
}

type kv struct {
	key string
	val reflect.Value
}

var (
	stringerType = reflect.TypeFor[fmt.Stringer]()
	errorType    = reflect.TypeFor[error]()
)

func dereference(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return reflect.Value{}
		}
		rv = rv.Elem()
	}
	return rv
}

func mapKeyString(key reflect.Value) string {
	if key.Kind() == reflect.String {
		return key.String()
	}
	return fmt.Sprint(key.Interface())
}
