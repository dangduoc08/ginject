package config

import (
	"testing"
)

func TestIsValidKey(t *testing.T) {
	key1 := "DATABASE_URL"
	expected1 := true
	output1 := isValidKey(key1)
	if output1 != expected1 {
		t.Errorf("TestIsValidKey Case 1: Ouput %v - Expected %v", output1, expected1)
	}

	key2 := "foobar"
	expected2 := true
	output2 := isValidKey(key2)
	if output2 != expected2 {
		t.Errorf("TestIsValidKey Case 2: Ouput %v - Expected %v", output2, expected2)
	}

	key3 := "NO-WORK"
	expected3 := false
	output3 := isValidKey(key3)
	if output3 != expected3 {
		t.Errorf("TestIsValidKey Case 3: Ouput %v - Expected %v", output3, expected3)
	}

	key4 := "ÜBER"
	expected4 := false
	output4 := isValidKey(key4)
	if output4 != expected4 {
		t.Errorf("TestIsValidKey Case 4: Ouput %v - Expected %v", output4, expected4)
	}

	key5 := "2MUCH"
	expected5 := false
	output5 := isValidKey(key5)
	if output5 != expected5 {
		t.Errorf("TestIsValidKey Case 5: Ouput %v - Expected %v", output5, expected5)
	}
}

func TestFlatten(t *testing.T) {
	input1 := make(map[string]any)
	input1["bool"] = true
	input1["string"] = "string"
	input1["int"] = -10
	input1["uint"] = 10
	input1["byte"] = 'A'
	m2 := make(map[string]any)
	m2["bool"] = true
	m2["string"] = "string"
	m2["int"] = -10
	m2["uint"] = 10
	m2["byte"] = 'A'
	input1["map"] = m2
	arr := []any{"A", "B", "C"}
	input1["arr"] = arr
	p1 := make(map[string]any)
	p1["name"] = "John Doe"
	p1["age"] = 28
	p1["isMale"] = true
	p1["favoriteSubjects"] = []any{"math", "history"}
	p2 := make(map[string]any)
	p2["name"] = "Jane Doe"
	p2["age"] = 25
	p2["isMale"] = false
	p2["favoriteSubjects"] = []any{"biology", "chemistry"}
	p := []any{
		p1,
		p2,
	}
	input1["p"] = p
	var a string = "pointer"
	input1["pointer"] = &a
	input1["nil"] = nil

	expect1 := input1
	expect1["map.bool"] = input1["map"].(map[string]any)["bool"]
	expect1["map.string"] = input1["map"].(map[string]any)["string"]
	expect1["map.int"] = input1["map"].(map[string]any)["int"]
	expect1["map.uint"] = input1["map"].(map[string]any)["unit"]
	expect1["map.byte"] = input1["map"].(map[string]any)["byte"]
	expect1["arr.0"] = input1["arr"].([]any)[0]
	expect1["arr.1"] = input1["arr"].([]any)[1]
	expect1["arr.2"] = input1["arr"].([]any)[2]
	expect1["p.0"] = p1
	expect1["p.1"] = p2
	expect1["p.0.name"] = p1["name"]
	expect1["p.0.age"] = p1["age"]
	expect1["p.0.isMale"] = p1["isMale"]
	expect1["p.0.favoriteSubjects"] = p1["favoriteSubjects"]
	expect1["p.0.favoriteSubjects.0"] = p1["favoriteSubjects"].([]any)[0]
	expect1["p.0.favoriteSubjects.1"] = p1["favoriteSubjects"].([]any)[1]
	expect1["p.1.name"] = p1["name"]
	expect1["p.1.age"] = p1["age"]
	expect1["p.1.isMale"] = p1["isMale"]
	expect1["p.1.favoriteSubjects"] = p1["favoriteSubjects"]
	expect1["p.1.favoriteSubjects.0"] = p1["favoriteSubjects"].([]any)[0]
	expect1["p.1.favoriteSubjects.1"] = p1["favoriteSubjects"].([]any)[1]

	flatten(input1, make(map[string]any), "")
	output1 := input1

	if len(output1) != len(expect1) {
		t.Errorf("len(output1) = %v; len(expect1) = %v", output1, expect1)
	}

	for key := range expect1 {
		if key != "nil" && output1[key] == nil {
			t.Errorf("output1[%v] = nil", key)
		}
	}
}
