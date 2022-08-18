package routing

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

func TestInsert(test *testing.T) {
	tr := newTrie[any]()
	tr.insert(normalText1, nil)
	tr.insert(normalText2, nil)
	tr.insert(diffFirstCharText1, nil)

	// Test tnsert logic
	output1 := tr.node["/"]
	if output1 == nil {
		test.Errorf("tr.node[\"/\"] = %v; expect ≠ nil", output1)
	}

	output2 := tr.node["/"].node["t"]
	if output2 == nil {
		test.Errorf("tr.node[\"/\"].node[\"t\"] = %v; expect ≠ nil", output2)
	}

	output3 := tr.node["/"].node["t"].node["1"]
	if output3 == nil {
		test.Errorf("tr.node[\"/\"].node[\"t\"].node[\"1\"] = %v; expect ≠ nil", output3)
	}

	output4 := tr.node["/"].node["t"].node["2"]
	if output4 == nil {
		test.Errorf("tr.node[\"/\"].node[\"t\"].node[\"2\"] = %v; expect ≠ nil", output4)
	}

	output5 := tr.node["t"]
	if output5 == nil {
		test.Errorf("tr.node[\"/\"] = %v; expect ≠ nil", output5)
	}

	// Test isEnd logic
	output6 := tr.node["/"].node["t"].isEnd
	if output6 {
		test.Errorf("tr.node[\"/\"].node[\"t\"].isEnd = %v; expect = false", output6)
	}

	output7 := tr.node["/"].node["t"].node["2"].isEnd
	if !output7 {
		test.Errorf("tr.node[\"/\"].node[\"t\"].node[\"2\"].isEnd = %v; expect = true", output7)
	}
}

func TestLen(test *testing.T) {
	tr := newTrie[any]()
	tr.insert(normalText1, nil)
	tr.insert(normalText2, nil)
	tr.insert(diffFirstCharText1, nil)

	var expect uint = 6
	output1 := tr.len()
	if output1 != uint(expect) {
		test.Errorf("tr.len() = %v; expect = %v", output1, expect)
	}
}

func TestSearch(test *testing.T) {
	tr := newTrie[any]()
	tr.insert(normalText1, nil)
	tr.insert(normalText2, nil)
	tr.insert(diffFirstCharText1, nil)

	// Test search logic
	output1 := tr.search(normalText1)
	if !output1 {
		test.Errorf("tr.search(normalText1) = %v; expect = true", output1)
	}

	output2 := tr.search(normalText2)
	if !output2 {
		test.Errorf("tr.search(normalText2) = %v; expect = true", output2)
	}

	output3 := tr.search(diffFirstCharText1)
	if !output3 {
		test.Errorf("tr.search(diffFirstCharText1) = %v; expect = true", output3)
	}

	output4 := tr.search(strconv.Itoa(rand.Intn(100)))
	if output4 {
		test.Errorf("tr.search(tr.search(strconv.Itoa(rand.Intn(100)))) = %v; expect = false", output4)
	}
}
