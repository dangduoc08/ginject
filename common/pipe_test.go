package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
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
		{CONTEXT_PIPEABLE, "context"},
		{BODY_PIPEABLE, "body"},
		{FORM_PIPEABLE, "form"},
		{QUERY_PIPEABLE, "query"},
		{HEADER_PIPEABLE, "header"},
		{PARAM_PIPEABLE, "param"},
		{FILE_PIPEABLE, "file"},
		{WS_PAYLOAD_PIPEABLE, "wsPayload"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Error(testutils.DiffMessage(c.got, c.want, "pipeable constant"))
		}
	}
}

func TestArgumentMetadata_Fields(t *testing.T) {
	m := ArgumentMetadata{
		ContextType: "http",
		ParamType:   BODY_PIPEABLE,
	}
	if m.ContextType != "http" {
		t.Error(testutils.DiffMessage(m.ContextType, "http", "ContextType"))
	}
	if m.ParamType != BODY_PIPEABLE {
		t.Error(testutils.DiffMessage(m.ParamType, BODY_PIPEABLE, "ParamType"))
	}
}

func TestArgumentMetadata_Zero(t *testing.T) {
	var m ArgumentMetadata
	if m.ContextType != "" {
		t.Error(testutils.DiffMessage(m.ContextType, "", "zero ContextType"))
	}
	if m.ParamType != "" {
		t.Error(testutils.DiffMessage(m.ParamType, "", "zero ParamType"))
	}
}
