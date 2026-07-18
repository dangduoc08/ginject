# Package Routing

*`routing` xây dựng router theo method/version/path mà module core của Ginject dùng để đăng ký, nhóm (group), và phân giải (resolve) các route HTTP, dựa trên cấu trúc trie theo segment từ `internal/ds`.*

- [Package Routing](#package-routing)
  - [Tính Năng Chính](#tính-năng-chính)
  - [Cách Dùng](#cách-dùng)
  - [Hằng Số](#hằng-số)
    - [SERVE](#serve)
    - [ADD, USE, FOR, GROUP](#add-use-for-group)
  - [Biến Của Package](#biến-của-package)
    - [OperationsMapHTTPMethods](#operationsmaphttpmethods)
    - [HTTPMethods](#httpmethods)
  - [Struct `RouterItem`](#struct-routeritem)
    - [Method](#method)
    - [Version](#version)
    - [Pattern](#pattern)
    - [Index](#index)
    - [HandlerIndex](#handlerindex)
    - [Handlers](#handlers)
    - [ParamKeys](#paramkeys)
  - [Struct `Router`](#struct-router)
    - [Trie](#trie)
    - [Hash](#hash)
    - [List](#list)
    - [GlobalMiddlewares](#globalmiddlewares)
    - [InjectableHandlers](#injectablehandlers)
  - [Hàm](#hàm)
    - [NewRouter](#newrouter)
    - [PatternToMethodRouteVersion](#patterntomethodrouteversion)
    - [ToEndpoint](#toendpoint)
    - [MethodRouteVersionToPattern](#methodrouteversiontopattern)
    - [ParseToParamKey](#parsetoparamkey)
  - [Phương Thức Của `*Router`](#phương-thức-của-router)
    - [Match](#match)
    - [Group](#group)
    - [Use](#use)
    - [For](#for)
    - [Add](#add)
    - [AddInjectableHandler](#addinjectablehandler)
  - [Benchmark](#benchmark)

## Tính Năng Chính
- Route được khớp (match) qua trie của `internal/ds`, nên các segment literal, `{param}`, và wildcard `*` đều được phân giải qua `Match` trong một lần duyệt
- Mỗi HTTP method và mỗi version tag trên cùng một path có chuỗi handler độc lập
- Ba cách gắn handler — `Use` (global), `For` (theo route, áp dụng cho nhiều method), `Add` (handler chính của route) — được kết hợp theo đúng thứ tự gọi
- `Group` cho phép gắn toàn bộ một sub-router đã xây dựng sẵn (route, handler, và injectable handler) vào dưới một path prefix
- `AddInjectableHandler` đăng ký một handler được resolve bằng reflection thay vì theo signature cố định `ctx.Handler`, dùng cho DI container

## Cách Dùng

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/routing"
)

func main() {
	r := routing.NewRouter()

	getHandler := func(c *ctx.HTTPContext) {}
	r.Add(http.MethodGet, "/users/{id}", "", getHandler)

	isMatched, pattern, paramKeys, paramVals, handlers := r.Match(http.MethodGet, "/users/123/", "")

	fmt.Println("matched:", isMatched)
	fmt.Println("pattern:", pattern)
	fmt.Println("paramKeys:", paramKeys)
	fmt.Println("paramVals:", paramVals)
	fmt.Println("handlers:", len(handlers))
}
```

Console:
```console
matched: true
pattern: /users/{id}/||/[GET]/
paramKeys: map[id:[0]]
paramVals: [123]
handlers: 1
```

## Hằng Số

### SERVE
Type: `string`

Value: `"SERVE"`

Một pseudo HTTP method dùng để đăng ký route phục vụ file tĩnh (static file). `OperationsMapHTTPMethods[SERVE]` quy về `http.MethodGet`, vì việc phục vụ một file được xử lý như một `GET`.

### ADD, USE, FOR, GROUP
Type: `int`

Values: `ADD = 1`, `USE = 2`, `FOR = 3`, `GROUP = 4` (khai báo bằng `iota + 1`)

Các marker đánh dấu nơi gọi (call-site), được truyền vào logic đăng ký route nội bộ (unexported) của package để xác định public method nào — `Add`, `Use`, `For`, hay `Group` — đã kích hoạt một lần đăng ký, để chuỗi handler được hợp nhất khác nhau tùy theo nơi gọi. Không có hàm public nào nhận một trong các giá trị này làm tham số; chúng được ghi lại ở đây chỉ vì đây là các identifier được export.

## Biến Của Package

### OperationsMapHTTPMethods
Type: `map[string]string`

Ánh xạ mỗi method mà package này nhận diện sang method `net/http` tương ứng cần được coi là: mỗi method chuẩn (`GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `CONNECT`, `OPTIONS`, `TRACE`) ánh xạ về chính nó, và pseudo-method `SERVE` ánh xạ về `http.MethodGet`.

### HTTPMethods
Type: `[]string`

Cùng tập method như các key của `OperationsMapHTTPMethods` (9 method chuẩn của `net/http` cộng thêm `SERVE`), dưới dạng slice có thứ tự — dùng để đăng ký một handler cho tất cả method cùng lúc, ví dụ `for _, m := range routing.HTTPMethods { r.Add(m, route, version, handler) }`.

## Struct `RouterItem`

Lưu mọi thứ `Match` cần sau khi trie đã phân giải một path: method/version nào item này được đăng ký cho, và chuỗi handler của nó. Nhiều `RouterItem` có thể dùng chung một `Index` (và cùng một leaf trong trie) khi cùng một path được đăng ký dưới nhiều method hoặc version khác nhau.

### Method
Type: `string`

Default: tham số `method` được truyền vào `Add` (hoặc vào `For`/`Use` cho một route đã đăng ký sẵn)

Required: `true`

HTTP method (hoặc `SERVE`) mà item này được đăng ký cho.

### Version
Type: `string`

Default: tham số `version` được truyền vào; `""` nếu không có

Required: `false`

Version tag mà item này được đăng ký cho. `Match` chỉ trả về một item có `Version` khớp chính xác với version được gọi.

### Pattern
Type: `string`

Default: `MethodRouteVersionToPattern(method, route, version)`

Required: `true`

Chuỗi pattern chuẩn xác định chính xác tổ hợp method+route+version này; đây là giá trị `Match` trả về làm giá trị thứ 2 khi khớp thành công.

### Index
Type: `int`

Default: vị trí của route của item này trong `List` của router

Required: `true`

ID mà route của item này phân giải tới trong trie nhúng (embedded); được nhiều `RouterItem` dùng chung nếu chúng đăng ký cho cùng một route path, bất kể method/version.

### HandlerIndex
Type: `int`

Default: `-1`

Required: `false`

Chỉ số (index), trong `Handlers`, của slot được `Add` ghi lần gần nhất. `-1` nghĩa là `Add` chưa đăng ký handler chính nào cho method/route/version này.

### Handlers
Type: `[]ctx.Handler`

Default: `nil` cho tới lần gọi `Use`, `For`, hoặc `Add` đầu tiên chạm tới method/route/version này

Required: `false`

Toàn bộ chuỗi handler — global middleware, middleware theo route, và handler chính, theo đúng thứ tự được gắn vào — mà `Match` trả về làm giá trị thứ 5.

### ParamKeys
Type: `map[string][]int`

Default: `nil` trừ khi route có chứa placeholder `{name}`

Required: `false`

Với mỗi tham số có tên trong route, vị trí (theo thứ tự duyệt segment `$`) mà nó xuất hiện; được xây dựng bởi `ParseToParamKey` khi item được đăng ký.

## Struct `Router`

Router chính nó: một trie được embed cùng với phần lưu trữ (bookkeeping) mà `Add`, `Use`, `For`, `Group`, và `Match` cần.

### Trie
Type: `*ds.Trie`

Default: `ds.NewTrie()`

Required: `false`

Được embed ẩn danh, nên các method được export của `ds.Trie` (`Len`, `Insert`, `Find`, `ToJSON`) được promote (đưa lên) trực tiếp vào `Router` (ví dụ `router.Len()`). `Add` chèn mỗi route đã đăng ký vào trie này, và `Match` phân giải các path đến qua nó; xem README của chính `internal/ds` để biết hành vi đã được ghi lại của các method đó.

### Hash
Type: `map[string][]RouterItem`

Default: `make(map[string][]RouterItem)`

Required: `false`

Mọi `RouterItem` đã đăng ký cho tới hiện tại, được đánh khóa (key) bằng endpoint path đã chuẩn hóa của route (`ToEndpoint(route)`) — một key có thể chứa nhiều item, mỗi item cho một tổ hợp method/version được đăng ký cho path đó.

### List
Type: `[]string`

Default: `nil`

Required: `false`

Mọi route path đã chuẩn hóa, khác biệt, đã đăng ký cho tới hiện tại, theo thứ tự đăng ký; một path được append vào lần đầu tiên nó xuất hiện, và vị trí (index) của nó trong slice này trở thành `Index`/id trong trie của route đó.

### GlobalMiddlewares
Type: `[]ctx.Handler`

Default: `[]ctx.Handler{}`

Required: `false`

Các handler được tích lũy qua `Use`. Chúng được thêm vào phía trước handler riêng của một route khi route đó được đăng ký mà chưa có handler nào, và được append một cách hồi tố (retroactively) vào chuỗi `Handlers` của mọi route đã đăng ký từ trước.

### InjectableHandlers
Type: `map[string]any`

Default: `make(map[string]any)`

Required: `false`

Các handler được đăng ký qua `AddInjectableHandler`, đánh khóa bằng `MethodRouteVersionToPattern(method, route, version)`. `Group` cũng sao chép các entry của một sub-router vào receiver, đánh khóa lại dưới group prefix.

## Hàm

### NewRouter

Tạo một `*Router` trống, sẵn sàng cho `Add`, `Use`, `For`, `Group`, `Match`, và `AddInjectableHandler`.

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `*Router`

- Mô tả: Một router với trie trống, `Hash` trống, `List` là `nil`, slice `GlobalMiddlewares` trống, và map `InjectableHandlers` trống.

#### Cách Dùng

```go
r := routing.NewRouter()
```

### PatternToMethodRouteVersion

Phân tích một chuỗi pattern do `MethodRouteVersionToPattern` tạo ra, trả về lại các thành phần method, route, và version — đây là hàm nghịch đảo của `MethodRouteVersionToPattern`.

#### Rules
- Khôi phục đúng bộ ba `(method, route, version)` đã tạo ra pattern đó, ví dụ `"/users/$/|v2|/[POST]/"` → `("POST", "/users/$", "v2")` (`TestPatternToMethodRouteVersion`).
- Một đoạn version trống (`||`) được phân tích lại thành chuỗi `version` trống, ví dụ `"/users/$/||/[GET]/"` → version `""` (`TestPatternToMethodRouteVersion`).

#### Parameters
- Tham số thứ 1: `string` (`pattern`)

- Mô tả: Một chuỗi pattern có dạng `<route>/<|version|>/<[METHOD]>/` do `MethodRouteVersionToPattern` tạo ra.

#### Returns
- Giá trị thứ 1: `string`

- Mô tả: HTTP method được trích từ đoạn `[METHOD]` của pattern.

- Giá trị thứ 2: `string`

- Mô tả: Route path được trích từ pattern, sau khi loại bỏ đoạn version/method ở cuối.

- Giá trị thứ 3: `string`

- Mô tả: Version tag được trích từ đoạn `|version|` của pattern, hoặc `""` nếu đoạn đó trống.

#### Cách Dùng

```go
method, route, version := routing.PatternToMethodRouteVersion("/users/$/|v2|/[POST]/")
fmt.Println(method, route, version)
```

Console:
```console
POST /users/$ v2
```

### ToEndpoint

Chuẩn hóa một path thô thành endpoint sẵn sàng cho router: luôn được bao bằng `/` ở đầu/cuối, đã loại bỏ khoảng trắng và gộp các `/`/`*` liên tiếp.

#### Rules
- Luôn bao kết quả bằng `/` ở đầu và cuối (`"users"` → `"/users/"`) (`TestToEndpoint`).
- Gộp các ký tự `/` liên tiếp và các ký tự `*` liên tiếp thành một ký tự duy nhất cho mỗi loại (`"//users//"` → `"/users/"`; `"/a/**/b/"` → `"/a/*/b/"`) (`TestToEndpoint`).
- Loại bỏ khoảng trắng ASCII ở bất kỳ đâu trong input, kể cả ở đầu/cuối (`" /users/ "` → `"/users/"`) (`TestToEndpoint`).

#### Parameters
- Tham số thứ 1: `string` (`str`)

- Mô tả: Path thô cần chuẩn hóa.

#### Returns
- Giá trị thứ 1: `string`

- Mô tả: Endpoint path đã được chuẩn hóa.

#### Cách Dùng

```go
fmt.Println(routing.ToEndpoint("//users//"))
fmt.Println(routing.ToEndpoint("/a/**/b/"))
```

Console:
```console
/users/
/a/*/b/
```

### MethodRouteVersionToPattern

Xây dựng chuỗi pattern chuẩn mà `Router` lưu trên mỗi `RouterItem` và dùng để tra cứu route đã đăng ký theo method/route/version.

#### Rules
- Tạo ra một chuỗi có dạng `"<endpoint>/<|version|>/<[METHOD]>/"`, ví dụ `(GET, "/users/{userId}", "")` → `"/users/{userId}/||/[GET]/"`; với version: `(POST, "/users/{userId}", "v2")` → `"/users/{userId}/|v2|/[POST]/"` (`TestMethodRouteVersionToPattern`).
- `method` rỗng vẫn tạo ra pattern với cặp ngoặc rỗng `[]` thay vì bỏ hẳn đoạn method, ví dụ `("", "/feeds/all", "")` → `"/feeds/all/||/[]/"` (`TestMethodRouteVersionToPattern`).

#### Parameters
- Tham số thứ 1: `string` (`method`)

- Mô tả: HTTP method cần nhúng vào pattern.

- Tham số thứ 2: `string` (`route`)

- Mô tả: Route path cần chuẩn hóa (qua `ToEndpoint`) và nhúng vào.

- Tham số thứ 3: `string` (`version`)

- Mô tả: Version tag cần nhúng vào; truyền `""` nếu không có version.

#### Returns
- Giá trị thứ 1: `string`

- Mô tả: Chuỗi pattern kết hợp.

#### Cách Dùng

```go
fmt.Println(routing.MethodRouteVersionToPattern(http.MethodPost, "/users/{userId}", "v2"))
```

Console:
```console
/users/{userId}/|v2|/[POST]/
```

### ParseToParamKey

Thay mỗi placeholder `{name}` trong một route bằng một segment literal `$` (dạng mà trie dùng để khớp) và ghi lại vị trí xuất hiện của mỗi tên tham số.

#### Rules
- Thay mỗi placeholder `{paramName}` bằng `$`, và trả về một `map[string][]int` ghi lại, với mỗi tên, chỉ số bắt đầu từ 0 (theo thứ tự xuất hiện) của mỗi placeholder dùng tên đó, ví dụ `"/users/{userId}/friends/{friendId}/"` → `"/users/$/friends/$/"` với `keys["userId"] == [0]` và `keys["friendId"] == [1]` (`TestParseToParamKey`).
- Một chuỗi không có placeholder `{...}` nào được trả về không thay đổi, kèm một param-key map trống (`"/plain/route/"` → `("/plain/route/", map[string][]int{})`) (`TestParseToParamKey`).

#### Parameters
- Tham số thứ 1: `string` (`str`)

- Mô tả: Route path cần quét tìm placeholder `{name}`.

#### Returns
- Giá trị thứ 1: `string`

- Mô tả: Route sau khi mỗi placeholder `{name}` đã được thay bằng `$`.

- Giá trị thứ 2: `map[string][]int`

- Mô tả: Với mỗi tên tham số, danh sách các vị trí segment `$` mà nó chiếm.

#### Cách Dùng

```go
str, keys := routing.ParseToParamKey("/users/{userId}/friends/{friendId}/")
fmt.Println(str, keys)
```

Console:
```console
/users/$/friends/$/ map[friendId:[1] userId:[0]]
```

## Phương Thức Của `*Router`

### Match

Phân giải một request đến `(method, route, version)` với mọi route đã đăng ký trên router, và trả về chuỗi handler của nó.

#### Rules
- Trả về `isMatched = false` khi không có `RouterItem` nào cho route đã phân giải dưới đúng tổ hợp `(method, version)` — kể cả khi cùng path đó được đăng ký dưới một method hoặc version khác (`TestRouterMatchSamePathDifferentMethodAndVersion`: một `DELETE` chưa từng được đăng ký, và một tổ hợp `POST` + `"v2"` chưa từng được đăng ký, cả hai đều không khớp).
- Giá trị trả về thứ 4 (`paramVals`) chứa các giá trị literal được capture cho mỗi segment `{name}`, theo đúng thứ tự các segment xuất hiện trong route (`TestRouterMatchSamePathDifferentMethodAndVersion`: khớp `/users/123/` với `/users/{id}` cho ra `paramVals[0] == "123"`).
- Giá trị trả về thứ 2 bằng `MethodRouteVersionToPattern(method, <route đã đăng ký>, version)` cho bất kỳ route đã đăng ký nào — literal, `{param}`, hay wildcard `*` — mà path được phân giải tới (`TestRouterMatch`, bao gồm route static, route có một hoặc nhiều param, và route wildcard/pattern literal như `in*.html`).

#### Parameters
- Tham số thứ 1: `string` (`method`)

- Mô tả: HTTP method của request đến.

- Tham số thứ 2: `string` (`route`)

- Mô tả: Path của request đến cần phân giải.

- Tham số thứ 3: `string` (`version`)

- Mô tả: Version tag cần khớp với; `""` nếu không có version.

#### Returns
- Giá trị thứ 1: `bool`

- Mô tả: Có tìm thấy handler đã đăng ký cho đúng method/version này hay không.

- Giá trị thứ 2: `string`

- Mô tả: `Pattern` chuẩn của route đã khớp, hoặc `""` nếu không khớp.

- Giá trị thứ 3: `map[string][]int`

- Mô tả: `ParamKeys` của route đã khớp, hoặc `nil` nếu không khớp.

- Giá trị thứ 4: `[]string`

- Mô tả: Các giá trị tham số đã capture, theo thứ tự trong route.

- Giá trị thứ 5: `[]ctx.Handler`

- Mô tả: Chuỗi handler của route đã khớp, hoặc `nil` nếu không khớp.

#### Cách Dùng

```go
isMatched, pattern, paramKeys, paramVals, handlers := r.Match(http.MethodGet, "/users/123/", "")
fmt.Println(isMatched, pattern, paramKeys, paramVals, len(handlers))
```

Console:
```console
true /users/{id}/||/[GET]/ map[id:[0]] [123] 1
```

### Group

Gắn mọi route (và injectable handler) từ một hoặc nhiều sub-router đã xây dựng sẵn vào receiver, dưới một path prefix.

#### Rules
- Với mỗi route đã đăng ký trên mỗi `subRouter`, đăng ký lại nó trên receiver dưới `prefix + route`, cùng method và version, giữ nguyên chuỗi handler của route đó (`TestRouterGroup`: nhóm một sub-router có `/users/update/{userId}` dưới prefix `/v1` khiến `PATCH /v1/users/update/123/` phân giải tới pattern của `/v1/users/update/{userId}`).
- Nhiều sub-router có thể được nhóm dưới cùng một prefix trong một lần gọi; route từ mỗi sub-router được hợp nhất vào receiver (`TestRouterGroup` nhóm hai sub-router cùng dưới `/v1` trong một lần gọi `Group`).

#### Parameters
- Tham số thứ 1: `string` (`prefix`)

- Mô tả: Path prefix để gắn route của mọi sub-router vào dưới.

- Tham số thứ 2: `...*Router` (`subRouters`)

- Mô tả: Một hoặc nhiều router đã xây dựng sẵn, có route (và injectable handler) cần được hợp nhất vào receiver.

#### Returns
- Giá trị thứ 1: `*Router`

- Mô tả: Receiver, để có thể gọi nối tiếp (chain).

#### Cách Dùng

```go
v1 := routing.NewRouter()
v1.Add(http.MethodPatch, "/users/update/{userId}", "", func(c *ctx.HTTPContext) {})

gr := routing.NewRouter()
gr.Group("/v1", v1)

_, pattern, _, _, _ := gr.Match(http.MethodPatch, "/v1/users/update/123/", "")
fmt.Println(pattern)
```

Console:
```console
/v1/users/update/{userId}/||/[PATCH]/
```

### Use

Đăng ký các handler làm global middleware cho mọi route trên router này.

#### Rules
- Append các handler được truyền vào, vào `GlobalMiddlewares`, và append một cách hồi tố (retroactively) vào chuỗi `Handlers` của mọi route đã đăng ký trên router (`TestRouterMiddleware`: một lệnh gọi `Use(handler1)` lần thứ hai, sau khi một route đã có 4 handler, làm chuỗi đó tăng lên 5).
- Các route được thêm *sau khi* `Use` đã tích lũy global middleware sẽ có các middleware đó được thêm vào phía trước handler riêng của chúng, miễn là chưa có handler nào được đăng ký cho đúng method/route/version đó (`TestRouterMiddleware`, router 0).
- Trả về receiver, nên có thể gọi nối tiếp (chain), ví dụ `gr.Use(handler4).Use(handler2).Use(handler1)` (`TestRouterMiddleware`).

#### Parameters
- Tham số thứ 1: `...ctx.Handler` (`handlers`)

- Mô tả: Các handler cần đăng ký làm global middleware.

#### Returns
- Giá trị thứ 1: `*Router`

- Mô tả: Receiver, để có thể gọi nối tiếp (chain).

#### Cách Dùng

```go
r.Use(func(c *ctx.HTTPContext) {
	c.Next()
})
```

### For

Trả về một hàm để đăng ký handler cho một route, áp dụng cho một tập HTTP method cụ thể.

#### Rules
- Closure được trả về append các handler của nó vào chuỗi `Handlers` của mọi method trong `methodInclusions`, cho `route`/`version` đã cho (`TestRouterMiddleware`: `r0.For(HTTPMethods, "/test0", "")(handler1)` thêm `handler1` vào chuỗi của mọi HTTP method cho `/test0`).
- Nếu `GlobalMiddlewares` đã được đặt qua `Use` trước khi có handler nào tồn tại cho route đó, các handler được append sẽ nằm sau chúng trong chuỗi (`TestRouterMiddleware`, router 1).

#### Parameters
- Tham số thứ 1: `[]string` (`methodInclusions`)

- Mô tả: Các HTTP method cần đăng ký handler dưới.

- Tham số thứ 2: `string` (`route`)

- Mô tả: Route path.

- Tham số thứ 3: `string` (`version`)

- Mô tả: Version tag; `""` nếu không có version.

#### Returns
- Giá trị thứ 1: `func(handlers ...ctx.Handler) *Router`

- Mô tả: Một hàm mà khi được gọi với các handler, sẽ đăng ký chúng và trả về receiver để gọi nối tiếp.

#### Cách Dùng

```go
r.For([]string{http.MethodGet, http.MethodPost}, "/users/{id}", "")(func(c *ctx.HTTPContext) {
	c.Next()
})
```

### Add

Đăng ký handler chính cho một tổ hợp `(method, route, version)` duy nhất, chèn route vào trie nhúng (embedded).

#### Rules
- Mỗi lần gọi `Add` chèn route vào trie nhúng của router, nên `Len()` của trie (được promote từ `ds.Trie`) phản ánh tổng số segment path khác biệt trên mọi route đã được thêm cho tới hiện tại, với các tiền tố (prefix) chung chỉ được tính một lần (`TestRouteAdd`: 4 route có chung tiền tố cho ra `Len() == 11`).
- Handler được truyền vào `Add` được append sau bất kỳ handler nào đã được `Use`/`For` đóng góp cho đúng method/route/version đó (`TestRouterMiddleware`, router 0 và router 1).

#### Parameters
- Tham số thứ 1: `string` (`method`)

- Mô tả: HTTP method cần đăng ký handler dưới.

- Tham số thứ 2: `string` (`route`)

- Mô tả: Route path, ví dụ `/users/{id}`.

- Tham số thứ 3: `string` (`version`)

- Mô tả: Version tag; `""` nếu không có version.

- Tham số thứ 4: `ctx.Handler` (`handler`)

- Mô tả: Handler chính cho route này.

#### Returns
- Giá trị thứ 1: `*Router`

- Mô tả: Receiver, để có thể gọi nối tiếp (chain).

#### Cách Dùng

```go
r.Add(http.MethodGet, "/users/{id}", "", func(c *ctx.HTTPContext) {})
```

### AddInjectableHandler

Đăng ký một handler được resolve bằng reflection (thay vì theo signature cố định `ctx.Handler`) cho một route, dùng cho DI container.

#### Rules
- Lưu `handler` vào `InjectableHandlers`, đánh khóa bằng `MethodRouteVersionToPattern(method, route, version)`, và cũng gọi `Add` để route trở nên có thể khớp được qua `Match` (`TestAddInjectableHandler`).
- Panic nếu `handler` là `nil` (`TestAddInjectableHandlerPanicsOnNil`).
- Panic nếu `handler` không phải là một function, dù không phải `nil` (`TestAddInjectableHandlerPanicsOnNonFunc`).

#### Parameters
- Tham số thứ 1: `string` (`method`)

- Mô tả: HTTP method cần đăng ký handler dưới.

- Tham số thứ 2: `string` (`route`)

- Mô tả: Route path.

- Tham số thứ 3: `string` (`version`)

- Mô tả: Version tag; `""` nếu không có version.

- Tham số thứ 4: `any` (`handler`)

- Mô tả: Handler cần lưu; phải là một function không phải `nil`.

#### Returns
- Giá trị thứ 1: `*Router`

- Mô tả: Receiver, để có thể gọi nối tiếp (chain).

#### Cách Dùng

```go
r.AddInjectableHandler(http.MethodGet, "/users/{userId}", "", func(userID string) {})
```

## Benchmark

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./routing/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/routing
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkPatternToMethodRouteVersion-12    	 7198441	       159.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkParseToParamKey-12                	197307458	         6.123 ns/op	       0 B/op	       0 allocs/op
BenchmarkToEndpoint-12                     	12655910	        88.90 ns/op	      32 B/op	       1 allocs/op
BenchmarkMethodRouteVersionToPattern-12    	 6494689	       185.7 ns/op	      60 B/op	       3 allocs/op
BenchmarkRouterAdd-12                      	  375076	      4246 ns/op	    2234 B/op	      23 allocs/op
BenchmarkRouterMatch_Static-12             	 5007302	       207.5 ns/op	      16 B/op	       1 allocs/op
BenchmarkRouterMatch_Param-12              	 2435484	       461.2 ns/op	     112 B/op	       2 allocs/op
BenchmarkRouterMatch_NoMatch-12            	   57026	     20988 ns/op	      24 B/op	       1 allocs/op
BenchmarkRouterUse-12                      	   10000	    119744 ns/op	   79594 B/op	     801 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/routing	18.081s
```
