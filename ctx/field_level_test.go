package ctx

import (
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestFieldLevel_Getters(t *testing.T) {
	fl := &FieldLevel{
		tag:       "name",
		nestedTag: "user.name",
		ns:        "User.Name",
		field:     "Name",
		index:     2,
		val:       "john",
		isVal:     true,
		kind:      reflect.String,
		typ:       reflect.TypeOf(""),
	}

	if fl.Tag() != "name" {
		t.Error(test.DiffMessage(fl.Tag(), "name", "Tag"))
	}
	if fl.NestedTag() != "user.name" {
		t.Error(test.DiffMessage(fl.NestedTag(), "user.name", "NestedTag"))
	}
	if fl.Namespace() != "User.Name" {
		t.Error(test.DiffMessage(fl.Namespace(), "User.Name", "Namespace"))
	}
	if fl.Field() != "Name" {
		t.Error(test.DiffMessage(fl.Field(), "Name", "Field"))
	}
	if fl.Index() != 2 {
		t.Error(test.DiffMessage(fl.Index(), 2, "Index"))
	}
	if fl.Value() != "john" {
		t.Error(test.DiffMessage(fl.Value(), "john", "Value"))
	}
	if !fl.IsValue() {
		t.Error(test.DiffMessage(fl.IsValue(), true, "IsValue"))
	}
	if fl.Kind() != reflect.String {
		t.Error(test.DiffMessage(fl.Kind(), reflect.String, "Kind"))
	}
	if fl.Type() != reflect.TypeOf("") {
		t.Error(test.DiffMessage(fl.Type(), reflect.TypeOf(""), "Type"))
	}
}

func TestFieldLevel_ZeroValue(t *testing.T) {
	fl := &FieldLevel{}
	if fl.IsValue() {
		t.Error(test.DiffMessage(fl.IsValue(), false, "zero value FieldLevel should have IsValue false"))
	}
	if fl.Value() != nil {
		t.Error(test.DiffMessage(fl.Value(), nil, "zero value FieldLevel should have nil Value"))
	}
}
