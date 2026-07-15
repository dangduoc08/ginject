package ctx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/internal/test"
	"golang.org/x/net/websocket"
)

func newTestContext() *Context {
	c := NewContext()
	c.Broker = broker.New()
	return c
}

func TestNewContext_DefaultCode(t *testing.T) {
	c := NewContext()
	if c.Code != http.StatusOK {
		t.Error(test.DiffMessage(c.Code, http.StatusOK, "NewContext default Code"))
	}
}

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

func TestSetType_ValidTypes(t *testing.T) {
	types := []string{HTTPType, WSType, RPCType, GQLType}
	for _, typ := range types {
		c := newTestContext()
		c.SetType(typ)
		if c.GetType() != typ {
			t.Error(test.DiffMessage(c.GetType(), typ, "SetType "+typ))
		}
	}
}

func TestSetType_InvalidIgnored(t *testing.T) {
	c := newTestContext()
	c.SetType("invalid")
	if c.GetType() != "" {
		t.Error(test.DiffMessage(c.GetType(), "", "SetType invalid stays empty"))
	}
}

func TestSetType_Idempotent(t *testing.T) {
	c := newTestContext()
	c.SetType(HTTPType)
	c.SetType(WSType)
	if c.GetType() != HTTPType {
		t.Error(test.DiffMessage(c.GetType(), HTTPType, "SetType first value wins"))
	}
}

func TestSetType_ReturnsSelf(t *testing.T) {
	c := newTestContext()
	ret := c.SetType(HTTPType)
	if ret != c {
		t.Error(test.DiffMessage(ret, c, "SetType returns self"))
	}
}

func TestSetID_FromHeader(t *testing.T) {
	c := newTestContext()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(RequestID, "test-request-id")
	c.Init(httptest.NewRecorder(), r)
	if c.id != "test-request-id" {
		t.Error(test.DiffMessage(c.id, "test-request-id", "SetID from header"))
	}
}

func TestSetID_GeneratedWhenNoHeader(t *testing.T) {
	c := newTestContext()
	r := httptest.NewRequest("GET", "/", nil)
	c.Init(httptest.NewRecorder(), r)
	if c.id == "" {
		t.Error(test.DiffMessage(c.id, "<non-empty UUID>", "SetID generates UUID"))
	}
}

func TestSetID_Idempotent(t *testing.T) {
	c := newTestContext()
	r := httptest.NewRequest("GET", "/", nil)
	c.Init(httptest.NewRecorder(), r)
	first := c.id
	c.SetID()
	if c.id != first {
		t.Error(test.DiffMessage(c.id, first, "SetID idempotent"))
	}
}

func TestGetID(t *testing.T) {
	c := newTestContext()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(RequestID, "abc-123")
	c.Init(httptest.NewRecorder(), r)
	if c.GetID() != "abc-123" {
		t.Error(test.DiffMessage(c.GetID(), "abc-123", "GetID"))
	}
}

func TestReset_ClearsAllFields(t *testing.T) {
	c := newTestContext()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set(RequestID, "some-id")
	w := httptest.NewRecorder()
	c.Init(w, r)
	c.SetType(HTTPType)
	c.Status(http.StatusNotFound)

	c.Reset()

	if c.Code != http.StatusOK {
		t.Error(test.DiffMessage(c.Code, http.StatusOK, "Reset Code"))
	}
	if c.typ != "" {
		t.Error(test.DiffMessage(c.typ, "", "Reset Type"))
	}
	if c.id != "" {
		t.Error(test.DiffMessage(c.id, "", "Reset ID"))
	}
	if c.Request != nil {
		t.Error(test.DiffMessage(c.Request, nil, "Reset Request"))
	}
	if c.ResponseWriter != nil {
		t.Error(test.DiffMessage(c.ResponseWriter, nil, "Reset ResponseWriter"))
	}
	if c.body != nil {
		t.Error(test.DiffMessage(c.body, nil, "Reset body"))
	}
	if c.ParamKeys != nil {
		t.Error(test.DiffMessage(c.ParamKeys, nil, "Reset ParamKeys"))
	}
	if c.ParamValues != nil {
		t.Error(test.DiffMessage(c.ParamValues, nil, "Reset ParamValues"))
	}
	if c.Next != nil {
		t.Error(test.DiffMessage(c.Next, nil, "Reset Next"))
	}
	if c.GetWSConfig() != nil {
		t.Error(test.DiffMessage(c.GetWSConfig(), nil, "Reset wsCfg"))
	}
	if c.WSConn() != nil {
		t.Error(test.DiffMessage(c.WSConn(), nil, "Reset wsConn"))
	}
	if c.WSPayload() != nil {
		t.Error(test.DiffMessage(c.WSPayload(), nil, "Reset wsPayload"))
	}
}

func TestSetWSConnGetWSConn(t *testing.T) {
	c := newTestContext()
	if c.WSConn() != nil {
		t.Error(test.DiffMessage(c.WSConn(), nil, "WSConn should be nil before SetWSConn"))
	}

	ret := c.SetWSConn(&websocket.Conn{})
	if ret != c {
		t.Error(test.DiffMessage(ret, c, "SetWSConn returns self"))
	}
	if c.WSConn() == nil {
		t.Error(test.DiffMessage(c.WSConn(), "non-nil *websocket.Conn", "WSConn after SetWSConn"))
	}
}

func TestSetWSPayloadGetWSPayload(t *testing.T) {
	c := newTestContext()
	if c.WSPayload() != nil {
		t.Error(test.DiffMessage(c.WSPayload(), nil, "WSPayload should be nil before SetWSPayload"))
	}

	ret := c.SetWSPayload(WSPayload{"foo": "bar"})
	if ret != c {
		t.Error(test.DiffMessage(ret, c, "SetWSPayload returns self"))
	}
	if c.WSPayload()["foo"] != "bar" {
		t.Error(test.DiffMessage(c.WSPayload(), WSPayload{"foo": "bar"}, "WSPayload after SetWSPayload"))
	}
}
