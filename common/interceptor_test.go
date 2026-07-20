package common

import (
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

type mockInterceptable struct{}

func (mockInterceptable) Intercept(_ *ctx.HTTPContext, _ *aggregation.Aggregation) any { return nil }

type mockWSInterceptable struct{}

func (mockWSInterceptable) Intercept(_ *ctx.WSContext, _ *aggregation.Aggregation) any { return nil }

type noInterceptMethod struct{}

type wrongParamInterceptable struct{}

func (wrongParamInterceptable) Intercept(_ int, _ *aggregation.Aggregation) any { return nil }

func TestBindInterceptor_Chaining(t *testing.T) {
	i := &Interceptor{}
	ret := i.BindInterceptor(mockInterceptable{})
	if ret != i {
		t.Error(test.DiffMessage(ret, i, "BindInterceptor should return self"))
	}
	if len(i.InterceptorHandlers) != 1 {
		t.Error(test.DiffMessage(len(i.InterceptorHandlers), 1, "one handler after one bind"))
	}
	i.BindInterceptor(mockInterceptable{})
	if len(i.InterceptorHandlers) != 2 {
		t.Error(test.DiffMessage(len(i.InterceptorHandlers), 2, "two handlers after two binds"))
	}
}

func TestInterceptorShapeError_MessageContainsType(t *testing.T) {
	err := InterceptorShapeError(noInterceptMethod{})
	if err == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil error", "InterceptorShapeError must not return nil"))
	}
	if !strings.Contains(err.Error(), "noInterceptMethod") {
		t.Error(test.DiffMessage(err.Error(), "contains noInterceptMethod", "error message should name the offending type"))
	}
}
