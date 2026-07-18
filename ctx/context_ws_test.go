package ctx

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
	"golang.org/x/net/websocket"
)

func TestSetWSConnGetWSConn(t *testing.T) {
	c := newTestHTTPContext()
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
	c := newTestHTTPContext()
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
