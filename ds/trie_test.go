package ds

import (
	"math/rand"
	"strconv"
	"testing"
)

func TestTrieInsert(test *testing.T) {
	var (
		normalText1        = "/t1"
		normalText2        = "/t2"
		diffFirstCharText1 = "t1"
	)

	trieInstance := NewTrie()
	trieInstance.Insert(normalText1, -1)
	trieInstance.Insert(normalText2, -1)
	trieInstance.Insert(diffFirstCharText1, -1)

	// Test tnsert logic
	output1 := trieInstance.Root["/"]
	if output1 == nil {
		test.Errorf("trieInstance.Root[\"/\"] = %v; expect ≠ nil", output1)
	}

	output2 := trieInstance.Root["/"].Root["t"]
	if output2 == nil {
		test.Errorf("trieInstance.Root[\"/\"].Root[\"t\"] = %v; expect ≠ nil", output2)
	}

	output3 := trieInstance.Root["/"].Root["t"].Root["1"]
	if output3 == nil {
		test.Errorf("trieInstance.Root[\"/\"].Root[\"t\"].Root[\"1\"] = %v; expect ≠ nil", output3)
	}

	output4 := trieInstance.Root["/"].Root["t"].Root["2"]
	if output4 == nil {
		test.Errorf("trieInstance.Root[\"/\"].Root[\"t\"].Root[\"2\"] = %v; expect ≠ nil", output4)
	}

	output5 := trieInstance.Root["t"]
	if output5 == nil {
		test.Errorf("trieInstance.Root[\"/\"] = %v; expect ≠ nil", output5)
	}

	// Test isEnd logic
	output6 := trieInstance.Root["/"].Root["t"].isEnd
	if output6 {
		test.Errorf("trieInstance.Root[\"/\"].Root[\"t\"].isEnd = %v; expect = false", output6)
	}

	output7 := trieInstance.Root["/"].Root["t"].Root["2"].isEnd
	if !output7 {
		test.Errorf("trieInstance.Root[\"/\"].Root[\"t\"].Root[\"2\"].isEnd = %v; expect = true", output7)
	}
}

func TestTrieLen(test *testing.T) {
	var (
		normalText1        = "/t1"
		normalText2        = "/t2"
		diffFirstCharText1 = "t1"
	)

	trieInstance := NewTrie()
	trieInstance.Insert(normalText1, -1)
	trieInstance.Insert(normalText2, -1)
	trieInstance.Insert(diffFirstCharText1, -1)

	var expect uint = 6
	output1 := trieInstance.Len()
	if output1 != uint(expect) {
		test.Errorf("trieInstance.len() = %v; expect = %v", output1, expect)
	}
}

func TestTrieFind(test *testing.T) {
	var (
		normalText1        = "/t1"
		normalText2        = "/t2"
		diffFirstCharText1 = "t1"
	)

	trieInstance := NewTrie()
	trieInstance.Insert(normalText1, -1)
	trieInstance.Insert(normalText2, -1)
	trieInstance.Insert(diffFirstCharText1, -1)

	// Test find logic
	output1, _, _ := trieInstance.Find(normalText1)
	if !output1 {
		test.Errorf("trieInstance.Find(normalText1) = %v; expect = true", output1)
	}

	output2, _, _ := trieInstance.Find(normalText2)
	if !output2 {
		test.Errorf("trieInstance.Find(normalText2) = %v; expect = true", output2)
	}

	output3, _, _ := trieInstance.Find(diffFirstCharText1)
	if !output3 {
		test.Errorf("trieInstance.Find(diffFirstCharText1) = %v; expect = true", output3)
	}

	output4, _, _ := trieInstance.Find(strconv.Itoa(rand.Intn(100)))
	if output4 {
		test.Errorf("trieInstance.Find(trieInstance.Find(strconv.Itoa(rand.Intn(100)))) = %v; expect = false", output4)
	}
}
