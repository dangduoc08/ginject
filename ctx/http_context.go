package ctx

import (
	"net/http"
	"time"

	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/internal/str"
)

type HTTPContext struct {
	*http.Request
	http.ResponseWriter

	dataWriter DataWriter

	body        Body
	form        Form
	file        File
	query       Query
	header      Header
	param       Param
	ParamKeys   map[string][]int
	ParamValues []string

	id string

	Next      Next
	Event     *event.Event
	Code      int
	Timestamp time.Time
}

func NewHTTPContext() *HTTPContext {
	return &HTTPContext{
		Code:  http.StatusOK,
		Event: event.NewEvent(),
	}
}

func (c *HTTPContext) Init(w http.ResponseWriter, r *http.Request) {
	c.Timestamp = time.Now()
	c.ResponseWriter = w
	c.Request = r
	c.SetID()
}

func (c *HTTPContext) Reset() {
	c.Code = http.StatusOK
	c.id = ""
	c.body = nil
	c.form = nil
	c.file = nil
	c.query = nil
	c.header = nil
	c.param = nil
	c.ParamKeys = nil
	c.ParamValues = nil
	c.Next = nil
	c.ResponseWriter = nil
	c.Request = nil
	c.Event.Reset()
}

func (c *HTTPContext) SetID() {
	reqID := c.Header().Get(RequestID)
	if reqID == "" {
		reqID, _ = crypto.UUID()
	}

	if c.id == "" {
		c.id = reqID
	}
}

func (c *HTTPContext) GetID() string {
	return c.id
}

func (c *HTTPContext) Status(code int) *HTTPContext {
	c.Code = code
	return c
}

func (c *HTTPContext) Text(data string, args ...any) {
	c.dataWriter = &Text{
		data:           data,
		args:           args,
		responseWriter: c.ResponseWriter,
	}
	c.dataWriter.WriteData(c.Code)
	c.Event.Emit(RequestFinished, c)
}

func (c *HTTPContext) JSON(data ...any) {
	c.dataWriter = &JSON{
		data:           data,
		responseWriter: c.ResponseWriter,
	}
	c.dataWriter.WriteData(c.Code)
	c.Event.Emit(RequestFinished, c)
}

func (c *HTTPContext) JSONP(data ...any) {
	callback := str.RemoveSpace(c.URL.Query().Get("callback"))
	if callback == "" {
		c.JSON(data...)
		return
	}

	c.dataWriter = &JSONP{
		data:           data,
		responseWriter: c.ResponseWriter,
		callback:       callback,
	}
	c.dataWriter.WriteData(c.Code)
	c.Event.Emit(RequestFinished, c)
}

func (c *HTTPContext) Redirect(url string) {
	c.Status(http.StatusMovedPermanently)
	http.Redirect(c.ResponseWriter, c.Request, url, c.Code)
	c.Event.Emit(RequestFinished, c)
}
