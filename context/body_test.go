package context

import (
	"testing"
)

func TestBodySet(t *testing.T) {
	body := Body{
		"data": map[string]any{
			"facebook":  "https://facebook.com",
			"instagram": "https://instagram.com",
		},
		"information": map[string]any{
			"fullname": "John Doe jr",
			"parent": map[string]any{
				"father": map[string]any{
					"age":      70,
					"fullname": "John Doe",
				},
			},
		},
	}

	key1 := "information.parent2.father.spouse"
	val1 := "Jane Doe 1"
	body.Set(key1, val1)
	if body.Get(key1).(string) != val1 {
		t.Errorf("%v = %v, should be %v", key1, body.Get(key1), val1)
	}

	key2 := "information2.parent.father.spouse"
	val2 := "Jane Doe 2"
	body.Set(key2, val2)
	if body.Get(key2).(string) != val2 {
		t.Errorf("%v = %v, should be %v", key2, body.Get(key2), val2)
	}

	key3 := "information.parent2.father.spouse2"
	val3 := "Jane Doe 3"
	body.Set(key3, val3)
	if body.Get(key3).(string) != val3 {
		t.Errorf("%v = %v, should be %v", key3, body.Get(key3), val3)
	}
}

func TestBodyGet(t *testing.T) {
	body := Body{
		"data": map[string]any{
			"facebook":  "https://facebook.com",
			"instagram": "https://instagram.com",
		},
		"information": map[string]any{
			"fullname": "John Doe jr",
			"parent": map[string]any{
				"father": map[string]any{
					"age":      70,
					"fullname": "John Doe",
				},
			},
		},
	}

	if body.Get("information.parent.father.age").(int) != 70 {
		t.Errorf("key information.parent.father.age = %v, should be %v", body.Get("information.parent.father.age"), 70)
	}

	if body.Get("information.children.father.fullname") == nil {
		t.Errorf("key information.children.father.fullname = %v, should be %v", body.Get("information.children.father.fullname"), nil)
	}
}
