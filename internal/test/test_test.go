package test

import (
	"strings"
	"testing"
)

func TestDiffMessage_WithDesc(t *testing.T) {
	msg := DiffMessage("got", "want", "field name")
	if !strings.Contains(msg, "field name") {
		t.Errorf("DiffMessage must contain desc, got: %q", msg)
	}
	if !strings.Contains(msg, "want") {
		t.Errorf("DiffMessage must contain expected value, got: %q", msg)
	}
	if !strings.Contains(msg, "got") {
		t.Errorf("DiffMessage must contain actual value, got: %q", msg)
	}
}

func TestDiffMessage_NoDesc(t *testing.T) {
	msg := DiffMessage("actual", "expected", "")
	if !strings.Contains(msg, "actual") {
		t.Errorf("DiffMessage must contain actual value, got: %q", msg)
	}
	if !strings.Contains(msg, "expected") {
		t.Errorf("DiffMessage must contain expected value, got: %q", msg)
	}
}

func TestDiffMessage_ContainsLabels(t *testing.T) {
	msg := DiffMessage(42, 99, "count")
	if !strings.Contains(msg, "Expected") {
		t.Errorf("DiffMessage must contain 'Expected' label, got: %q", msg)
	}
	if !strings.Contains(msg, "Actual") {
		t.Errorf("DiffMessage must contain 'Actual' label, got: %q", msg)
	}
}

func TestDiffMessage_NilValues(t *testing.T) {
	msg := DiffMessage(nil, nil, "")
	if !strings.Contains(msg, "Expected") || !strings.Contains(msg, "Actual") {
		t.Errorf("DiffMessage must work with nil values, got: %q", msg)
	}
}
