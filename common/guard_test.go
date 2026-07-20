package common

import (
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

type mockGuarder struct{}

func (mockGuarder) CanActivate(_ *ctx.HTTPContext) bool { return true }

type mockWSGuarder struct{}

func (mockWSGuarder) CanActivate(_ *ctx.WSContext) bool { return true }

type denyGuarder struct{}

func (denyGuarder) CanActivate(_ *ctx.HTTPContext) bool { return false }

type denyWSGuarder struct{}

func (denyWSGuarder) CanActivate(_ *ctx.WSContext) bool { return false }

type noCanActivateGuarder struct{}

type wrongReturnGuarder struct{}

func (wrongReturnGuarder) CanActivate(_ *ctx.HTTPContext) string { return "" }

type wrongParamGuarder struct{}

func (wrongParamGuarder) CanActivate(_ int) bool { return true }

func TestBindGuard_Chaining(t *testing.T) {
	g := &Guard{}
	ret := g.BindGuard(mockGuarder{})
	if ret != g {
		t.Error(test.DiffMessage(ret, g, "BindGuard should return self"))
	}
	if len(g.GuardHandlers) != 1 {
		t.Error(test.DiffMessage(len(g.GuardHandlers), 1, "one handler after one bind"))
	}
	g.BindGuard(mockGuarder{})
	if len(g.GuardHandlers) != 2 {
		t.Error(test.DiffMessage(len(g.GuardHandlers), 2, "two handlers after two binds"))
	}
}

func TestGuardShapeError_MessageContainsType(t *testing.T) {
	err := GuardShapeError(noCanActivateGuarder{})
	if err == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil error", "GuardShapeError must not return nil"))
	}
	if !strings.Contains(err.Error(), "noCanActivateGuarder") {
		t.Error(test.DiffMessage(err.Error(), "contains noCanActivateGuarder", "error message should name the offending type"))
	}
}
