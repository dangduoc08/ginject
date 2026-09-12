package ctx

import "reflect"

type WSPayload map[string]any

type WSTopic string

func (p WSPayload) Bind(s any) (any, []FieldLevel) {
	fls := make([]FieldLevel, 0, reflect.TypeOf(s).NumField())
	return BindStruct(p, &fls, s, "", "")
}
