# Package DS

*`ds` là package nội bộ (internal) hiện thực cấu trúc trie phân đoạn (segment-based trie) mà router của Ginject dùng để đăng ký route và khớp (match) đường dẫn request.*

- [Package DS](#package-ds)
  - [Tính Năng Chính](#tính-năng-chính)
  - [Cách Dùng](#cách-dùng)
  - [Kiểu `Node`](#kiểu-node)
  - [Struct `Trie`](#struct-trie)
    - [IsEnd](#isend)
    - [Raw](#raw)
    - [Children](#children)
  - [Hàm](#hàm)
    - [NewTrie](#newtrie)
  - [Phương Thức Của `*Trie`](#phương-thức-của-trie)
    - [Len](#len)
    - [Insert](#insert)
    - [Remove](#remove)
    - [Find](#find)
    - [ToJSON](#tojson)
  - [Benchmark](#benchmark)

## Tính Năng Chính
- Phân đoạn (segment) đường dẫn theo một byte phân tách (separator) do người gọi tự chỉ định, thay vì luôn giả định là `/`
- Ba loại segment: văn bản literal, `$` cho segment tham số (param) được capture, và `*` cho segment wildcard
- Khớp tham số `$` là tùy chọn (opt-in) theo từng lần gọi `Find`: truyền `false` sẽ bỏ qua hoàn toàn việc check node con `$` (trường hợp phổ biến của các package như `broker`/`broker1` chưa từng đăng ký segment `$`); truyền `true` mới bật check này
- `Find` trả về kết quả khớp chính xác cùng với phương án dự phòng (fallback) wildcard tốt nhất trong một lần duyệt
- `Remove` hủy đăng ký một đường dẫn đã chèn trước đó và dọn (prune) mọi node tổ tiên trở nên "chết" sau khi xóa
- `ToJSON` xuất cấu trúc của trie ra để debug hoặc trực quan hóa

## Cách Dùng

Một `Trie` lưu một đường dẫn dưới hai chuỗi: chuỗi `raw` (được trả về khi khớp) và chuỗi thực sự được duyệt để xây dựng trie, trong đó `$` đánh dấu segment động (dynamic) và `*` đánh dấu segment wildcard. Cả lúc chèn (insert) và tra cứu (lookup) đều phải dùng cùng một byte phân tách. Để khớp segment `$` như tham số được capture, cần truyền `true` cho tham số thứ 3 của `Find` — truyền `false` sẽ coi `$` chỉ là một segment literal bình thường:

```go
package main

import (
	"fmt"

	"github.com/dangduoc08/ginject/internal/ds"
)

func main() {
	tr := ds.NewTrie()

	tr.Insert("/users/:id/", "/users/$/", '/')
	tr.Insert("/users/:id/friends/", "/users/$/friends/", '/')

	raw, wildcardRaw, params := tr.Find("/users/123/", '/', true)

	fmt.Println("matched raw:", raw)
	fmt.Println("wildcard raw:", wildcardRaw)
	fmt.Println("params:", params)
}
```

Console:
```console
matched raw: /users/:id/
wildcard raw:
params: [123]
```

## Kiểu `Node`
Type: `map[string]*Trie`

`Node` là kiểu map nằm sau field `Trie.Children`. Mỗi key là một segment của đường dẫn — có thể là token literal, hoặc một trong hai token đặc biệt `$` (tham số được capture) và `*` (wildcard) — ánh xạ tới node `Trie` con tương ứng với segment đó.

## Struct `Trie`

### IsEnd
Type: `bool`

Default: `false`

Required: `false`

Đánh dấu node kết thúc (terminate) một đường dẫn đã chèn. `NewTrie` để giá trị này ở mặc định `false`, và `Find` coi đây là "chưa có route nào được đăng ký tại đây." `Insert` chỉ đặt giá trị này thành `true` trên node tương ứng với segment cuối cùng của chuỗi được chèn; `Remove` đặt lại nó về `false`.

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
- Trả về một trie với `IsEnd` được đặt là `false` và `Children` là một map trống, không nil; gọi `Len()` ngay sau đó sẽ trả về `0` (`TestTrieLenEmpty`).

#### Parameters
Không có.

#### Returns
- Giá trị thứ 1: `*Trie`

- Mô tả: Một trie mới với `IsEnd` được đặt là `false` và `Children` là một map trống.

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
tr.Insert("/users/{userId}/", "/users/{userId}/", '/')
tr.Insert("/feeds/all/", "/feeds/all/", '/')
tr.Insert("/users/{userId}/friends/all/", "/users/{userId}/friends/all/", '/')

fmt.Println(tr.Len())
```

Console:
```console
6
```

### Insert

Tách (split) `insertedStr` theo `sep` và duyệt/tạo một node con cho mỗi segment, lưu `raw` và đánh dấu `IsEnd` trên node của segment cuối cùng. Dùng segment literal `$` để đánh dấu một segment động (tham số) và `*` để đánh dấu một segment wildcard. Trả về receiver, nên có thể gọi nối tiếp (chain) nhiều lệnh gọi.

#### Rules
- Chỉ node của segment cuối cùng trong `insertedStr` có `IsEnd` được đặt thành `true` và nhận `raw` được truyền vào; mọi node segment trung gian giữ giá trị `IsEnd` mặc định là `false`, trừ khi chính segment đó cũng là segment cuối cùng của một đường dẫn khác được chèn riêng (`TestTrieInsert`).
- Chèn các đường dẫn có tiền tố chung sẽ dùng lại các node đã có cho tiền tố đó thay vì tạo node trùng lặp (`TestTrieInsert`, `TestTrieLen`).

#### Parameters
- Tham số thứ 1: `string` (`raw`)

- Mô tả: Chuỗi route gốc cần lưu trên node đã khớp; được `Find` trả về sau này.

- Tham số thứ 2: `string` (`insertedStr`)

- Mô tả: Chuỗi thực sự được phân đoạn và duyệt để xây dựng đường đi trong trie. Dùng segment `$` và `*` cho tham số và wildcard.

- Tham số thứ 3: `byte` (`sep`)

- Mô tả: Byte phân tách dùng để tách `insertedStr` thành các segment.

#### Returns
- Giá trị thứ 1: `*Trie`

- Mô tả: Receiver trie, được trả về để cho phép nối tiếp (chain) thêm các lệnh gọi `Insert`.

#### Cách Dùng

```go
tr := ds.NewTrie()
tr.
	Insert("/users/:id/", "/users/$/", '/').
	Insert("/feeds/all/", "/feeds/all/", '/')
```

### Remove

Duyệt `removedStr` theo từng segment dựa trên `sep`, chỉ đi theo các node con đã tồn tại (không tạo node mới). Nếu đường đi không dẫn tới một node đã từng được `Insert` đánh dấu kết thúc (tức `IsEnd == true`), trie giữ nguyên không đổi và `Remove` trả về `false`. Ngược lại, hàm xóa `IsEnd`/`Raw` của node đó, rồi duyệt ngược lên theo đúng đường đi, xóa mọi node tổ tiên hiện đã vừa không còn con vừa không còn là node kết thúc, dừng lại ở tổ tiên đầu tiên còn giữ con hoặc chính nó là một node `IsEnd`.

Giống `Insert` và `Find`, `Remove` không có lock nội bộ — nó thay đổi cùng các map `Children` mà `Insert` viết và `Find` đọc, nên nếu một goroutine gọi `Remove` đồng thời với `Insert`/`Find` trên cùng một `*Trie`, caller phải tự khóa (lock) bên ngoài cho cả ba hàm này (`TestTrieConcurrentRemoveAndFind_RequiresExternalLock` ghi nhận điều này bằng cách bị `go test -race` báo lỗi có chủ đích).

#### Rules
- Chỉ đường dẫn đã từng là đối tượng của một lệnh gọi `Insert` (node có `IsEnd == true`) mới có thể bị xóa; gọi `Remove` trên đường dẫn chưa từng được chèn, trên một đường dẫn trung gian chưa hoàn chỉnh, hoặc với input không hợp lệ (chuỗi rỗng, không có separator) sẽ giữ trie không đổi và trả về `false` (`TestTrieRemove_NoMatch_ReturnsFalse`).
- Xóa một đường dẫn sẽ dọn (prune) mọi segment tổ tiên trở nên vừa không còn con vừa không còn là node kết thúc, lên tới (nhưng không bao gồm) root — xóa đường dẫn duy nhất trong trie sẽ đưa `Len()` về `0` (`TestTrieRemove_PrunesDeadBranch`).
- Một segment tổ tiên vẫn còn là một phần của đường dẫn khác (tiền tố chung) sẽ không bao giờ bị dọn, dù đường dẫn đang xóa là một nhánh con của nó (`TestTrieRemove_KeepsSharedPrefix`).
- Xóa cùng một đường dẫn hai lần sẽ trả về `true` ở lần đầu và `false` ở lần thứ hai (`TestTrieRemove_AlreadyRemoved_ReturnsFalse`).

#### Parameters
- Tham số thứ 1: `string` (`removedStr`)

- Mô tả: Đường dẫn đã chèn trước đó cần xóa, được phân đoạn theo đúng cách `insertedStr` đã dùng lúc chèn.

- Tham số thứ 2: `byte` (`sep`)

- Mô tả: Byte phân tách dùng để tách `removedStr` thành các segment; phải khớp với separator đã dùng lúc chèn đường dẫn đó.

#### Returns
- Giá trị thứ 1: `bool`

- Mô tả: `true` nếu tìm thấy và xóa được một đường dẫn đã đăng ký; `false` nếu đường dẫn đó không tồn tại.

#### Cách Dùng

```go
tr := ds.NewTrie()
tr.Insert("/users/:id/", "/users/$/", '/')

ok := tr.Remove("/users/$/", '/')
fmt.Println(ok, tr.Len())
```

Console:
```console
true 0
```

### Find

Duyệt `path` theo từng segment dựa trên `sep`, ưu tiên khớp literal chính xác ở mỗi cấp, sau đó — chỉ khi `supportParams` là `true` — tới node con `$` (tham số), rồi tới node con `*` (wildcard), và cuối cùng dự phòng (fallback) bằng cách so khớp với bất kỳ segment anh em (sibling) nào chứa pattern `*` literal (ví dụ `*.html`). Trong khi duyệt, hàm cũng theo dõi node `*` cụ thể nhất đã đi qua, để vẫn có phương án wildcard dự phòng ngay cả khi không tìm được khớp chính xác.

#### Rules
- Đường dẫn phải được duyệt hết đúng tới một node kết thúc mới được coi là khớp: một đường dẫn chỉ là tiền tố chưa đầy đủ của route đã đăng ký sẽ trả về `""` cho cả `matchedRaw` và `wildcardRaw` (`TestTrieFind`, case "incomplete path should not match").
- Với `supportParams: true`, segment `$` capture giá trị literal của đường dẫn vào `paramVals`, theo đúng thứ tự duyệt từ trái sang phải (`TestTrieFind`, case "deep param match"); với `supportParams: false`, `Find` không bao giờ tìm node con `$`, nên một segment `$` đã chèn chỉ khớp khi segment truy vấn đúng là chuỗi literal `"$"` (`TestTrieFind_ParamSupportDisabled`, `TestTrieFind_ParamSupportEnabled`).
- Khi đường dẫn đã đi qua một node `*`, `Raw` của node đó được trả về qua `wildcardRaw`, và kết quả khớp này vẫn giữ nguyên dù đường dẫn có thêm các segment dư ở cuối vượt quá độ dài của route wildcard (`TestTrieFind`, case "wildcard deep match, extra trailing segments").
- Một khớp wildcard được dùng làm phương án dự phòng ngay cả khi có một route anh em (sibling) không liên quan, ở nhánh sâu hơn, không khớp với đường dẫn (`TestTrieFindWildcardFallbackThroughUnrelatedSibling`).

#### Parameters
- Tham số thứ 1: `string` (`path`)

- Mô tả: Đường dẫn cần tra cứu, dùng cùng byte phân tách đã dùng lúc chèn (insert).

- Tham số thứ 2: `byte` (`sep`)

- Mô tả: Byte phân tách dùng để tách `path` thành các segment.

- Tham số thứ 3: `bool` (`supportParams`)

- Mô tả: Truyền `true` để check node con `$` ở mỗi segment và capture giá trị của nó; truyền `false` để bỏ qua hoàn toàn việc check đó và coi `$` chỉ là literal bình thường.

#### Returns
- Giá trị thứ 1: `string`

- Mô tả: `Raw` của node khớp chính xác toàn bộ đường dẫn, hoặc `""` nếu không có khớp chính xác.

- Giá trị thứ 2: `string`

- Mô tả: `Raw` của node wildcard (`*`) cụ thể nhất đã gặp trong quá trình duyệt đường dẫn, hoặc `""` nếu không đi qua node wildcard nào.

- Giá trị thứ 3: `[]string`

- Mô tả: Các giá trị được capture cho mỗi segment `$`, theo đúng thứ tự được khớp. Luôn là `nil` khi `supportParams` là `false`.

#### Cách Dùng

```go
tr := ds.NewTrie()
tr.Insert("/users/:id/", "/users/$/", '/')

raw, wildcardRaw, params := tr.Find("/users/123/", '/', true)
fmt.Println(raw, wildcardRaw, params)
```

Console:
```console
/users/:id/  [123]
```

### ToJSON

Chuyển cấu trúc của trie — path, `IsEnd` và children của từng segment — thành một chuỗi JSON. Vì `Children` là một map trong Go, thứ tự các phần tử anh em (sibling) trong kết quả không được đảm bảo ổn định giữa các lần gọi (các key trong mỗi object JSON luôn là `children`, `isEnd`, `path`, được `encoding/json` sắp xếp theo thứ tự alphabet).

#### Rules
- Object JSON của node gốc (root) không có key `"path"`, chỉ có `"children"`; mọi node khác đều có `"path"` (chính là key segment của nó), `"isEnd"`, và `"children"` (`TestTrieToJSON`).

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
tr.Insert("/users/$/", "/users/$/", '/')
tr.Insert("/feeds/all/", "/feeds/all/", '/')

js, err := tr.ToJSON()
if err != nil {
	panic(err)
}
fmt.Println(js)
```

Console (một trong các thứ tự khả dĩ — thứ tự sibling có thể thay đổi):
```console
{"children":[{"children":[{"children":[],"isEnd":true,"path":"$"}],"isEnd":false,"path":"users"},{"children":[{"children":[],"isEnd":true,"path":"all"}],"isEnd":false,"path":"feeds"}]}
```

## Benchmark

Được ghi lại bằng cách chạy `go test -run=^$ -bench=. -benchmem ./internal/ds/...`. Các số liệu phụ thuộc vào máy chạy và được ghi lại tại thời điểm tạo tài liệu — hãy tự chạy lại lệnh này để có baseline mới.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/internal/ds
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkMatchWildcard-12         	47284537	        25.25 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_Static-12       	14455305	        81.98 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_WithParam-12    	 6470072	       182.5 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_DeepParam-12    	 3634706	       318.3 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_NoMatch-12      	 6769974	       182.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieRemove-12            	 1916504	       567.6 ns/op	     144 B/op	       2 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/internal/ds	11.808s
```
