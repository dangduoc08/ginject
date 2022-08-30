package data_structure

import (
	"math/rand"
	"strconv"
	"testing"
)

const (
	normalText1        = "/t1"
	normalText2        = "/t2"
	diffFirstCharText1 = "t1"
)

func TestTrieInsert(test *testing.T) {
	tr := NewTrie()
	tr.Insert(normalText1, -1)
	tr.Insert(normalText2, -1)
	tr.Insert(diffFirstCharText1, -1)

	// Test tnsert logic
	output1 := tr.Node["/"]
	if output1 == nil {
		test.Errorf("tr.Node[\"/\"] = %v; expect ≠ nil", output1)
	}

	output2 := tr.Node["/"].Node["t"]
	if output2 == nil {
		test.Errorf("tr.Node[\"/\"].Node[\"t\"] = %v; expect ≠ nil", output2)
	}

	output3 := tr.Node["/"].Node["t"].Node["1"]
	if output3 == nil {
		test.Errorf("tr.Node[\"/\"].Node[\"t\"].Node[\"1\"] = %v; expect ≠ nil", output3)
	}

	output4 := tr.Node["/"].Node["t"].Node["2"]
	if output4 == nil {
		test.Errorf("tr.Node[\"/\"].Node[\"t\"].Node[\"2\"] = %v; expect ≠ nil", output4)
	}

	output5 := tr.Node["t"]
	if output5 == nil {
		test.Errorf("tr.Node[\"/\"] = %v; expect ≠ nil", output5)
	}

	// Test IsEnd logic
	output6 := tr.Node["/"].Node["t"].IsEnd
	if output6 {
		test.Errorf("tr.Node[\"/\"].Node[\"t\"].IsEnd = %v; expect = false", output6)
	}

	output7 := tr.Node["/"].Node["t"].Node["2"].IsEnd
	if !output7 {
		test.Errorf("tr.Node[\"/\"].Node[\"t\"].Node[\"2\"].IsEnd = %v; expect = true", output7)
	}
}

func TestTrieLen(test *testing.T) {
	tr := NewTrie()
	tr.Insert(normalText1, -1)
	tr.Insert(normalText2, -1)
	tr.Insert(diffFirstCharText1, -1)

	var expect uint = 6
	output1 := tr.Len()
	if output1 != uint(expect) {
		test.Errorf("tr.len() = %v; expect = %v", output1, expect)
	}
}

func TestTrieFind(test *testing.T) {
	tr := NewTrie()
	tr.Insert(normalText1, -1)
	tr.Insert(normalText2, -1)
	tr.Insert(diffFirstCharText1, -1)

	// Test find logic
	output1, _, _ := tr.Find(normalText1)
	if !output1 {
		test.Errorf("tr.Find(normalText1) = %v; expect = true", output1)
	}

	output2, _, _ := tr.Find(normalText2)
	if !output2 {
		test.Errorf("tr.Find(normalText2) = %v; expect = true", output2)
	}

	output3, _, _ := tr.Find(diffFirstCharText1)
	if !output3 {
		test.Errorf("tr.Find(diffFirstCharText1) = %v; expect = true", output3)
	}

	output4, _, _ := tr.Find(strconv.Itoa(rand.Intn(100)))
	if output4 {
		test.Errorf("tr.Find(tr.Find(strconv.Itoa(rand.Intn(100)))) = %v; expect = false", output4)
	}
}
