package context

import (
	"encoding/json"
	"fmt"
	"testing"
)

type EmbeddedStruct struct {
	Bool1 bool `bind:"bool_1"`
	Bool2 bool `bind:"bool_2"`
	Bool3 bool `bind:"bool_3"`

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

	Float1 float32 `bind:"float_1"`
	Float2 float64 `bind:"float_2"`

	Complex1 complex64  `bind:"complex_1"`
	Complex2 complex128 `bind:"complex_2"`

	BoolArray []string `bind:"array_1"`
}

type TestDTO struct {
	// Bool1 bool `bind:"bool_1"`
	// Bool3 bool `bind:"bool_2"`

	// String1 string `bind:"string_1"`
	// String2 string `bind:"string_2"`

	// Integer1 int   `bind:"integer_1"`
	// Integer2 int8  `bind:"integer_2"`
	// Integer3 int16 `bind:"integer_3"`
	// Integer4 int32 `bind:"integer_4"`
	// Integer5 int64 `bind:"integer_5"`

	// Uinteger1 int   `bind:"uinteger_1"`
	// Uinteger2 int8  `bind:"uinteger_2"`
	// Uinteger3 int16 `bind:"uinteger_3"`
	// Uinteger4 int32 `bind:"uinteger_4"`
	// Uinteger5 int64 `bind:"uinteger_5"`

	// Float1 float32 `bind:"float_1"`
	// Float2 float64 `bind:"float_2"`

	// Complex1 complex64  `bind:"complex_1"`
	// Complex2 complex128 `bind:"complex_2"`

	// BoolArray []bool `bind:"bool_array"`

	// StringArray []string `bind:"string_array"`

	// IntArray   []int   `bind:"int_array"`
	// Int8Array  []int8  `bind:"int8_array"`
	// Int16Array []int16 `bind:"int16_array"`
	// Int32Array []int32 `bind:"int32_array"`
	// Int64Array []int64 `bind:"int64_array"`

	// UintArray   []uint   `bind:"uint_array"`
	// Uint8Array  []uint8  `bind:"uint8_array"`
	// Uint16Array []uint16 `bind:"uint16_array"`
	// Uint32Array []uint32 `bind:"uint32_array"`
	// Uint64Array []uint64 `bind:"uint64_array"`

	// Float32Array []float32 `bind:"float32_array"`
	// Float64Array []float64 `bind:"float64_array"`

	// Complex64Array  []complex64  `bind:"complex64_array"`
	// Complex128Array []complex128 `bind:"complex128_array"`

	// EmbeddedStructMap map[string]EmbeddedStruct `bind:"embedded_struct"`

	// MultiDimension [][][][]float32 `bind:"multi_dimension"`

	// Object map[string][][][][]bool `bind:"object_2"`

	// Object2 map[string][]int `bind:"object_3"`

	Arr []map[string]EmbeddedStruct `bind:"arr"`
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
		"float_1": 1.401298464e-45,
		"float_2": 1.7976931348623157e+308,
		"complex_1": 1.401298464e-45,
		"complex_2": 18446744073709551615,
		"bool_array": [
			true,
			false
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
		"string_array_array": [
			[
				"string 0 0",
				"string 0 1"
			],
			[
				"string 1 0",
				"string 1 1"
			]
		],
		"embedded_struct": {
			"embedded_struct": {
				"bool_1": true
			}
		},
		"object_array": [
			[
				[
					{
						"bool_1": true,
						"integer_1": 678
					},
					{
						"bool_1": false,
						"integer_1": 987
					}
				]
			],
			[
				[
					{
						"bool_1": false,
						"integer_1": 123
					},
					{
						"bool_1": true,
						"integer_1": 456
					}
				]
			]
		],
		"object": {
			"sub_key_1": {
				"bool_1": true,
				"integer_1": 9223372036854775807
			},
			"sub_key_22": true,
			"sub_key_2": {
				"bool_2": "false",
				"integer_1": 9223372036854775807
			}
		},
		"multi_dimension": [
			[
				[
					[
						1000,
						2000,
						3000,
						4000
					],
					[
						4001,
						5001,
						6001,
						7001
					]
				],
				[
					[
						1010,
						2010,
						3010,
						4010
					],
					[
						4011,
						5011,
						6011,
						7011
					]
				]
			],
			[
				[
					[
						1100,
						2100,
						3100,
						4100
					],
					[
						4101,
						5101,
						6101,
						7101
					]
				],
				[
					[
						1110,
						2110,
						3110,
						4110
					],
					[
						4111,
						5111,
						6111,
						7111
					]
				]
			]
		],
		"object_2": {
			"multi_dimension": [
				[
					[
						[
							1000,
							2000,
							3000,
							4000
						],
						[
							4001,
							5001,
							6001,
							7001
						]
					],
					[
						[
							1010,
							2010,
							3010,
							4010
						],
						[
							4011,
							5011,
							6011,
							7011
						]
					]
				],
				[
					[
						[
							1100,
							2100,
							3100,
							4100
						],
						[
							4101,
							5101,
							6101,
							7101
						]
					],
					[
						[
							1110,
							2110,
							3110,
							4110
						],
						[
							4111,
							5111,
							6111,
							7111
						]
					]
				]
			]
		},
		"object_3": {
			"multi_dimension": [
				1000,
				2000,
				3000,
				4000
			]
		},
		"arr": [
			{
				"cc": {
					"bool_1": true,
					"bool_2": true,
					"bool_3": true
				}
			},
			{
				"cc": {
					"bool_1": true,
					"bool_2": true,
					"bool_3": true
				}
			}
		]
	}`), &testData)

	if err != nil {
		panic(err)
	}

	d := BindStruct(testData, TestDTO{})
	dto := d.(TestDTO)

	fmt.Println("Dto", dto.Arr[0]["cc"].Bool1)
}
