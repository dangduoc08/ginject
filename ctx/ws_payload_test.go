package ctx

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

type wsPayloadBindDTO struct {
	Name string `bind:"name"`
	Age  int    `bind:"age"`
}

func TestWSPayload_Bind(t *testing.T) {
	p := WSPayload{"name": "joe", "age": float64(30)}
	result, fls := p.Bind(wsPayloadBindDTO{})

	dto, ok := result.(wsPayloadBindDTO)
	if !ok {
		t.Fatal(test.DiffMessage(result, wsPayloadBindDTO{}, "Bind should return a wsPayloadBindDTO"))
	}
	if dto.Name != "joe" || dto.Age != 30 {
		t.Error(test.DiffMessage(dto, wsPayloadBindDTO{Name: "joe", Age: 30}, "Bind should populate fields from the payload map"))
	}
	if len(fls) != 2 {
		t.Error(test.DiffMessage(len(fls), 2, "Bind should report a FieldLevel per bound field"))
	}
}
