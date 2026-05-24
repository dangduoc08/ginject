package benchmarks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	ctxpkg "github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/modules/httpclient"
)

type Controller struct {
	common.REST
	httpclient.ClientService
}

func (instance Controller) NewController() core.Controller {

	return instance
}

func (instance Controller) READ_ping() ginject.Map {
	return ginject.Map{
		"message": "Hello, World!",
	}
}

// POST /notify — demo GoSafe: trả 202 ngay, background job chạy async.
// Thử: curl -X POST http://localhost:3000/notify
// Thử cancel giữa chừng: curl -X POST http://localhost:3000/notify & sleep 1 && kill %1
func (instance Controller) CREATE_notify(c ginject.Context) {
	reqID := c.GetID()
	exec := c.GetExec()

	ctxpkg.GoSafe(exec, func(bgCtx context.Context) {
		fmt.Printf("[GoSafe][%s] background job started\n", reqID)

		select {
		case <-bgCtx.Done():
			// client ngắt kết nối hoặc request timeout — không cần làm gì
			fmt.Printf("[GoSafe][%s] cancelled: %v\n", reqID, bgCtx.Err())
			return
		case <-time.After(3 * time.Second):
		}

		// giả lập gửi notification thất bại → panic
		panic("simulated notification send failure")

	}, func(err error, stack []byte) {
		// panic được recover, không crash server
		fmt.Printf("[GoSafe][%s] recovered panic: %v\nStack:\n%s\n", reqID, err, stack)
	})

	// trả response ngay, không đợi background job
	c.Status(http.StatusAccepted).JSON(ginject.Map{
		"status":    "accepted",
		"requestID": reqID,
		"note":      "notification is being sent in the background",
	})
}

// DELETE /crash — demo crash server: spawn goroutine thuần (không GoSafe), panic sau 1s.
// handleRESTRequest đã return trước khi panic xảy ra → defer recover() không còn tác dụng.
// Thử: curl -X DELETE http://localhost:4000/crash
func (instance Controller) DELETE_crash(c ginject.Context) {
	reqID := c.GetID()

	go func() {
		fmt.Printf("[UNSAFE][%s] background goroutine started — will panic in 1s\n", reqID)
		time.Sleep(time.Second)
		// Không có recover() → crash toàn bộ server
		panic("unrecovered panic: server will crash")
	}()

	c.Status(http.StatusAccepted).JSON(ginject.Map{
		"status":    "accepted",
		"requestID": reqID,
		"note":      "server will crash in ~1s",
	})
}

// DELETE /safe_crash — tương đương DELETE /crash nhưng dùng GoSafe.
// Cùng panic sau 1s, nhưng được recover → server sống, log lỗi ra console.
// Thử: curl -X DELETE http://localhost:4000/safe_crash
func (instance Controller) DELETE_safe_crash(c ginject.Context) {
	reqID := c.GetID()

	ctxpkg.GoSafe(c.GetExec(), func(_ context.Context) {
		fmt.Printf("[GoSafe][%s] background goroutine started — will panic in 1s\n", reqID)
		time.Sleep(time.Second)
		panic("panic inside GoSafe: server stays alive")
	}, func(err error, stack []byte) {
		fmt.Printf("[GoSafe][%s] recovered panic: %v\nStack:\n%s\n", reqID, err, stack)
	})

	c.Status(http.StatusAccepted).JSON(ginject.Map{
		"status":    "accepted",
		"requestID": reqID,
		"note":      "panic will be recovered, server stays alive",
	})
}
