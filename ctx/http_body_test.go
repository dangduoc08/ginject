package ctx

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
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

	if body.Get("information.children.father.fullname") != nil {
		t.Errorf("key information.children.father.fullname = %v, should be %v", body.Get("information.children.father.fullname"), nil)
	}
}

func TestBodySet_OverwritesExistingMapValue(t *testing.T) {
	b := Body{"information": map[string]any{"fullname": "John"}}
	b.Set("information", "newValue")

	if b.Get("information") != "newValue" {
		t.Error(test.DiffMessage(b.Get("information"), "newValue", "Set should overwrite a key even when its existing value is a nested map"))
	}
}

func TestBodyGet_MissingIntermediateKeyReturnsNil(t *testing.T) {
	b := Body{"y": "sneaky", "other": 1}

	if got := b.Get("x.y"); got != nil {
		t.Error(test.DiffMessage(got, nil, "Get should return nil when an intermediate path segment does not exist, not fall through to an unrelated sibling key"))
	}
}

func TestHTTPContext_BodyJSON(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"foo":"bar"}`))
	c.Request.Header.Set("Content-Type", applicationJSON)

	b := c.Body()
	if b.Get("foo") != "bar" {
		t.Error(test.DiffMessage(b.Get("foo"), "bar", "Body should parse a JSON request body"))
	}
}

func TestHTTPContext_BodyNonJSON(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader("foo=bar"))
	c.Request.Header.Set("Content-Type", applicationXWWWFormUrlencoded)

	b := c.Body()
	if len(b) != 0 {
		t.Error(test.DiffMessage(b, Body{}, "Body should be empty when the content type is not application/json"))
	}
}

func TestHTTPContext_BodyIsCached(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"foo":"bar"}`))
	c.Request.Header.Set("Content-Type", applicationJSON)

	b1 := c.Body()
	b1.Set("foo", "mutated")
	b2 := c.Body()

	if b2.Get("foo") != "mutated" {
		t.Error(test.DiffMessage(b2.Get("foo"), "mutated", "Body should be cached across calls on the same context"))
	}
}

func TestHTTPContext_BodyPanicsOnInvalidJSON(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when the JSON body cannot be unmarshaled")
		}
	}()
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{not-json`))
	c.Request.Header.Set("Content-Type", applicationJSON)
	c.Body()
}

func TestBody_Del(t *testing.T) {
	b := Body{"foo": "bar"}
	b.Del("foo")
	if b.Has("foo") {
		t.Error(test.DiffMessage(b.Has("foo"), false, "Has should be false after Del"))
	}
}

func TestBody_Has(t *testing.T) {
	b := Body{"foo": "bar"}
	if !b.Has("foo") {
		t.Error(test.DiffMessage(b.Has("foo"), true, "Has should be true for an existing key"))
	}
	if b.Has("missing") {
		t.Error(test.DiffMessage(b.Has("missing"), false, "Has should be false for a missing key"))
	}
}

type bodyBindDTO struct {
	Name string `bind:"name"`
}

func TestBody_Bind(t *testing.T) {
	b := Body{"name": "joe"}
	result, fls := b.Bind(bodyBindDTO{})

	dto, ok := result.(bodyBindDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, bodyBindDTO{}, "Bind should return a bodyBindDTO"))
	}
	if dto.Name != "joe" {
		t.Error(test.DiffMessage(dto.Name, "joe", "Bind should populate fields from the body map"))
	}
	if len(fls) != 1 {
		t.Error(test.DiffMessage(len(fls), 1, "Bind should report a FieldLevel per bound field"))
	}
}
