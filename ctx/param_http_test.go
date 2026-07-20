package ctx

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestHTTPContext_Param(t *testing.T) {
	c := newTestHTTPContext()
	c.ParamKeys = map[string][]int{"id": {0}, "name": {1}}
	c.ParamValues = []string{"42", "joe"}

	p := c.Param()
	if p.Get("id") != "42" {
		t.Error(test.DiffMessage(p.Get("id"), "42", "Param should map ParamKeys indexes into ParamValues"))
	}
	if p.Get("name") != "joe" {
		t.Error(test.DiffMessage(p.Get("name"), "joe", "Param should map ParamKeys indexes into ParamValues"))
	}
}

func TestHTTPContext_ParamIsCached(t *testing.T) {
	c := newTestHTTPContext()
	c.ParamKeys = map[string][]int{"id": {0}}
	c.ParamValues = []string{"42"}

	p1 := c.Param()
	p1.Set("id", "mutated")
	p2 := c.Param()

	if p2.Get("id") != "mutated" {
		t.Error(test.DiffMessage(p2.Get("id"), "mutated", "Param should be cached across calls on the same context"))
	}
}

func TestParam_GetOnNilMap(t *testing.T) {
	var p Param
	if p.Get("missing") != "" {
		t.Error(test.DiffMessage(p.Get("missing"), "", "Get on a nil Param should return empty string"))
	}
}

func TestParam_GetMissingKey(t *testing.T) {
	p := Param{}
	if p.Get("missing") != "" {
		t.Error(test.DiffMessage(p.Get("missing"), "", "Get on missing key should return empty string"))
	}
}

func TestParam_SetAddDelHas(t *testing.T) {
	p := Param{}
	p.Set("id", "1")
	if p.Get("id") != "1" {
		t.Error(test.DiffMessage(p.Get("id"), "1", "Set should store the value"))
	}

	p.Add("id", "2")
	if got := p["id"]; len(got) != 2 || got[1] != "2" {
		t.Error(test.DiffMessage(got, []string{"1", "2"}, "Add should append to the existing values"))
	}

	if !p.Has("id") {
		t.Error(test.DiffMessage(p.Has("id"), true, "Has should be true after Set"))
	}

	p.Del("id")
	if p.Has("id") {
		t.Error(test.DiffMessage(p.Has("id"), false, "Has should be false after Del"))
	}
}

type paramBindDTO struct {
	ID string `bind:"id"`
}

func TestParam_Bind(t *testing.T) {
	p := Param{"id": {"42"}}
	result, fls := p.Bind(paramBindDTO{})

	dto, ok := result.(paramBindDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, paramBindDTO{}, "Bind should return a paramBindDTO"))
	}
	if dto.ID != "42" {
		t.Error(test.DiffMessage(dto.ID, "42", "Bind should populate fields from param values"))
	}
	if len(fls) != 1 {
		t.Error(test.DiffMessage(len(fls), 1, "Bind should report a FieldLevel per bound field"))
	}
}
