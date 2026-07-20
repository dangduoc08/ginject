package ctx

import (
	"reflect"
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestToJSONBuffer_PlainData(t *testing.T) {
	b, err := toJSONBuffer(map[string]any{"foo": "bar"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"foo":"bar"}` {
		t.Error(test.DiffMessage(string(b), `{"foo":"bar"}`, "toJSONBuffer should marshal plain data as-is"))
	}
}

func TestToJSONBuffer_FormatString(t *testing.T) {
	b, err := toJSONBuffer(`{"greeting":"%s"}`, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"greeting":"hi"}` {
		t.Error(test.DiffMessage(string(b), `{"greeting":"hi"}`, "toJSONBuffer should treat a string arg as a format string for the remaining args"))
	}
}

func TestToJSONBuffer_Error(t *testing.T) {
	_, err := toJSONBuffer(make(chan int))
	if err == nil {
		t.Error("expected an error marshaling an unsupported type")
	}
}

type fnCacheDTO struct {
	Name string `bind:"name"`
	Age  int    `bind:"age"`
}

func TestGetFieldBindTags(t *testing.T) {
	typ := reflect.TypeOf(fnCacheDTO{})
	tags := getFieldBindTags(typ)

	if len(tags) != 2 {
		t.Fatalf("expected 2 field tags, got %d", len(tags))
	}
	if !tags[0].ok || tags[0].field != "name" {
		t.Error(test.DiffMessage(tags[0], fieldBindTag{ok: true, field: "name"}, "field 0 should be tagged 'name'"))
	}
	if !tags[1].ok || tags[1].field != "age" {
		t.Error(test.DiffMessage(tags[1], fieldBindTag{ok: true, field: "age"}, "field 1 should be tagged 'age'"))
	}
}

func TestGetFieldBindTags_IsCachedPerType(t *testing.T) {
	typ := reflect.TypeOf(fnCacheDTO{})
	first := getFieldBindTags(typ)
	second := getFieldBindTags(typ)

	if &first[0] != &second[0] {
		t.Error("getFieldBindTags should return the same cached slice for a repeated type")
	}
}

func TestGetFieldBindTags_ConcurrentAccess_NoDataRace(t *testing.T) {
	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			BindStruct(map[string]any{"name": "joe", "age": float64(1)}, &[]FieldLevel{}, fnCacheDTO{}, "", "")
		}()
	}

	wg.Wait()
}

func TestGetTagParamIndex_NegativeIndexFallsBackToZero(t *testing.T) {
	idx, field := GetTagParamIndex("items.-1")
	if idx != 0 || field != "items" {
		t.Error(test.DiffMessage([2]any{idx, field}, [2]any{0, "items"}, "a negative parsed index should fall back to 0"))
	}
}

func TestBindArray_TypeMismatchResetsToEmpty(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([]int{})
	result := bindArray([]any{float64(1), "not-a-number", float64(3)}, fls, typ, "", "")
	intArr, ok := result.([]int)
	if !ok {
		t.Fatal(test.DiffMessage(result, []int{}, "bindArray should return an []int"))
	}
	if len(intArr) != 0 {
		t.Error(test.DiffMessage(intArr, []int{}, "a type mismatch mid-array should reset the result to empty rather than partially fill it"))
	}
}

func TestBindArray_Interface(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([]any{})
	arr := []any{"a", float64(1), true}
	result := bindArray(arr, fls, typ, "", "")
	if r, ok := result.([]any); !ok || len(r) != 3 {
		t.Error(test.DiffMessage(result, arr, "bindArray should pass interface slices through unchanged"))
	}
}

func TestBindArray_NestedSlice(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([][]int{})
	arr := []any{
		[]any{float64(1), float64(2)},
		[]any{float64(3)},
	}
	result := bindArray(arr, fls, typ, "", "")
	nested, ok := result.([][]int)
	if !ok {
		t.Fatal(test.DiffMessage(result, [][]int{}, "bindArray should return a [][]int for a nested slice"))
	}
	if len(nested) != 2 || len(nested[0]) != 2 || nested[0][0] != 1 || nested[1][0] != 3 {
		t.Error(test.DiffMessage(nested, [][]int{{1, 2}, {3}}, "bindArray should recursively bind nested slice dimensions"))
	}
}

type fnArrStructDTO struct {
	Name string `bind:"name"`
}

func TestBindArray_SliceOfStruct(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([]fnArrStructDTO{})
	arr := []any{
		map[string]any{"name": "joe"},
		map[string]any{"name": "ana"},
	}
	result := bindArray(arr, fls, typ, "", "")
	structs, ok := result.([]fnArrStructDTO)
	if !ok || len(structs) != 2 {
		t.Fatal(test.DiffMessage(result, []fnArrStructDTO{}, "bindArray should return a slice of bound structs"))
	}
	if structs[0].Name != "joe" || structs[1].Name != "ana" {
		t.Error(test.DiffMessage(structs, []fnArrStructDTO{{Name: "joe"}, {Name: "ana"}}, "bindArray should bind each struct element from its map"))
	}
}

func TestBindArray_SliceOfMap(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([]map[string]int{})
	arr := []any{
		map[string]any{"a": float64(1)},
	}
	result := bindArray(arr, fls, typ, "", "")
	maps, ok := result.([]map[string]int)
	if !ok || len(maps) != 1 || maps[0]["a"] != 1 {
		t.Error(test.DiffMessage(result, []map[string]int{{"a": 1}}, "bindArray should return a slice of bound maps"))
	}
}

func TestBindArray_UnsupportedKindReturnsNil(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([]func(){})
	result := bindArray([]any{}, fls, typ, "", "")
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "bindArray should return nil for an unsupported element kind"))
	}
}

func TestBindMap_Bool(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]bool{})
	result := bindMap(map[string]any{"a": true}, fls, typ, "", "")
	m, ok := result.(map[string]bool)
	if !ok || !m["a"] {
		t.Error(test.DiffMessage(result, map[string]bool{"a": true}, "bindMap should bind a bool map"))
	}
}

func TestBindMap_Int(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]int{})
	result := bindMap(map[string]any{"a": float64(42)}, fls, typ, "", "")
	m, ok := result.(map[string]int)
	if !ok || m["a"] != 42 {
		t.Error(test.DiffMessage(result, map[string]int{"a": 42}, "bindMap should bind an int map"))
	}
}

func TestBindMap_TypeMismatchResetsToEmpty(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]int{})
	result := bindMap(map[string]any{"a": float64(1), "b": "not-a-number"}, fls, typ, "", "")
	m, ok := result.(map[string]int)
	if !ok {
		t.Fatal(test.DiffMessage(result, map[string]int{}, "bindMap should return a map[string]int"))
	}
	if len(m) != 0 {
		t.Error(test.DiffMessage(m, map[string]int{}, "a type mismatch anywhere in the map should reset the result to empty"))
	}
}

func TestBindMap_Float64(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]float64{})
	result := bindMap(map[string]any{"a": float64(1.5)}, fls, typ, "", "")
	m, ok := result.(map[string]float64)
	if !ok || m["a"] != 1.5 {
		t.Error(test.DiffMessage(result, map[string]float64{"a": 1.5}, "bindMap should bind a float64 map"))
	}
}

func TestBindMap_Complex64(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]complex64{})
	result := bindMap(map[string]any{"a": float64(2)}, fls, typ, "", "")
	m, ok := result.(map[string]complex64)
	if !ok || m["a"] != complex64(complex(2, 0)) {
		t.Error(test.DiffMessage(result, map[string]complex64{"a": complex(2, 0)}, "bindMap should bind a complex64 map"))
	}
}

func TestBindMap_String(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]string{})
	result := bindMap(map[string]any{"a": "hi"}, fls, typ, "", "")
	m, ok := result.(map[string]string)
	if !ok || m["a"] != "hi" {
		t.Error(test.DiffMessage(result, map[string]string{"a": "hi"}, "bindMap should bind a string map"))
	}
}

func TestBindMap_Interface(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]any{})
	obj := map[string]any{"a": "hi"}
	result := bindMap(obj, fls, typ, "", "")
	m, ok := result.(map[string]any)
	if !ok || m["a"] != "hi" {
		t.Error(test.DiffMessage(result, obj, "bindMap should pass an interface-valued map through unchanged"))
	}
}

func TestBindMap_Slice(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string][]int{})
	obj := map[string]any{"a": []any{float64(1), float64(2)}}
	result := bindMap(obj, fls, typ, "", "")
	m, ok := result.(map[string][]int)
	if !ok || len(m["a"]) != 2 || m["a"][0] != 1 {
		t.Error(test.DiffMessage(result, map[string][]int{"a": {1, 2}}, "bindMap should bind a map of slices"))
	}
}

func TestBindMap_Struct(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]fnArrStructDTO{})
	obj := map[string]any{"a": map[string]any{"name": "joe"}}
	result := bindMap(obj, fls, typ, "", "")
	m, ok := result.(map[string]fnArrStructDTO)
	if !ok || m["a"].Name != "joe" {
		t.Error(test.DiffMessage(result, map[string]fnArrStructDTO{"a": {Name: "joe"}}, "bindMap should bind a map of structs"))
	}
}

func TestBindMap_NestedMapOfBool(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]map[string]bool{})
	obj := map[string]any{"a": map[string]any{"x": true}}
	result := bindMap(obj, fls, typ, "", "")
	m, ok := result.(map[string]map[string]bool)
	if !ok || !m["a"]["x"] {
		t.Error(test.DiffMessage(result, map[string]map[string]bool{"a": {"x": true}}, "bindMap should bind a nested map of maps"))
	}
}

func TestBindMap_NestedMapOfSlice(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]map[string][]int{})
	obj := map[string]any{"a": map[string]any{"x": []any{float64(1)}}}
	result := bindMap(obj, fls, typ, "", "")
	m, ok := result.(map[string]map[string][]int)
	if !ok || len(m["a"]["x"]) != 1 || m["a"]["x"][0] != 1 {
		t.Error(test.DiffMessage(result, map[string]map[string][]int{"a": {"x": {1}}}, "bindMap should bind a nested map of slices"))
	}
}

func TestBindMap_NestedMapOfStruct(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]map[string]fnArrStructDTO{})
	obj := map[string]any{"a": map[string]any{"x": map[string]any{"name": "joe"}}}
	result := bindMap(obj, fls, typ, "", "")
	m, ok := result.(map[string]map[string]fnArrStructDTO)
	if !ok || m["a"]["x"].Name != "joe" {
		t.Error(test.DiffMessage(result, map[string]map[string]fnArrStructDTO{"a": {"x": {Name: "joe"}}}, "bindMap should bind a nested map of structs"))
	}
}

func TestBindMap_UnsupportedKindReturnsNil(t *testing.T) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]func(){})
	result := bindMap(map[string]any{}, fls, typ, "", "")
	if result != nil {
		t.Error(test.DiffMessage(result, nil, "bindMap should return nil for an unsupported element kind"))
	}
}
