package ctx

import (
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestStatus_SetsCodeAndReturnsSelf(t *testing.T) {
	c := newTestContext()
	ret := c.Status(http.StatusCreated)
	if c.Code != http.StatusCreated {
		t.Error(test.DiffMessage(c.Code, http.StatusCreated, "Status code"))
	}
	if ret != c {
		t.Error(test.DiffMessage(ret, c, "Status returns self"))
	}
}
