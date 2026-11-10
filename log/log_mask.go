package log

import (
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/pattern"
)

const maskPlaceholder = "[REDACTED]"

var (
	logValuerType = reflect.TypeFor[slog.LogValuer]()
	slogValueType = reflect.TypeFor[slog.Value]()
	slogAttrType  = reflect.TypeFor[slog.Attr]()
	slogAttrsType = reflect.TypeFor[[]slog.Attr]()
)

type levelEnabler interface {
	enabled(level slog.Level) bool
}

func WrapLogger(next common.Logger, maskFields []string) common.Logger {
	rules := make([]pattern.Pattern, len(maskFields))
	for i, f := range maskFields {
		rules[i] = pattern.NewPattern(strings.ToLower(f))
	}
	enabler, _ := next.(levelEnabler)
	return &maskingLogger{next: next, rules: rules, enabler: enabler}
}

type maskingLogger struct {
	next    common.Logger
	rules   []pattern.Pattern
	enabler levelEnabler
}

func (l *maskingLogger) Debug(msg string, args ...any) {
	if l.enabled(DebugLevel) {
		l.next.Debug(msg, l.transform(args)...)
	}
}

func (l *maskingLogger) Info(msg string, args ...any) {
	if l.enabled(InfoLevel) {
		l.next.Info(msg, l.transform(args)...)
	}
}

func (l *maskingLogger) Warn(msg string, args ...any) {
	if l.enabled(WarnLevel) {
		l.next.Warn(msg, l.transform(args)...)
	}
}

func (l *maskingLogger) Error(msg string, args ...any) {
	if l.enabled(ErrorLevel) {
		l.next.Error(msg, l.transform(args)...)
	}
}

func (l *maskingLogger) Fatal(msg string, args ...any) { l.next.Fatal(msg, l.transform(args)...) }

func (l *maskingLogger) enabled(level slog.Level) bool {
	return l.enabler == nil || l.enabler.enabled(level)
}

func (l *maskingLogger) transform(args []any) []any {
	var out []any
	budget := maxNodes
	for i := 0; i < len(args); {
		switch arg := args[i].(type) {
		case slog.Attr:
			if masked, changed := l.maskAttr("", arg, &budget); changed {
				out = setArg(out, args, i, masked)
			}
			i++
		case string:
			if i+1 < len(args) {
				if masked, changed := l.maskAny(l.childPath("", arg), args[i+1], &budget); changed {
					out = setArg(out, args, i+1, masked)
				}
			}
			i += 2
		default:
			i++
		}
	}
	if out == nil {
		return args
	}
	return out
}

func setArg(out, args []any, i int, value any) []any {
	if out == nil {
		out = slices.Clone(args)
	}
	out[i] = value
	return out
}

func (l *maskingLogger) maskAttr(parent string, attr slog.Attr, budget *int) (slog.Attr, bool) {
	path := l.childPath(parent, attr.Key)
	if attr.Key != "" && l.matches(path) {
		return slog.String(attr.Key, maskPlaceholder), true
	}

	if masked, changed := l.maskSlogValue(path, attr.Value.Resolve(), budget); changed {
		return slog.Attr{Key: attr.Key, Value: masked}, true
	}
	return attr, false
}

func (l *maskingLogger) maskSlogValue(path string, value slog.Value, budget *int) (slog.Value, bool) {
	switch value.Kind() {
	case slog.KindGroup:
		group := value.Group()
		var out []slog.Attr
		for i, sub := range group {
			if masked, changed := l.maskAttr(path, sub, budget); changed {
				if out == nil {
					out = slices.Clone(group)
				}
				out[i] = masked
			}
		}
		if out != nil {
			return slog.GroupValue(out...), true
		}
	case slog.KindAny:
		if masked, changed := l.maskNested(path, value.Any(), budget); changed {
			return slog.AnyValue(masked), true
		}
	}
	return value, false
}

func (l *maskingLogger) childPath(parent, key string) string {
	if len(l.rules) == 0 {
		return ""
	}
	return joinPath(parent, strings.ToLower(key))
}

func joinPath(parent, key string) string {
	switch {
	case parent == "":
		return key
	case key == "":
		return parent
	default:
		return parent + "." + key
	}
}

func (l *maskingLogger) maskAny(path string, v any, budget *int) (any, bool) {
	if l.matches(path) {
		return maskPlaceholder, true
	}
	return l.maskNested(path, v, budget)
}

func (l *maskingLogger) maskNested(path string, v any, budget *int) (any, bool) {
	switch value := v.(type) {
	case slog.LogValuer:
		if masked, changed := l.maskSlogValue(path, slog.AnyValue(value).Resolve(), budget); changed {
			return masked, true
		}
		return v, false
	case slog.Value:
		if masked, changed := l.maskSlogValue(path, value.Resolve(), budget); changed {
			return masked, true
		}
		return v, false
	case slog.Attr:
		if masked, changed := l.maskSlogValue(path, slog.GroupValue(value), budget); changed {
			return masked, true
		}
		return v, false
	}
	if !isPossiblyTaggable(v) {
		return v, false
	}
	return l.maskValue(path, reflect.ValueOf(v), 0, budget)
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

func (l *maskingLogger) maskValue(path string, rv reflect.Value, depth int, budget *int) (any, bool) {
	rv = dereference(rv)
	if !rv.IsValid() || !isMaskContainer(rv) {
		return nil, false
	}
	if depth >= maxDepth {
		return maxDepthPlaceholder, true
	}

	switch rv.Kind() {
	case reflect.Struct:
		fields := taggedFields(rv)
		out := make(map[string]any, len(fields))
		for i, f := range fields {
			if *budget <= 0 {
				out["…"] = strconv.Itoa(len(fields)-i) + " more"
				break
			}
			out[f.key] = l.maskChild(l.childPath(path, f.key), f.val, depth, budget)
		}
		return out, true
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for seen := 0; iter.Next(); seen++ {
			if *budget <= 0 {
				out["…"] = strconv.Itoa(rv.Len()-seen) + " more"
				break
			}
			key := mapKeyString(iter.Key())
			out[key] = l.maskChild(l.childPath(path, key), iter.Value(), depth, budget)
		}
		return out, true
	default:
		if !hasNestedElements(rv.Type().Elem()) {
			return nil, false
		}
		n := rv.Len()
		out := make([]any, 0, n)
		for i := range n {
			if *budget <= 0 {
				out = append(out, "… ("+strconv.Itoa(n-i)+" more)")
				break
			}
			*budget--
			out = append(out, l.maskElem(path, rv.Index(i), depth, budget))
		}
		return out, true
	}
}

func isMaskContainer(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		t := rv.Type()
		return !t.Implements(stringerType) && !t.Implements(errorType) && !t.Implements(logValuerType)
	default:
		return false
	}
}

func (l *maskingLogger) maskChild(path string, rv reflect.Value, depth int, budget *int) any {
	*budget--
	if l.matches(path) {
		return maskPlaceholder
	}
	return l.maskElem(path, rv, depth, budget)
}

func (l *maskingLogger) maskElem(path string, rv reflect.Value, depth int, budget *int) any {
	if value, ok := slogValueOf(rv); ok {
		return l.plainValue(path, value, depth+1, budget)
	}
	if masked, changed := l.maskValue(path, rv, depth+1, budget); changed {
		return masked
	}
	return interfaceOf(rv)
}

func (l *maskingLogger) plainValue(path string, value slog.Value, depth int, budget *int) any {
	switch value.Kind() {
	case slog.KindGroup:
		if depth >= maxDepth {
			return maxDepthPlaceholder
		}
		out := make(map[string]any, len(value.Group()))
		l.plainGroup(out, path, value.Group(), depth, budget)
		return out
	case slog.KindAny:
		v := value.Any()
		if !isPossiblyTaggable(v) {
			return v
		}
		if nested, ok := slogValueOf(reflect.ValueOf(v)); ok {
			if depth >= maxDepth {
				return maxDepthPlaceholder
			}
			return l.plainValue(path, nested, depth+1, budget)
		}
		if masked, changed := l.maskValue(path, reflect.ValueOf(v), depth, budget); changed {
			return masked
		}
		return v
	default:
		return value.Any()
	}
}

func (l *maskingLogger) plainGroup(out map[string]any, path string, group []slog.Attr, depth int, budget *int) {
	for i, attr := range group {
		if *budget <= 0 {
			out["…"] = strconv.Itoa(len(group)-i) + " more"
			return
		}
		value := attr.Value.Resolve()
		*budget--
		if attr.Key == "" && value.Kind() == slog.KindGroup {
			if depth >= maxDepth {
				out["…"] = maxDepthPlaceholder
				return
			}
			l.plainGroup(out, path, value.Group(), depth+1, budget)
			continue
		}
		childPath := l.childPath(path, attr.Key)
		if l.matches(childPath) {
			out[attr.Key] = maskPlaceholder
			continue
		}
		out[attr.Key] = l.plainValue(childPath, value, depth+1, budget)
	}
}

func slogValueOf(rv reflect.Value) (slog.Value, bool) {
	for rv.IsValid() && rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	if !rv.IsValid() || !rv.CanInterface() {
		return slog.Value{}, false
	}
	switch valueType := rv.Type(); {
	case valueType == slogValueType:
		return rv.Interface().(slog.Value).Resolve(), true
	case valueType == slogAttrType:
		return slog.GroupValue(rv.Interface().(slog.Attr)), true
	case valueType == slogAttrsType:
		return slog.GroupValue(rv.Interface().([]slog.Attr)...), true
	case valueType.Implements(logValuerType):
		return slog.AnyValue(rv.Interface()).Resolve(), true
	}
	return slog.Value{}, false
}

func hasNestedElements(elemType reflect.Type) bool {
	switch elemType.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

func interfaceOf(rv reflect.Value) any {
	if !rv.IsValid() {
		return nil
	}
	return rv.Interface()
}

func (l *maskingLogger) matches(path string) bool {
	for _, rule := range l.rules {
		if matchesSuffix(rule, path) {
			return true
		}
	}
	return false
}

func matchesSuffix(rule pattern.Pattern, path string) bool {
	if rule.IsExact() {
		raw := rule.Raw()
		return path == raw || (len(path) > len(raw) && path[len(path)-len(raw)-1] == '.' && strings.HasSuffix(path, raw))
	}
	for {
		if rule.Match(path) {
			return true
		}
		dot := strings.IndexByte(path, '.')
		if dot < 0 {
			return false
		}
		path = path[dot+1:]
	}
}
