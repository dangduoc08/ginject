package ctx

import (
	"encoding/json"
	"fmt"
	"testing"
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

	NestedStructArray []Person `bind:"nested_struct_array"`

	EmbeddedStruct EmbeddedStruct `bind:"struct"`
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
		}
	}`), &testData)

	if err != nil {
		panic(err)
	}

	d := BindStruct(testData, TestDTO{})
	bindedDTO := d.(TestDTO)

	expected1 := true
	if bindedDTO.Bool1 != expected1 {
		t.Errorf("Bool1 should %v but got %v", expected1, bindedDTO.Bool1)
	}

	expected2 := false
	if bindedDTO.Bool2 != expected2 {
		t.Errorf("Bool2 should %v but got %v", expected2, bindedDTO.Bool2)
	}

	fmt.Println(bindedDTO.Complex128 == 18446744073709551615)
}
