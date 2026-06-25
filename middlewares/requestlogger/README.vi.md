# Middleware RequestLogger

*`requestlogger` ghi log một bản tóm tắt có cấu trúc cho mỗi request HTTP hoặc WebSocket đã hoàn tất, dưới dạng một middleware của Ginject.*

- [Middleware RequestLogger](#middleware-requestlogger)
  - [Tính Năng Chính](#tính-năng-chính)
  - [Cách Dùng](#cách-dùng)
  - [Cách Hoạt Động](#cách-hoạt-động)
  - [Struct `RequestLogger`](#struct-requestlogger)
    - [Logger](#logger)
  - [Phương Thức Của `RequestLogger`](#phương-thức-của-requestlogger)
    - [Use](#use)
  - [Benchmark](#benchmark)

## Tính Năng Chính
- Chỉ ghi log một lần cho mỗi request, sau khi response đã thực sự hoàn tất, bằng cách subscribe vào broker event `ctx.RequestFinished` thay vì bọc trực tiếp `next`
- Một dòng log cho mỗi request HTTP, gồm URL, method, status, response time, protocol, user agent, và request ID
- Một dòng log cho mỗi event WebSocket, gồm tên event, response time, subprotocol, và user agent
- Giao việc ghi log thực tế cho bất kỳ hiện thực `common.Logger` nào được cấu hình — mặc định là logger có cấu trúc riêng của Ginject

## Cách Dùng

```go
package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/middlewares/requestlogger"
)

func main() {
	app := core.New()

	app.BindGlobalMiddlewares(requestlogger.RequestLogger{})

	app.Create(
		core.ModuleBuilder().Build(),
	)
}
```

`RequestLogger` cũng hoạt động khi gắn theo từng controller riêng, vì nó thỏa mãn trực tiếp interface `common.MiddlewareFn`:

```go
func (c APIController) NewController() core.Controller {
	c.BindMiddleware(requestlogger.RequestLogger{})
	return c
}
```

## Cách Hoạt Động

`RequestLogger` embed `common.Logger` thay vì bọc nó trong một field riêng, nên struct này tự expose `Debug`/`Info`/`Warn`/`Error`/`Fatal`. Khi được gắn (bind) qua `BindGlobalMiddlewares` hoặc `BindMiddleware`, DI container của Ginject sẽ tự động inject `common.Logger` đã cấu hình của app (logger của package `log` theo mặc định, hoặc bất kỳ logger nào đã truyền vào `app.UseLogger`) vào field embedded đó — bạn không cần tự đặt nó trong code ứng dụng. Khi khởi tạo một `RequestLogger` trực tiếp, như các test của package này làm, bạn phải cung cấp `Logger` một cách rõ ràng, vì nếu không, interface embedded đó sẽ là `nil`.

`Use` không ghi log một cách đồng bộ: nó subscribe một handler vào broker event `ctx.RequestFinished`, rồi gọi `next` ngay lập tức. Handler đã subscribe sẽ chạy khi broker publish `ctx.RequestFinished` cho context đó (sau khi response đã thực sự được gửi), và ghi log một dòng HTTP hoặc WebSocket tùy theo `ctx.Context.GetType()`.

## Struct `RequestLogger`

### Logger
Type: `common.Logger` (embedded)

Default: `nil`

Required: `false` khi được gắn qua `BindGlobalMiddlewares`/`BindMiddleware` (được DI container của Ginject tự động inject); ngược lại phải được cung cấp rõ ràng để tránh panic do interface nil khi một request hoàn tất.

```go
requestlogger.RequestLogger{Logger: myLogger}
```

## Phương Thức Của `RequestLogger`

### Use

Subscribe một handler ghi log một lần (one-shot) vào event `ctx.RequestFinished` cho context hiện tại, sau đó gọi `next`.

#### Rules
- `next` luôn được gọi (`TestRequestLogger_Use_CallsNext`).
- Không có gì được ghi log cho tới khi `ctx.RequestFinished` được publish cho context đó — chỉ gọi `Use` một mình thì không ghi log (`TestRequestLogger_Use_NoLogWithoutEventEmit`).
- Với một context HTTP, khi `ctx.RequestFinished` được publish, đúng một lệnh gọi `Info` được thực hiện với URL path của request làm message, và các cặp key/value `Method`, `Status`, `Time`, `Protocol`, `User-Agent`, và `ctx.RequestID` làm args (`TestRequestLogger_Use_HTTPLogsURL`, `TestRequestLogger_Use_HTTPLogsMethod`, `TestRequestLogger_Use_HTTPLogsStatus`, `TestRequestLogger_Use_HTTPLogsProtocol`, `TestRequestLogger_Use_HTTPLogsRequestID`).
- Arg `Time` là response time được format dưới dạng chuỗi `time.Duration` của Go, kết thúc bằng `"ms"` đối với độ trễ dưới 1 giây (`TestRequestLogger_Use_HTTPLogsTime`).
- Một context có type không phải HTTP cũng không phải WebSocket (ví dụ `Type` trống) sẽ không ghi log gì khi `ctx.RequestFinished` được publish (`TestRequestLogger_Use_UnknownTypeNoLog`).

#### Parameters
- Tham số thứ 1: `*ctx.Context` (`c`)

- Mô tả: Context của request hiện tại; `Broker` của nó được subscribe để nhận event request đã hoàn tất.

- Tham số thứ 2: `ctx.Next` (`next`)

- Mô tả: Được gọi để chuyển quyền xử lý cho handler kế tiếp trong chuỗi.

#### Returns
Không có.

#### Cách Dùng

```go
requestlogger.RequestLogger{Logger: myLogger}.Use(c, next)
```

## Benchmark

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./middlewares/requestlogger/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/middlewares/requestlogger
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkRequestLogger_Use_HTTP-12            	  214678	      5300 ns/op	    7147 B/op	      36 allocs/op
BenchmarkRequestLogger_Use_RegisterOnly-12    	 1000000	      1378 ns/op	     343 B/op	       4 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/middlewares/requestlogger	4.692s
```
