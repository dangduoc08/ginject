package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

type mockContextPipeable struct{}

func (mockContextPipeable) Transform(_ *ctx.Context, m ArgumentMetadata) any { return m }

type mockBodyPipeable struct{}

func (mockBodyPipeable) Transform(_ ctx.Body, m ArgumentMetadata) any { return m }

type mockFormPipeable struct{}

func (mockFormPipeable) Transform(_ ctx.Form, m ArgumentMetadata) any { return m }

type mockQueryPipeable struct{}

func (mockQueryPipeable) Transform(_ ctx.Query, m ArgumentMetadata) any { return m }

type mockHeaderPipeable struct{}

func (mockHeaderPipeable) Transform(_ ctx.Header, m ArgumentMetadata) any { return m }

type mockParamPipeable struct{}

func (mockParamPipeable) Transform(_ ctx.Param, m ArgumentMetadata) any { return m }

type mockFilePipeable struct{}

func (mockFilePipeable) Transform(_ ctx.File, m ArgumentMetadata) any { return m }

type mockWSPayloadPipeable struct{}

func (mockWSPayloadPipeable) Transform(_ ctx.WSPayload, m ArgumentMetadata) any { return m }

var (
	_ ContextPipeable   = mockContextPipeable{}
	_ BodyPipeable      = mockBodyPipeable{}
	_ FormPipeable      = mockFormPipeable{}
	_ QueryPipeable     = mockQueryPipeable{}
	_ HeaderPipeable    = mockHeaderPipeable{}
	_ ParamPipeable     = mockParamPipeable{}
	_ FilePipeable      = mockFilePipeable{}
	_ WSPayloadPipeable = mockWSPayloadPipeable{}
)

func TestPipeableConstants(t *testing.T) {
	cases := []struct{ got, want string }{
		{ContextPipeableKey, "context"},
		{BodyPipeableKey, "body"},
		{FormPipeableKey, "form"},
		{QueryPipeableKey, "query"},
		{HeaderPipeableKey, "header"},
		{ParamPipeableKey, "param"},
		{FilePipeableKey, "file"},
		{WSPayloadPipeableKey, "wsPayload"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Error(test.DiffMessage(c.got, c.want, "pipeable constant"))
		}
	}
}

func TestArgumentMetadata_Fields(t *testing.T) {
	m := ArgumentMetadata{
		ContextType: "http",
		ParamType:   BodyPipeableKey,
	}
	if m.ContextType != "http" {
		t.Error(test.DiffMessage(m.ContextType, "http", "ContextType"))
	}
	if m.ParamType != BodyPipeableKey {
		t.Error(test.DiffMessage(m.ParamType, BodyPipeableKey, "ParamType"))
	}
}

func TestArgumentMetadata_Zero(t *testing.T) {
	var m ArgumentMetadata
	if m.ContextType != "" {
		t.Error(test.DiffMessage(m.ContextType, "", "zero ContextType"))
	}
	if m.ParamType != "" {
		t.Error(test.DiffMessage(m.ParamType, "", "zero ParamType"))
	}
}
