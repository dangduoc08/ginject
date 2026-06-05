package runtime

import (
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestNodeID_NonEmpty(t *testing.T) {
	id := NodeID()
	if id == "" {
		t.Error(test.DiffMessage(id, "non-empty", "NodeID must be non-empty"))
	}
}

func TestNodeID_ContainsTwoColons(t *testing.T) {
	id := NodeID()
	if strings.Count(id, ":") < 2 {
		t.Error(test.DiffMessage(strings.Count(id, ":"), ">=2", "NodeID must have at least 2 colon separators"))
	}
}

func TestNodeID_Idempotent(t *testing.T) {
	a := NodeID()
	b := NodeID()
	if a != b {
		t.Error(test.DiffMessage(b, a, "NodeID must return the same value on every call"))
	}
}

func TestNodeID_ContainsPID(t *testing.T) {
	id := NodeID()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) < 3 {
		t.Error(test.DiffMessage(len(parts), 3, "NodeID must have 3 colon-separated segments"))
		return
	}
	if parts[1] == "" {
		t.Error(test.DiffMessage(parts[1], "non-empty PID", "NodeID PID segment must not be empty"))
	}
}

func TestNodeID_ContainsTimestamp(t *testing.T) {
	id := NodeID()
	parts := strings.SplitN(id, ":", 3)
	if len(parts) < 3 {
		t.Error(test.DiffMessage(len(parts), 3, "NodeID must have 3 colon-separated segments"))
		return
	}
	if parts[2] == "" {
		t.Error(test.DiffMessage(parts[2], "non-empty timestamp", "NodeID timestamp segment must not be empty"))
	}
}
