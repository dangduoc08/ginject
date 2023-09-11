package context

import (
	"testing"
)

type EmbeddedDTO struct {
	Title string  `bind:"title"`
	Rank  float64 `bind:"rank"`
}

type Person struct {
	FullName string `bind:"full_name"`
}

type TestDTO struct {
	Bool1 bool `bind:"bool_1"`
	Bool2 bool `bind:"bool_2"`
	Bool3 bool `bind:"bool_3"`
	Bool4 bool `bind:"bool_4"`
	Bool5 bool `bind:"bool_5"`
	Bool6 bool `bind:"bool_6"`
	Bool7 bool `bind:"bool_6.1"`
	Bool8 bool `bind:"bool_6.2"`

	Int1 int `bind:"int_1"`
	Int2 int `bind:"int_2"`
	Int3 int `bind:"int_3"`
	Int4 int `bind:"int_4"`
	Int5 int `bind:"int_5"`
	Int6 int `bind:"int_6"`

	Name        string   `bind:"name"`
	SalaryRange []string `bind:"salary_range"`
	Persons     []Person `bind:"persons"`
	Age         int      `bind:"age"`
	EmbeddedDTO `bind:"embedded_dto"`
	Stuffs      any `bind:"stuffs"`
	Others      any `bind:"others"`
}

func TestBind(t *testing.T) {
	// 	testData := make(map[string]any)
	// 	err := json.Unmarshal([]byte(`{
	// 		"bool_1": true,
	// 		"bool_2": false,
	// 		"bool_3": "true",
	// 		"bool_4": "false",
	// 		"bool_5": "foo_bar",
	// 		"bool_6": ["true", false, "true", true, "foo_bar"],

	// 		"int_1": 1234567890,
	// 		"int_2": "1234567890",
	// 		"int_3": -1234567890,
	// 		"int_4": "-1234567890",
	// 		"int_5": [1234567890, "1234567890", -1234567890, "-1234567890"]

	// 	}`), &testData)

	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	b := &Binding{
	// 		Options: BindingOptions{
	// 			IsParseArray:  true,
	// 			IsParseString: true,
	// 		},
	// 		Structure: TestDTO{},
	// 		Data:      testData,
	// 	}

	// 	d := b.Bind()
	// 	dto := d.(TestDTO)

	// 	actual1 := dto.Bool1
	// 	expected1 := true
	// 	if actual1 != expected1 {
	// 		t.Errorf(utils.ErrorMessage(actual1, expected1, "bool should be binded"))
	// 	}

	// 	actual2 := dto.Bool2
	// 	expected2 := false
	// 	if actual2 != expected2 {
	// 		t.Errorf(utils.ErrorMessage(actual2, expected2, "bool should be binded"))
	// 	}

	// 	actual3 := dto.Bool3
	// 	expected3 := true
	// 	if actual3 != expected3 {
	// 		t.Errorf(utils.ErrorMessage(actual3, expected3, "bool string should be binded"))
	// 	}

	// 	actual4 := dto.Bool4
	// 	expected4 := false
	// 	if actual4 != expected4 {
	// 		t.Errorf(utils.ErrorMessage(actual4, expected4, "bool string should be binded"))
	// 	}

	// 	actual5 := dto.Bool5
	// 	expected5 := false
	// 	if actual5 != expected5 {
	// 		t.Errorf(utils.ErrorMessage(actual5, expected5, "bool string shoule fallback, due to not match pattern"))
	// 	}

	// 	actual6 := dto.Bool6
	// 	expected6 := true
	// 	if actual6 != expected6 {
	// 		t.Errorf(utils.ErrorMessage(actual6, expected6, "bool should bind from first element of array"))
	// 	}

	// 	actual7 := dto.Bool7
	// 	expected7 := false
	// 	if actual7 != expected7 {
	// 		t.Errorf(utils.ErrorMessage(actual7, expected7, "bool should bind from first element of array"))
	// 	}

	// actual8 := dto.Bool8
	// expected8 := true
	//
	//	if actual8 != expected8 {
	//		t.Errorf(utils.ErrorMessage(actual8, expected8, "bool should bind from first element of array"))
	//	}
}
