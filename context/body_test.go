package context

import (
	"testing"
)

func TestBodyGet(t *testing.T) {
	body := Body{
		"data": Body{
			"facebook":  "https://facebook.com",
			"instagram": "https://instagram.com",
		},
		"information": Body{
			"fullname": "John Doe jr",
			"parent": Body{
				"father": Body{
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
