package ctx

import (
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestHTTPContext_Query(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/?foo=bar&foo=baz", nil)

	q := c.Query()
	if q.Get("foo") != "bar" {
		t.Error(test.DiffMessage(q.Get("foo"), "bar", "Query should parse the request URL query string"))
	}
}

func TestHTTPContext_QueryIsCached(t *testing.T) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/?foo=bar", nil)

	q1 := c.Query()
	q1.Set("foo", "mutated")
	q2 := c.Query()

	if q2.Get("foo") != "mutated" {
		t.Error(test.DiffMessage(q2.Get("foo"), "mutated", "Query should be cached across calls on the same context"))
	}
}

func TestQuery_GetMissingKey(t *testing.T) {
	q := Query{}
	if q.Get("missing") != "" {
		t.Error(test.DiffMessage(q.Get("missing"), "", "Get on missing key should return empty string"))
	}
}

func TestQuery_SetAddDelHas(t *testing.T) {
	q := Query{}
	q.Set("a", "1")
	if q.Get("a") != "1" {
		t.Error(test.DiffMessage(q.Get("a"), "1", "Set should store the value"))
	}

	q.Add("a", "2")
	if len(q["a"]) != 2 || q["a"][1] != "2" {
		t.Error(test.DiffMessage(q["a"], []string{"1", "2"}, "Add should append to the existing values"))
	}

	if !q.Has("a") {
		t.Error(test.DiffMessage(q.Has("a"), true, "Has should be true after Set"))
	}

	q.Del("a")
	if q.Has("a") {
		t.Error(test.DiffMessage(q.Has("a"), false, "Has should be false after Del"))
	}
}

type queryBindDTO struct {
	Name string `bind:"name"`
	Age  int    `bind:"age"`
}

func TestQuery_Bind(t *testing.T) {
	q := Query{"name": {"joe"}, "age": {"30"}}
	result, fls := q.Bind(queryBindDTO{})

	dto, ok := result.(queryBindDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, queryBindDTO{}, "Bind should return a queryBindDTO"))
	}
	if dto.Name != "joe" || dto.Age != 30 {
		t.Error(test.DiffMessage(dto, queryBindDTO{Name: "joe", Age: 30}, "Bind should populate fields from query values"))
	}
	if len(fls) != 2 {
		t.Error(test.DiffMessage(len(fls), 2, "Bind should report a FieldLevel per bound field"))
	}
}
