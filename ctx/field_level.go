package ctx

import "reflect"

type FieldLevel struct {
	val       any
	typ       reflect.Type
	tag       string
	nestedTag string
	ns        string
	field     string
	index     int
	kind      reflect.Kind
	isVal     bool
}

func (fl *FieldLevel) Tag() string {
	return fl.tag
}

func (fl *FieldLevel) NestedTag() string {
	return fl.nestedTag
}

func (fl *FieldLevel) Namespace() string {
	return fl.ns
}

func (fl *FieldLevel) Field() string {
	return fl.field
}

func (fl *FieldLevel) Index() int {
	return fl.index
}

func (fl *FieldLevel) Value() any {
	return fl.val
}

func (fl *FieldLevel) IsValue() bool {
	return fl.isVal
}

func (fl *FieldLevel) Kind() reflect.Kind {
	return fl.kind
}

func (fl *FieldLevel) Type() reflect.Type {
	return fl.typ
}
