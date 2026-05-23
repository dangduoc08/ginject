package ctx

import (
	"encoding/json"
	"testing"

	"github.com/dangduoc08/ginject/testutils"
)

type Address struct {
	Street  string `bind:"street"`
	City    string `bind:"city"`
	ZipCode string `bind:"zip_code"`
}

type Person struct {
	Name    string  `bind:"name"`
	Age     int     `bind:"age"`
	Address Address `bind:"address"`
	Email   string  `bind:"email"`
}

type PhoneNumber struct {
	Type  string `bind:"type"`
	Value string `bind:"value"`
}

type EmbeddedStruct struct {
	Name         string            `bind:"name"`
	Age          int               `bind:"age"`
	IsMarried    bool              `bind:"is_married"`
	PhoneNumbers []PhoneNumber     `bind:"phone_numbers"`
	Address      map[string]string `bind:"address"`
}

type TestDTO struct {
	Bool1 bool `bind:"bool_1"`
	Bool2 bool `bind:"bool_2"`

	String1 string `bind:"string_1"`
	String2 string `bind:"string_2"`

	Integer1 int   `bind:"integer_1"`
	Integer2 int8  `bind:"integer_2"`
	Integer3 int16 `bind:"integer_3"`
	Integer4 int32 `bind:"integer_4"`
	Integer5 int64 `bind:"integer_5"`

	Uinteger1 int   `bind:"uinteger_1"`
	Uinteger2 int8  `bind:"uinteger_2"`
	Uinteger3 int16 `bind:"uinteger_3"`
	Uinteger4 int32 `bind:"uinteger_4"`
	Uinteger5 int64 `bind:"uinteger_5"`

	Float32 float32 `bind:"float_32"`
	Float64 float64 `bind:"float_64"`

	Complex64  complex64  `bind:"complex_64"`
	Complex128 complex128 `bind:"complex_128"`

	BoolArray []bool `bind:"bool_array"`

	StringArray []string `bind:"string_array"`

	IntArray   []int   `bind:"int_array"`
	Int8Array  []int8  `bind:"int8_array"`
	Int16Array []int16 `bind:"int16_array"`
	Int32Array []int32 `bind:"int32_array"`
	Int64Array []int64 `bind:"int64_array"`

	UintArray   []uint   `bind:"uint_array"`
	Uint8Array  []uint8  `bind:"uint8_array"`
	Uint16Array []uint16 `bind:"uint16_array"`
	Uint32Array []uint32 `bind:"uint32_array"`
	Uint64Array []uint64 `bind:"uint64_array"`

	Float32Array []float32 `bind:"float32_array"`
	Float64Array []float64 `bind:"float64_array"`

	Complex64Array  []complex64  `bind:"complex64_array"`
	Complex128Array []complex128 `bind:"complex128_array"`

	ThreeDimensionsStringArray [][][]string `bind:"3_dimensions_string_array"`

	MapStringStringArray []map[string]string `bind:"map_string_string_array"`

	MapPerson map[string]Person `bind:"map_person"`

	NestedStructArray []Person `bind:"nested_struct_array"`

	EmbeddedStruct    EmbeddedStruct  `bind:"struct"`
	EmbeddedStructPtr *EmbeddedStruct `bind:"struct"`
}

func TestBindStruct(t *testing.T) {
	testData := make(map[string]any)
	err := json.Unmarshal([]byte(`{
		"bool_1": true,
		"bool_2": false,
		"string_1": "string 1",
		"string_2": "string 2",
		"integer_1": 9223372036854775807,
		"integer_2": -128,
		"integer_3": -32768,
		"integer_4": -2147483648,
		"integer_5": -19223372036854775808,
		"uinteger_1": 18446744073709551615,
		"uinteger_2": 255,
		"uinteger_3": 65535,
		"uinteger_4": 4294967295,
		"uinteger_5": 18446744073709551615,
		"float_32": 1.401298464e-45,
		"float_64": 1.7976931348623157e+308,
		"complex_64": 1.401298464e-45,
		"complex_128": 18446744073709551615,
		"bool_array": [
			true,
			false
		],
		"string_array": [
			"string 1",
			"string 2"
		],
		"int_array": [
			9223372036854775807
		],
		"int8_array": [
			-128
		],
		"int16_array": [
			-32768
		],
		"int32_array": [
			-2147483648
		],
		"int64_array": [
			-19223372036854775808
		],
		"uint_array": [
			18446744073709551615
		],
		"uint8_array": [
			255
		],
		"uint16_array": [
			65535
		],
		"uint32_array": [
			4294967295
		],
		"uint64_array": [
			18446744073709551615
		],
		"float32_array": [
			1.401298464e-45
		],
		"float64_array": [
			1.7976931348623157e+308
		],
		"complex64_array": [
			1.401298464e-45
		],
		"complex128_array": [
			18446744073709551615
		],
		"3_dimensions_string_array": [
			[
				[
					"string 0 0 0",
					"string 0 0 1"
				],
				[
					"string 0 1 0",
					"string 0 1 1"
				]
			],
			[
				[
					"string 1 0 0",
					"string 1 0 1"
				],
				[
					"string 1 1 0",
					"string 1 1 1"
				]
			]
		],
		"map_string_string_array": [
			{
				"name": "John Doe",
				"gender": "Male",
				"dob": "1994-08-20"
			},
			{
				"name": "Jane Doe",
				"gender": "Female",
				"dob": "1994-08-20"
			}
		],
		"nested_struct_array": [
			{
				"name": "John Doe",
				"age": 30,
				"address": {
					"street": "123 Main St",
					"city": "Anytown",
					"zip_code": "12345"
				},
				"email": "john.doe@example.com"
			},
			{
				"name": "Alice Smith",
				"age": 25,
				"address": {
					"street": "456 Elm St",
					"city": "Sometown",
					"zip_code": "54321"
				},
				"email": "alice.smith@example.com"
			}
		],
		"struct": {
			"name": "Bob Johnson",
			"age": 35,
			"is_married": true,
			"phone_numbers": [
				{
					"type": "home",
					"value": "123-456-7890"
				},
				{
					"type": "work",
					"value": "987-654-3210"
				}
			],
			"address": {
				"street": "789 Oak St",
				"city": "Villagetown",
				"zip_code": "67890"
			}
		},
		"map_person": {
			"person_1": {
				"name": "John Doe",
				"age": 30,
				"address": {
					"street": "123 Main St",
					"city": "Anytown",
					"zip_code": "12345"
				},
				"email": "john.doe@example.com"
			},
			"person_2": {
				"name": "Jane Doe",
				"age": 32,
				"address": {
					"street": "123 Main St",
					"city": "Anytown",
					"zip_code": "12345"
				},
				"email": "john.doe@example.com"
			}
		}
	}`), &testData)

	if err != nil {
		panic(err)
	}

	d, _ := BindStruct(testData, &[]FieldLevel{}, TestDTO{}, "", "")
	bindedDTO := d.(TestDTO)

	expected1 := true
	if bindedDTO.Bool1 != expected1 {
		t.Errorf("Bool1 should %v but got %v", expected1, bindedDTO.Bool1)
	}

	expected2 := false
	if bindedDTO.Bool2 != expected2 {
		t.Errorf("Bool2 should %v but got %v", expected2, bindedDTO.Bool2)
	}
}

type PtrDTO struct {
	PtrBool       *bool       `bind:"ptr_bool"`
	PtrInt        *int        `bind:"ptr_int"`
	PtrInt8       *int8       `bind:"ptr_int8"`
	PtrInt16      *int16      `bind:"ptr_int16"`
	PtrInt32      *int32      `bind:"ptr_int32"`
	PtrInt64      *int64      `bind:"ptr_int64"`
	PtrUint       *uint       `bind:"ptr_uint"`
	PtrUint8      *uint8      `bind:"ptr_uint8"`
	PtrUint16     *uint16     `bind:"ptr_uint16"`
	PtrUint32     *uint32     `bind:"ptr_uint32"`
	PtrUint64     *uint64     `bind:"ptr_uint64"`
	PtrFloat32    *float32    `bind:"ptr_float32"`
	PtrFloat64    *float64    `bind:"ptr_float64"`
	PtrComplex64  *complex64  `bind:"ptr_complex64"`
	PtrComplex128 *complex128 `bind:"ptr_complex128"`
	PtrString     *string     `bind:"ptr_string"`
	PtrStruct     *Address    `bind:"ptr_struct"`
	PtrSlice      *[]string   `bind:"ptr_slice"`
	PtrMap        *map[string]string `bind:"ptr_map"`
	PtrMissing    *string     `bind:"ptr_missing"`
}

func boolPtr(v bool) *bool             { return &v }
func intPtr(v int) *int                { return &v }
func int8Ptr(v int8) *int8             { return &v }
func int16Ptr(v int16) *int16          { return &v }
func int32Ptr(v int32) *int32          { return &v }
func int64Ptr(v int64) *int64          { return &v }
func uintPtr(v uint) *uint             { return &v }
func uint8Ptr(v uint8) *uint8          { return &v }
func uint16Ptr(v uint16) *uint16       { return &v }
func uint32Ptr(v uint32) *uint32       { return &v }
func float32Ptr(v float32) *float32    { return &v }
func float64Ptr(v float64) *float64    { return &v }
func stringPtr(v string) *string       { return &v }

func TestBindStruct_Pointers(t *testing.T) {
	testData := make(map[string]any)
	err := json.Unmarshal([]byte(`{
		"ptr_bool": true,
		"ptr_int": 42,
		"ptr_int8": -12,
		"ptr_int16": -1000,
		"ptr_int32": -100000,
		"ptr_int64": -9223372036854775808,
		"ptr_uint": 99,
		"ptr_uint8": 255,
		"ptr_uint16": 65535,
		"ptr_uint32": 4294967295,
		"ptr_uint64": 18446744073709551615,
		"ptr_float32": 3.14,
		"ptr_float64": 1.7976931348623157e+308,
		"ptr_complex64": 2.5,
		"ptr_complex128": 9.9,
		"ptr_string": "hello",
		"ptr_struct": {"street": "1 Main St", "city": "Anytown", "zip_code": "00001"},
		"ptr_slice": ["a", "b", "c"],
		"ptr_map": {"k1": "v1", "k2": "v2"}
	}`), &testData)
	if err != nil {
		t.Fatal(err)
	}

	d, _ := BindStruct(testData, &[]FieldLevel{}, PtrDTO{}, "", "")
	dto := d.(PtrDTO)

	if dto.PtrBool == nil || *dto.PtrBool != true {
		t.Error(testutils.DiffMessage(dto.PtrBool, boolPtr(true), "*bool"))
	}
	if dto.PtrInt == nil || *dto.PtrInt != 42 {
		t.Error(testutils.DiffMessage(dto.PtrInt, intPtr(42), "*int"))
	}
	if dto.PtrInt8 == nil || *dto.PtrInt8 != -12 {
		t.Error(testutils.DiffMessage(dto.PtrInt8, int8Ptr(-12), "*int8"))
	}
	if dto.PtrInt16 == nil || *dto.PtrInt16 != -1000 {
		t.Error(testutils.DiffMessage(dto.PtrInt16, int16Ptr(-1000), "*int16"))
	}
	if dto.PtrInt32 == nil || *dto.PtrInt32 != -100000 {
		t.Error(testutils.DiffMessage(dto.PtrInt32, int32Ptr(-100000), "*int32"))
	}
	if dto.PtrInt64 == nil || *dto.PtrInt64 != -9223372036854775808 {
		t.Error(testutils.DiffMessage(dto.PtrInt64, int64Ptr(-9223372036854775808), "*int64"))
	}
	if dto.PtrUint == nil || *dto.PtrUint != 99 {
		t.Error(testutils.DiffMessage(dto.PtrUint, uintPtr(99), "*uint"))
	}
	if dto.PtrUint8 == nil || *dto.PtrUint8 != 255 {
		t.Error(testutils.DiffMessage(dto.PtrUint8, uint8Ptr(255), "*uint8"))
	}
	if dto.PtrUint16 == nil || *dto.PtrUint16 != 65535 {
		t.Error(testutils.DiffMessage(dto.PtrUint16, uint16Ptr(65535), "*uint16"))
	}
	if dto.PtrUint32 == nil || *dto.PtrUint32 != 4294967295 {
		t.Error(testutils.DiffMessage(dto.PtrUint32, uint32Ptr(4294967295), "*uint32"))
	}
	if dto.PtrUint64 == nil {
		t.Error(testutils.DiffMessage(dto.PtrUint64, new(uint64), "*uint64 should be non-nil"))
	}
	if dto.PtrFloat32 == nil {
		t.Error(testutils.DiffMessage(dto.PtrFloat32, float32Ptr(3.14), "*float32"))
	}
	if dto.PtrFloat64 == nil || *dto.PtrFloat64 != 1.7976931348623157e+308 {
		t.Error(testutils.DiffMessage(dto.PtrFloat64, float64Ptr(1.7976931348623157e+308), "*float64"))
	}
	if dto.PtrComplex64 == nil || *dto.PtrComplex64 != complex64(complex(2.5, 0)) {
		t.Error(testutils.DiffMessage(dto.PtrComplex64, new(complex64), "*complex64"))
	}
	if dto.PtrComplex128 == nil || *dto.PtrComplex128 != complex(9.9, 0) {
		t.Error(testutils.DiffMessage(dto.PtrComplex128, new(complex128), "*complex128"))
	}
	if dto.PtrString == nil || *dto.PtrString != "hello" {
		t.Error(testutils.DiffMessage(dto.PtrString, stringPtr("hello"), "*string"))
	}
	if dto.PtrStruct == nil || dto.PtrStruct.City != "Anytown" || dto.PtrStruct.Street != "1 Main St" {
		t.Error(testutils.DiffMessage(dto.PtrStruct, &Address{Street: "1 Main St", City: "Anytown", ZipCode: "00001"}, "*Struct"))
	}
	if dto.PtrSlice == nil || len(*dto.PtrSlice) != 3 || (*dto.PtrSlice)[0] != "a" {
		t.Error(testutils.DiffMessage(dto.PtrSlice, &[]string{"a", "b", "c"}, "*[]string"))
	}
	if dto.PtrMap == nil {
		t.Error(testutils.DiffMessage(dto.PtrMap, &map[string]string{"k1": "v1", "k2": "v2"}, "*map"))
	} else {
		m := *dto.PtrMap
		if m["k1"] != "v1" || m["k2"] != "v2" {
			t.Error(testutils.DiffMessage(m, map[string]string{"k1": "v1", "k2": "v2"}, "*map values"))
		}
	}
	if dto.PtrMissing != nil {
		t.Error(testutils.DiffMessage(dto.PtrMissing, nil, "*string missing key should be nil"))
	}
}

func TestBindStruct_Pointers_NilOnWrongType(t *testing.T) {
	testData := make(map[string]any)
	err := json.Unmarshal([]byte(`{
		"ptr_bool": "not-a-bool",
		"ptr_int": "not-a-number",
		"ptr_string": 123,
		"ptr_struct": "not-an-object",
		"ptr_slice": "not-an-array",
		"ptr_map": "not-an-object"
	}`), &testData)
	if err != nil {
		t.Fatal(err)
	}

	d, _ := BindStruct(testData, &[]FieldLevel{}, PtrDTO{}, "", "")
	dto := d.(PtrDTO)

	if dto.PtrBool != nil {
		t.Error(testutils.DiffMessage(dto.PtrBool, nil, "*bool wrong type should be nil"))
	}
	if dto.PtrInt != nil {
		t.Error(testutils.DiffMessage(dto.PtrInt, nil, "*int wrong type should be nil"))
	}
	if dto.PtrString != nil {
		t.Error(testutils.DiffMessage(dto.PtrString, nil, "*string wrong type should be nil"))
	}
	if dto.PtrStruct != nil {
		t.Error(testutils.DiffMessage(dto.PtrStruct, nil, "*struct wrong type should be nil"))
	}
	if dto.PtrSlice != nil {
		t.Error(testutils.DiffMessage(dto.PtrSlice, nil, "*slice wrong type should be nil"))
	}
	if dto.PtrMap != nil {
		t.Error(testutils.DiffMessage(dto.PtrMap, nil, "*map wrong type should be nil"))
	}
}
