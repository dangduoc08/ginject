package ctx

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestContextID_SetID_TruncatesToRequestIDLength(t *testing.T) {
	var c contextID
	c.SetID()
	if len(c.id) != requestIDLength {
		t.Error(test.DiffMessage(len(c.id), requestIDLength, "SetID should truncate the generated id to requestIDLength"))
	}
}

func TestContextID_SetID_StillUnique(t *testing.T) {
	var c1, c2 contextID
	c1.SetID()
	c2.SetID()
	if c1.id == c2.id {
		t.Error(test.DiffMessage(c1.id, "<different id>", "truncated ids should still be unique across separate contexts"))
	}
}
