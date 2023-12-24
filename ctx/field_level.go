package ctx

import "reflect"

type FieldLevel struct {
	tag   string
	ns    string
	field string
	index int
	val   any
	isVal bool
	kind  reflect.Kind
	typ   reflect.Type
}

func (fl *FieldLevel) Tag() string {
	return fl.tag
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
