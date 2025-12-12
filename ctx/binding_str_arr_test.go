package ctx

import (
	"encoding/json"
	"testing"

	"github.com/dangduoc08/ginject/utils"
)

type TestStrArrDTO struct {
	Bool1 bool `bind:"bool_1"`
	Bool2 bool `bind:"bools_1.0"`
	Bool3 bool `bind:"bools_1.1"`

	String1 string `bind:"string_1"`
	String2 string `bind:"strings_1.1"`

	Integer1 int   `bind:"integer_1"`
	Integer2 int8  `bind:"integers_1.1"`
	Integer3 int16 `bind:"integers_1.2"`
	Integer4 int32 `bind:"integers_1.3"`
	Integer5 int64 `bind:"integers_1.4"`
	Integer6 int   `bind:"integers_1.5"`
	Integer7 int   `bind:"integers_1.100"` // test out range
	Integer8 int   `bind:"integers_1.6"`

	Uinteger1 uint   `bind:"uinteger_1"`
	Uinteger2 uint8  `bind:"uintegers_1.1"`
	Uinteger3 uint16 `bind:"uintegers_1.2"`
	Uinteger4 uint32 `bind:"uintegers_1.3"`
	Uinteger5 uint64 `bind:"uintegers_1.4"`
	Uinteger6 uint   `bind:"uintegers_1.5"`
	Uinteger7 uint   `bind:"uintegers_1.100"` // test out range
	Uinteger8 uint   `bind:"uintegers_1.6"`
	Uinteger9 uint   `bind:"uintegers_1.7"`

	Float1 float32 `bind:"float_1"`
	Float2 float64 `bind:"floats_1.1"`
	Float3 float32 `bind:"floats_1.5"`
	Float4 float64 `bind:"floats_1.100"` // test out range
	Float5 float64 `bind:"floats_1.6"`
	Float6 float64 `bind:"floats_1.7"`

	Complex1 complex64  `bind:"complex_1"`
	Complex2 complex128 `bind:"complexes_1.1"`
	Complex3 complex128 `bind:"complexes_1.2"`
	Complex4 complex128 `bind:"complexes_1.3"`
	Complex5 complex128 `bind:"complexes_1.4"`

	Array1 []string `bind:"array_1,limit=3"`
}

func TestBindStrArr(t *testing.T) {
	testData := make(map[string][]string)

	err := json.Unmarshal([]byte(`{
			"bool_1": [
				"TRUE"
			],
			"bools_1": [
				"true",
				"this is shouldn't boolean",
				"F"
			],
			"string_1": [
				"this is string"
			],
			"strings_1": [
				"this is string 1",
				"this is string 2"
			],
			"integer_1": [
				"-9223372036854775808"
			],
			"integers_1": [
				"9223372036854775807",
				"-128",
				"-32768",
				"-2147483648",
				"-9223372036854775808",
				"this is shouldn't integer",
				"12.10"
			],
			"uinteger_1": [
				"18446744073709551615"
			],
			"uintegers_1": [
				"18446744073709551615",
				"255",
				"65535",
				"4294967295",
				"18446744073709551615",
				"this is shouldn't unsigned integer",
				"12.10",
				"-128"
			],
			"float_1": [
				"1.401298464e-45"
			],
			"floats_1": [
				"1.401298464e-45",
				"1.7976931348623157e+308",
				"255",
				"65535",
				"4294967295",
				"18446744073709551615",
				"this is shouldn't float",
				"-128",
				"12.10"
			],
			"complex_1": [
				"5 + 10i"
			],
			"complexes_1": [
				"5 + 10i",
				"21.20 + 21i",
				"this is shouldn't complex",
				"-123.14",
				"18446744073709551615"
			],
			"array_1": [
				"TRUE",
				"this is string",
				"-9223372036854775808",
				"-128",
				"-32768",
				"-2147483648",
				"18446744073709551615",
				"255",
				"65535",
				"4294967295",
				"18446744073709551615",
				"1.401298464e-45",
				"1.7976931348623157e+308",
				"5 + 10i",
				"21.20 + 21i"
			]
		}`), &testData)

	if err != nil {
		panic(err)
	}

	d, fls := BindStrArr(testData, &[]FieldLevel{}, TestStrArrDTO{})
	dto := d.(TestStrArrDTO)

	for _, fl := range fls {
		if fl.Field() == "Complex1" && dto.Complex1 != fl.Value() {
			t.Error(utils.ErrorMessage(dto.Complex1, fl.Value(), "Complex1 should be binded"))
		}
	}

	actual1 := dto.Bool1
	expected1 := true
	if actual1 != expected1 {
		t.Error(utils.ErrorMessage(actual1, expected1, "bool should be binded"))
	}

	actual2 := dto.Bool2
	expected2 := true
	if actual2 != expected2 {
		t.Error(utils.ErrorMessage(actual2, expected2, "bool should be binded"))
	}

	actual3 := dto.Bool3
	expected3 := false
	if actual3 != expected3 {
		t.Error(utils.ErrorMessage(actual3, expected3, "bool should be binded"))
	}

	actual4 := dto.String1
	expected4 := "this is string"
	if actual4 != expected4 {
		t.Error(utils.ErrorMessage(actual4, expected4, "string should be binded"))
	}

	actual5 := dto.String2
	expected5 := "this is string 2"
	if actual5 != expected5 {
		t.Error(utils.ErrorMessage(actual5, expected5, "string should be binded"))
	}

	actual6 := dto.Integer1
	expected6 := -9223372036854775808
	if actual6 != expected6 {
		t.Error(utils.ErrorMessage(actual6, expected6, "integer should be binded"))
	}

	actual7 := dto.Integer2
	var expected7 int8 = -128
	if actual7 != expected7 {
		t.Error(utils.ErrorMessage(actual7, expected7, "integer should be binded"))
	}

	actual8 := dto.Integer3
	var expected8 int16 = -32768
	if actual8 != expected8 {
		t.Error(utils.ErrorMessage(actual8, expected8, "integer should be binded"))
	}

	actual9 := dto.Integer4
	var expected9 int32 = -2147483648
	if actual9 != expected9 {
		t.Error(utils.ErrorMessage(actual9, expected9, "integer should be binded"))
	}

	actual10 := dto.Integer5
	var expected10 int64 = -9223372036854775808
	if actual10 != expected10 {
		t.Error(utils.ErrorMessage(actual10, expected10, "integer should be binded"))
	}

	actual11 := dto.Integer6
	expected11 := 0
	if actual11 != expected11 {
		t.Error(utils.ErrorMessage(actual11, expected11, "integer shouldn't be binded"))
	}

	actual12 := dto.Integer7
	expected12 := 0
	if actual12 != expected12 {
		t.Error(utils.ErrorMessage(actual12, expected12, "integer shouldn't be binded"))
	}

	actual13 := dto.Integer8
	expected13 := 0
	if actual13 != expected13 {
		t.Error(utils.ErrorMessage(actual13, expected13, "integer shouldn't be binded"))
	}

	actual14 := dto.Uinteger1
	var expected14 uint = 18446744073709551615
	if actual14 != expected14 {
		t.Error(utils.ErrorMessage(actual14, expected14, "unsigned integer should be binded"))
	}

	actual15 := dto.Uinteger2
	var expected15 uint8 = 255
	if actual15 != expected15 {
		t.Error(utils.ErrorMessage(actual15, expected15, "unsigned integer should be binded"))
	}

	actual16 := dto.Uinteger3
	var expected16 uint16 = 65535
	if actual16 != expected16 {
		t.Error(utils.ErrorMessage(actual16, expected16, "unsigned integer should be binded"))
	}

	actual17 := dto.Uinteger4
	var expected17 uint32 = 4294967295
	if actual17 != expected17 {
		t.Error(utils.ErrorMessage(actual17, expected17, "unsigned integer should be binded"))
	}

	actual18 := dto.Uinteger5
	var expected18 uint64 = 18446744073709551615
	if actual18 != expected18 {
		t.Error(utils.ErrorMessage(actual18, expected18, "unsigned integer should be binded"))
	}

	actual19 := dto.Uinteger6
	var expected19 uint = 0
	if actual19 != expected19 {
		t.Error(utils.ErrorMessage(actual19, expected19, "unsigned integer shouldn't be binded"))
	}

	actual20 := dto.Uinteger7
	var expected20 uint = 0
	if actual20 != expected20 {
		t.Error(utils.ErrorMessage(actual20, expected20, "unsigned integer shouldn't be binded"))
	}

	actual21 := dto.Uinteger8
	var expected21 uint = 0
	if actual21 != expected21 {
		t.Error(utils.ErrorMessage(actual21, expected21, "unsigned integer shouldn't be binded"))
	}

	actual22 := dto.Uinteger9
	var expected22 uint = 0
	if actual22 != expected22 {
		t.Error(utils.ErrorMessage(actual22, expected22, "unsigned integer shouldn't be binded"))
	}

	actual23 := dto.Float1
	var expected23 float32 = 1.401298464e-45
	if actual23 != expected23 {
		t.Error(utils.ErrorMessage(actual23, expected23, "float should be binded"))
	}

	actual24 := dto.Float2
	var expected24 float64 = 1.7976931348623157e+308
	if actual24 != expected24 {
		t.Error(utils.ErrorMessage(actual24, expected24, "float should be binded"))
	}

	actual25 := dto.Float3
	var expected25 float32 = 18446744073709551615
	if actual25 != expected25 {
		t.Error(utils.ErrorMessage(actual25, expected25, "float should be binded"))
	}

	actual26 := dto.Float4
	var expected26 float64 = 0
	if actual26 != expected26 {
		t.Error(utils.ErrorMessage(actual26, expected26, "float shouldn't be binded"))
	}

	actual27 := dto.Float5
	var expected27 float64 = 0
	if actual27 != expected27 {
		t.Error(utils.ErrorMessage(actual27, expected27, "float shouldn't be binded"))
	}

	actual28 := dto.Float6
	var expected28 float64 = -128
	if actual28 != expected28 {
		t.Error(utils.ErrorMessage(actual28, expected28, "float shouldn be binded"))
	}

	actual29 := dto.Complex1
	var expected29 complex64 = 5 + 10i
	if actual29 != expected29 {
		t.Error(utils.ErrorMessage(actual29, expected29, "complex shouldn be binded"))
	}

	actual30 := dto.Complex2
	var expected30 complex128 = 21.20 + 21i
	if actual30 != expected30 {
		t.Error(utils.ErrorMessage(actual30, expected30, "complex shouldn be binded"))
	}

	actual31 := dto.Complex3
	var expected31 complex128 = 0 + 0i
	if actual31 != expected31 {
		t.Error(utils.ErrorMessage(actual31, expected31, "complex shouldn be binded"))
	}

	actual32 := dto.Complex4
	var expected32 complex128 = -123.14 + 0i
	if actual32 != expected32 {
		t.Error(utils.ErrorMessage(actual32, expected32, "complex shouldn be binded"))
	}

	actual33 := dto.Complex5
	var expected33 complex128 = 1.8446744073709552e+19 + 0i
	if actual33 != expected33 {
		t.Error(utils.ErrorMessage(actual33, expected33, "complex shouldn be binded"))
	}
}
