# Package DS

*`ds` là package nội bộ (internal) hiện thực cấu trúc trie phân đoạn (segment-based trie) mà router của Ginject dùng để đăng ký route và khớp (match) đường dẫn request.*

- [Package DS](#package-ds)
  - [Tính Năng Chính](#tính-năng-chính)
  - [Cách Dùng](#cách-dùng)
  - [Kiểu `Node`](#kiểu-node)
  - [Struct `Trie`](#struct-trie)
    - [Index](#index)
    - [Raw](#raw)
    - [Children](#children)
  - [Hàm](#hàm)
    - [NewTrie](#newtrie)
  - [Phương Thức Của `*Trie`](#phương-thức-của-trie)
    - [Len](#len)
    - [Insert](#insert)
    - [Find](#find)
    - [ToJSON](#tojson)
  - [Benchmark](#benchmark)

## Tính Năng Chính
- Phân đoạn (segment) đường dẫn theo một byte phân tách (separator) do người gọi tự chỉ định, thay vì luôn giả định là `/`
- Ba loại segment: văn bản literal, `$` cho segment tham số (param) được capture, và `*` cho segment wildcard
- `Find` trả về kết quả khớp chính xác cùng với phương án dự phòng (fallback) wildcard tốt nhất trong một lần duyệt
- `ToJSON` xuất cấu trúc của trie ra để debug hoặc trực quan hóa

## Cách Dùng

Một `Trie` lưu một đường dẫn dưới hai chuỗi: chuỗi `raw` (được trả về khi khớp) và chuỗi thực sự được duyệt để xây dựng trie, trong đó `$` đánh dấu segment động (dynamic) và `*` đánh dấu segment wildcard. Cả lúc chèn (insert) và tra cứu (lookup) đều phải dùng cùng một byte phân tách:

```go
package main

import (
	"fmt"

	"github.com/dangduoc08/ginject/internal/ds"
)

func main() {
	tr := ds.NewTrie()

	tr.Insert("/users/:id/", "/users/$/", '/', 0)
	tr.Insert("/users/:id/friends/", "/users/$/friends/", '/', 1)

	index, raw, wildcardIndex, wildcardRaw, params := tr.Find("/users/123/", '/')

	fmt.Println("matched index:", index)
	fmt.Println("matched raw:", raw)
	fmt.Println("wildcard index:", wildcardIndex)
	fmt.Println("wildcard raw:", wildcardRaw)
	fmt.Println("params:", params)
}
```

Console:
```console
matched index: 0
matched raw: /users/:id/
wildcard index: -1
wildcard raw:
params: [123]
```

## Kiểu `Node`
Type: `map[string]*Trie`

`Node` là kiểu map nằm sau field `Trie.Children`. Mỗi key là một segment của đường dẫn — có thể là token literal, hoặc một trong hai token đặc biệt `$` (tham số được capture) và `*` (wildcard) — ánh xạ tới node `Trie` con tương ứng với segment đó.

## Struct `Trie`

### Index
Type: `int`

Default: `-1`

Required: `false`

Định danh được lưu trên node kết thúc (terminate) một đường dẫn đã chèn. `NewTrie` khởi tạo giá trị này là `-1`, và `Find` coi đây là "chưa có route nào được đăng ký tại đây." `Insert` chỉ ghi đè giá trị này (bằng tham số `index` của nó) lên node tương ứng với segment cuối cùng của chuỗi được chèn.

### Raw
Type: `string`

Default: `""`

Required: `false`

Chuỗi route gốc (raw) được truyền vào làm tham số đầu tiên của `Insert`, lưu trên node kết thúc đường dẫn đó. `Find` trả về giá trị này dưới dạng `matchedRaw`/`wildcardRaw` để có thể khôi phục lại pattern gốc (trước khi parse) sau khi khớp.

### Children
Type: `Node`

Default: một map trống, không nil (`make(Node)`, được khởi tạo bởi `NewTrie`)

Required: `false`

Các segment con của node, được đánh khóa (key) bằng văn bản literal hoặc bằng các token đặc biệt `$`/`*`.

## Hàm

### NewTrie

Tạo một trie trống, sẵn sàng cho `Insert` và `Find`.

#### Rules
- Trả về một trie với `Index` được đặt là `-1` và `Children` là một map trống, không nil; gọi `Len()` ngay sau đó sẽ trả về `0` (`TestTrieLenEmpty`).

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `*Trie`

- Mô tả: Một trie mới với `Index` được đặt là `-1` và `Children` là một map trống.

#### Cách Dùng

```go
tr := ds.NewTrie()
```

## Phương Thức Của `*Trie`

### Len

Đếm số lượng node trong trie bên dưới receiver, tức là một node cho mỗi segment đường dẫn đã được chèn trên tất cả các route (không chỉ riêng các node lá/leaf).

#### Rules
- `Len()` của một trie trống là `0` (`TestTrieLenEmpty`).
- `Len()` đếm một node cho mỗi segment khác biệt trên tất cả các đường dẫn đã chèn; các segment được nhiều route dùng chung (tiền tố chung) chỉ được tính một lần, không tính theo từng route — ba route có chung tiền tố cho ra `Len() == 6`, không phải `3` (`TestTrieLen`).

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `int`

- Mô tả: Tổng số node con cháu (descendant).

#### Cách Dùng

```go
tr := ds.NewTrie()
tr.Insert("/users/{userId}/", "/users/{userId}/", '/', -1)
tr.Insert("/feeds/all/", "/feeds/all/", '/', -1)
tr.Insert("/users/{userId}/friends/all/", "/users/{userId}/friends/all/", '/', -1)

fmt.Println(tr.Len())
```

Console:
```console
6
```

### Insert

Tách (split) `insertedStr` theo `sep` và duyệt/tạo một node con cho mỗi segment, lưu `raw` và `index` trên node của segment cuối cùng. Dùng segment literal `$` để đánh dấu một segment động (tham số) và `*` để đánh dấu một segment wildcard. Trả về receiver, nên có thể gọi nối tiếp (chain) nhiều lệnh gọi.

#### Rules
- Chỉ node của segment cuối cùng trong `insertedStr` nhận `index` và `raw` được truyền vào; mọi node segment trung gian giữ giá trị `Index` mặc định là `-1`, trừ khi chính segment đó cũng là segment cuối cùng của một đường dẫn khác được chèn riêng (`TestTrieInsert`).
- Chèn các đường dẫn có tiền tố chung sẽ dùng lại các node đã có cho tiền tố đó thay vì tạo node trùng lặp (`TestTrieInsert`, `TestTrieLen`).

#### Parameters
- Tham số thứ 1: `string` (`raw`)

- Mô tả: Chuỗi route gốc cần lưu trên node đã khớp; được `Find` trả về sau này.

- Tham số thứ 2: `string` (`insertedStr`)

- Mô tả: Chuỗi thực sự được phân đoạn và duyệt để xây dựng đường đi trong trie. Dùng segment `$` và `*` cho tham số và wildcard.

- Tham số thứ 3: `byte` (`sep`)

- Mô tả: Byte phân tách dùng để tách `insertedStr` thành các segment.

- Tham số thứ 4: `int` (`index`)

- Mô tả: Định danh cần lưu trên node của segment cuối cùng; được `Find` trả về sau này.

#### Returns
- Giá trị thứ 1: `*Trie`

- Mô tả: Receiver trie, được trả về để cho phép nối tiếp (chain) thêm các lệnh gọi `Insert`.

#### Cách Dùng

```go
tr := ds.NewTrie()
tr.
	Insert("/users/:id/", "/users/$/", '/', 0).
	Insert("/feeds/all/", "/feeds/all/", '/', 1)
```

### Find

Duyệt `path` theo từng segment dựa trên `sep`, ưu tiên khớp literal chính xác ở mỗi cấp, sau đó tới node con `$` (tham số), rồi tới node con `*` (wildcard), và cuối cùng dự phòng (fallback) bằng cách so khớp với bất kỳ segment anh em (sibling) nào chứa pattern `*` literal (ví dụ `*.html`). Trong khi duyệt, hàm cũng theo dõi node `*` cụ thể nhất đã đi qua, để vẫn có phương án wildcard dự phòng ngay cả khi không tìm được khớp chính xác.

#### Rules
- Đường dẫn phải được duyệt hết đúng tới một node kết thúc mới được coi là khớp: một đường dẫn chỉ là tiền tố chưa đầy đủ của route đã đăng ký sẽ trả về `""` cho cả `matchedRaw` và `wildcardRaw` (`TestTrieFind`, case "incomplete path should not match").
- Các segment `$` capture giá trị literal của đường dẫn vào `paramVals`, theo đúng thứ tự duyệt từ trái sang phải (`TestTrieFind`, case "deep param match").
- Khi đường dẫn đã đi qua một node `*`, `Index`/`Raw` của node đó được trả về qua `wildcardIndex`/`wildcardRaw`, và kết quả khớp này vẫn giữ nguyên dù đường dẫn có thêm các segment dư ở cuối vượt quá độ dài của route wildcard (`TestTrieFind`, case "wildcard deep match, extra trailing segments").
- Một khớp wildcard được dùng làm phương án dự phòng ngay cả khi có một route anh em (sibling) không liên quan, ở nhánh sâu hơn, không khớp với đường dẫn (`TestTrieFindWildcardFallbackThroughUnrelatedSibling`).

#### Parameters
- Tham số thứ 1: `string` (`path`)

- Mô tả: Đường dẫn cần tra cứu, dùng cùng byte phân tách đã dùng lúc chèn (insert).

- Tham số thứ 2: `byte` (`sep`)

- Mô tả: Byte phân tách dùng để tách `path` thành các segment.

#### Returns
- Giá trị thứ 1: `int`

- Mô tả: `Index` của node khớp chính xác toàn bộ đường dẫn, hoặc `-1` nếu không có khớp chính xác.

- Giá trị thứ 2: `string`

- Mô tả: `Raw` của node khớp chính xác đó, hoặc `""` nếu không có khớp chính xác.

- Giá trị thứ 3: `int`

- Mô tả: `Index` của node wildcard (`*`) cụ thể nhất đã gặp trong quá trình duyệt đường dẫn, hoặc `-1` nếu không đi qua node wildcard nào.

- Giá trị thứ 4: `string`

- Mô tả: `Raw` của node wildcard đó, hoặc `""` nếu không đi qua node wildcard nào.

- Giá trị thứ 5: `[]string`

- Mô tả: Các giá trị được capture cho mỗi segment `$`, theo đúng thứ tự được khớp.

#### Cách Dùng

```go
tr := ds.NewTrie()
tr.Insert("/users/:id/", "/users/$/", '/', 0)

index, raw, wildcardIndex, wildcardRaw, params := tr.Find("/users/123/", '/')
fmt.Println(index, raw, wildcardIndex, wildcardRaw, params)
```

Console:
```console
0 /users/:id/ -1  [123]
```

### ToJSON

Chuyển cấu trúc của trie — path, `Index` và children của từng segment — thành một chuỗi JSON. Vì `Children` là một map trong Go, thứ tự các phần tử anh em (sibling) trong kết quả không được đảm bảo ổn định giữa các lần gọi (các key trong mỗi object JSON luôn là `children`, `index`, `path`, được `encoding/json` sắp xếp theo thứ tự alphabet).

#### Rules
- Object JSON của node gốc (root) không có key `"path"`, chỉ có `"children"`; mọi node khác đều có `"path"` (chính là key segment của nó), `"index"`, và `"children"` (`TestTrieToJSON`).

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `string`

- Mô tả: Biểu diễn JSON của trie.

- Giá trị thứ 2: `error`

- Mô tả: Khác nil nếu việc marshal JSON gặp lỗi.

#### Cách Dùng

```go
tr := ds.NewTrie()
tr.Insert("/users/$/", "/users/$/", '/', 0)
tr.Insert("/feeds/all/", "/feeds/all/", '/', 1)

js, err := tr.ToJSON()
if err != nil {
	panic(err)
}
fmt.Println(js)
```

Console (một trong các thứ tự khả dĩ — thứ tự sibling có thể thay đổi):
```console
{"children":[{"children":[{"children":[],"index":0,"path":"$"}],"index":-1,"path":"users"},{"children":[{"children":[],"index":1,"path":"all"}],"index":-1,"path":"feeds"}]}
```

## Benchmark

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./internal/ds/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/internal/ds
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkMatchWildcard-12         	56382829	        22.31 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_Static-12       	18563512	        64.83 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_WithParam-12    	 7714236	       149.9 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_DeepParam-12    	 4847652	       256.6 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_NoMatch-12      	 6658142	       179.4 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/internal/ds	7.329s
```
