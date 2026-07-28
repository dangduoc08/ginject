package log

import (
	"reflect"
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/matcher"
)

const maskPlaceholder = "[REDACTED]"

func WrapLogger(next common.Logger, maskFields []string) common.Logger {
	rules := make([]matcher.Pattern, len(maskFields))
	for i, f := range maskFields {
		rules[i] = matcher.Parse(f)
	}
	return &maskingLogger{next: next, rules: rules}
}

type maskingLogger struct {
	next  common.Logger
	rules []matcher.Pattern
}

func (l *maskingLogger) Debug(msg string, args ...any) { l.next.Debug(msg, l.transform(args)...) }
func (l *maskingLogger) Info(msg string, args ...any)  { l.next.Info(msg, l.transform(args)...) }
func (l *maskingLogger) Warn(msg string, args ...any)  { l.next.Warn(msg, l.transform(args)...) }
func (l *maskingLogger) Error(msg string, args ...any) { l.next.Error(msg, l.transform(args)...) }
func (l *maskingLogger) Fatal(msg string, args ...any) { l.next.Fatal(msg, l.transform(args)...) }

func (l *maskingLogger) transform(args []any) []any {
	if len(args) == 0 {
		return args
	}

	out := make([]any, len(args))
	for i := 0; i < len(args); i += 2 {
		out[i] = args[i]
		if i+1 >= len(args) {
			break
		}
		key, ok := args[i].(string)
		if !ok {
			out[i+1] = args[i+1]
			continue
		}
		out[i+1] = l.resolveAny(key, args[i+1])
	}
	return out
}

func isPossiblyTaggable(v any) bool {
	switch v.(type) {
	case nil, string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		error:
		return false
	default:
		return true
	}
}

func (l *maskingLogger) resolveAny(path string, v any) any {
	if !isPossiblyTaggable(v) {
		if l.matchesAny(path) {
			return maskPlaceholder
		}
		return v
	}
	return l.resolveReflect(path, reflect.ValueOf(v))
}

func (l *maskingLogger) resolveReflect(path string, rv reflect.Value) any {
	if l.matchesExact(path) {
		return maskPlaceholder
	}

	rv = dereference(rv)
	if !rv.IsValid() {
		return nil
	}

	if isTaggableStruct(rv) {
		fields := taggedFields(rv)
		if len(fields) == 0 {
			if l.matchesAny(path) {
				return maskPlaceholder
			}
			return rv.Interface()
		}
		out := make(map[string]any, len(fields))
		for _, f := range fields {
			out[f.key] = l.resolveReflect(path+"."+f.key, f.val)
		}
		return out
	}

	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		keys := rv.MapKeys()
		if len(keys) == 0 {
			if l.matchesAny(path) {
				return maskPlaceholder
			}
			return rv.Interface()
		}
		out := make(map[string]any, len(keys))
		for _, k := range keys {
			ks := k.String()
			out[ks] = l.resolveReflect(path+"."+ks, rv.MapIndex(k))
		}
		return out
	}

	if l.matchesAny(path) {
		return maskPlaceholder
	}
	return rv.Interface()
}

func (l *maskingLogger) matchesExact(path string) bool {
	for _, r := range l.rules {
		if r.IsExact() && matchesTrailingPath(r.Raw(), path) {
			return true
		}
	}
	return false
}

func (l *maskingLogger) matchesAny(path string) bool {
	for _, r := range l.rules {
		if r.IsExact() {
			if matchesTrailingPath(r.Raw(), path) {
				return true
			}
			continue
		}
		if matcher.Match(r, path) {
			return true
		}
	}
	return false
}

func matchesTrailingPath(pattern, path string) bool {
	if pattern == path {
		return true
	}
	return strings.HasSuffix(path, "."+pattern)
}
