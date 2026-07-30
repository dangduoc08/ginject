# Routing System

**Optimization**: Decision trees, pseudo-algorithms, explicit rules. Minimal explanatory prose.

## 1. Route Naming Convention (Method Names → HTTP Routes)

### 1.1 Base Tokens (HTTP Methods)

| Token | HTTP Method | Example |
|-------|-------------|---------|
| `READ` | GET | `READ()` → `GET /` |
| `CREATE` | POST | `CREATE()` → `POST /` |
| `UPDATE` | PUT | `UPDATE()` → `PUT /` |
| `MODIFY` | PATCH | `MODIFY()` → `PATCH /` |
| `DELETE` | DELETE | `DELETE()` → `DELETE /` |
| `PREFLIGHT` | OPTIONS | `PREFLIGHT()` → `OPTIONS /` |

**Rule**: Exactly one base token per method name. If missing, method is NOT a route handler.

### 1.2 Separator Tokens (Path Construction)

| Token | Meaning | Example |
|-------|---------|---------|
| `BY` | Path parameter | `READ_BY_ID` → `GET /:id` |
| `AND` | Additional path segment | `READ_BY_ID_AND_NAME` → `GET /:id/:name` |
| `OF` | Sub-resource | `READ_OF_POSTS` → `GET /posts` |
| `ANY` | Wildcard segment | `READ_ANY` → `GET /*` |
| `FILE` | File serving | `READ_FILE` → `GET /*` (static files) |

**Examples**:
- `READ_BY_ID` → `GET /:id`
- `READ_BY_ID_AND_PAGE` → `GET /:id/:page`
- `READ_OF_COMMENTS` → `GET /comments`
- `CREATE_COMMENT` → `POST /comment`
- `DELETE_BY_ID_AND_VERSION` → `DELETE /:id/:version`
- `READ_ANY` → `GET /*` (catch-all)
- `READ_FILE_ANY` → `GET /*` (static file serving)

### 1.3 Versioning Token

| Token | Meaning | Example |
|-------|---------|---------|
| `VERSION_X` | API version | `READ_VERSION_1` → `GET /v1/...` |
| `VERSION_X_BY_ID` | Versioned with param | `READ_VERSION_2_BY_ID` → `GET /v2/:id` |

**Rule**: Version tag is appended AFTER all other tokens. Examples:
- `READ_VERSION_1` → version 1 of READ
- `READ_BY_ID_VERSION_2` → version 2 of READ/:id
- `CREATE_OF_USERS_VERSION_3` → version 3 of POST /users

### 1.4 Reserved/Special Tokens

| Token | Special Meaning |
|-------|-----------------|
| `PREFLIGHT` | Generates OPTIONS method |
| `SERVE` | Static file serving (GET only) |
| `FILE` | File upload/download marker |

---

## 2. Route Matching Algorithm

### 2.1 Matching Decision Tree

```
request.URL.Path = "/api/users/123"
request.Method = "GET"

Router.Match(path, method):
    1. Exact match in trie?
       YES → return RouterItem[method]
       NO → continue
    
    2. Prefix wildcard match?
       Example: "/api/users/:userId" matches "/api/users/123"
       YES → return RouterItem[method]
       NO → continue
    
    3. Complex regex match?
       Example: pattern "/api/(.+)/profile" matches "/api/john/profile"
       YES → return RouterItem[method]
       NO → continue
    
    4. No match found
       → return 404 Not Found
```

### 2.2 Trie-Based Prefix Matching

**Performance**: O(1) for exact path, O(log n) for prefix matching

**Structure**:
```
Trie node per path segment:
  "/" → trie node
    "api" → trie node
      "users" → trie node
        ":userId" → trie node (dynamic)
```

**Segment Types**:
- **Static**: "api" (matches exactly)
- **Dynamic**: ":userId" (matches any single segment)
- **Wildcard**: "*" (matches remaining path)

### 2.3 Conflict Detection

**Route Conflicts** occur when:
1. Two routes with identical path + method + version
2. Two dynamic routes that could both match the same request

**Example Conflicts**:
```
READ_BY_ID_AND_NAME        → GET /:id/:name
READ_BY_ID_AND_COMMENT     → GET /:id/:comment

// Both match GET /123/456 — CONFLICT!
```

**Behavior**: Panic at app.Create() with `route conflict: ...` message

---

## 3. RouterItem Structure

```go
type RouterItem struct {
    Method       string              // "GET", "POST", etc.
    Version      string              // API version (empty if unversioned)
    Pattern      string              // Full route pattern
    Index        int                 // Trie leaf index
    HandlerIndex int                 // Handler function index
    Handlers     []ctx.HTTPHandler   // Middleware + handler chain
    ParamKeys    map[string][]int    // Path param positions
}
```

**Usage**:
1. Router.Match() returns RouterItem
2. Item.Handlers executed sequentially (middleware chain)
3. Item.ParamKeys used to extract and store path params

---

## 4. Path Parameter Extraction

### 4.1 Named Parameters

**Pattern**: `READ_BY_ID` → `GET /:id`

**In Handler**:
```go
func (c *UserController) READ_BY_ID(httpCtx *ctx.HTTPContext, params ctx.Param) string {
    id := params.GetInt("id")  // ":id" → integer
    return fmt.Sprintf("User %d", id)
}
```

**Extraction Mechanism**:
1. Route pattern defines param names: `:id`, `:userId`, etc.
2. Request path matches pattern: "/users/123"
3. RouterItem.ParamKeys maps name → position in path
4. Param middleware extracts and stores in context
5. `ctx.Param` interface provides getters: `Get()`, `GetInt()`, `GetString()`, etc.

### 4.2 Wildcard Matching

**Pattern**: `READ_ANY` → `GET /*`

**Behavior**:
- Matches any path starting with parent route
- Remaining path available via param: `*`
- Example: "GET /files/docs/readme.txt" → param["*"] = "docs/readme.txt"

---

## 5. Version-Specific Routes

### 5.1 Versioning Mechanism

**Pattern**: `READ_VERSION_1` → `GET /v1/...`

**Registration**:
```go
// Method names define versions implicitly
func (c *UserController) READ_VERSION_1() { /* v1 handler */ }
func (c *UserController) READ_VERSION_2() { /* v2 handler */ }
```

**Behavior**:
- Same path ("/users"), different handlers per version
- Version tag added to URL prefix automatically
- Request "/v1/users" → routes to READ_VERSION_1
- Request "/v2/users" → routes to READ_VERSION_2

### 5.2 Version Precedence

If both versioned and unversioned routes exist:
- Versioned routes take precedence
- Unversioned falls back if version not matched

---

## 6. Static File Serving

### 6.1 FILE Token

**Pattern**: `READ_FILE_ANY` → `GET /*` (static file serving)

**Behavior**:
- Serves files from filesystem
- Path after route prefix = file path
- Example: "GET /static/img.png" → serves from `static/img.png`

**Router Handling**:
- Marks route as static file (uses `http.FileServer` under hood)
- No dependency injection in static routes
- Bypasses normal handler pipeline

---

## 7. Route Registration Process

### 7.1 Auto-Discovery at app.Create()

**Process**:
```
1. Collect all controllers from module tree
2. For each controller:
   a. Reflect on all methods
   b. For each method:
      i. Check if method name matches route pattern
      ii. If yes: extract HTTP method, path, version
      iii. Register in routing.Router
   c. Build handler chain (middleware + handler)
3. Detect conflicts (panic if found)
4. Build trie for fast matching
```

### 7.2 Handler Chain Building

**For route: `READ_BY_ID` in UserController**:

```
1. Create RouterItem
2. Add global middlewares (if any)
3. Add module middlewares (scoped to this route's module)
4. Add route guard (if any)
5. Add route interceptor pre-handler
6. Add actual handler method
7. Add route interceptor post-handler
8. Store in item.Handlers[]
```

**Execution**: Handlers array executed sequentially via `ctx.HTTPHandler` chain

---

## 8. Controller Registration

### 8.1 Controller Discovery

**Requirement**: Embed `common.REST` or `common.WS`

```go
type UserController struct {
    common.REST  // Makes this an HTTP controller
    UserService  // Dependency to inject
}

func (c UserController) NewController() Controller {
    return c
}

// Routes automatically discovered:
func (c UserController) READ_BY_ID() { /* GET /:id */ }
func (c UserController) CREATE() { /* POST / */ }
```

### 8.2 Instance Lifecycle

- Controller instantiated per-request
- NewController() factory method called to get instance
- Fields with types matching providers are injected via reflection
- After request, controller discarded (no pooling)

---

## 9. Routing Edge Cases & Rules

### 9.1 Matching Precedence

```
1. Exact static path match (highest)
2. Prefix dynamic path match
3. Complex regex pattern match
4. Catch-all wildcard match (lowest)
```

### 9.2 Ambiguous Routes

**Problem**: Multiple patterns could match same request

**Solution**: Error at app.Create() → prevent ambiguity

**Example of Ambiguous Routes** (BANNED):
```go
func (c *UserController) READ_BY_ID() { /* GET /:id */ }
func (c *UserController) READ_BY_ANY() { /* GET /:anything */ }
// Both match "GET /123" — CONFLICT!
```

### 9.3 Path Parameter Naming

**Rule**: All `:paramName` tokens must match method name

**Valid**:
```go
// Method name: READ_BY_ID_AND_NAME
// Routes to: GET /:id/:name ✓
func (c *UserController) READ_BY_ID_AND_NAME() { ... }
```

**Invalid**:
```go
// Method name defines params, but handler expects different names
// Routes to: GET /:id (only :id extracted)
// Param map won't have "userId" → error at runtime
func (c *UserController) READ_BY_ID(params ctx.Param) {
    // This fails:
    id := params.Get("userId")  // NOT "id"
}
```

---

## 10. MethodRouteVersionToPattern() Function

**Purpose**: Convert method name → HTTP method + route + version

**Input**: "READ_BY_ID_VERSION_1"

**Output**:
```
httpMethod = "GET"
route = "/:id"
version = "1"
pattern = "GET /:id" (used for routing lookup)
```

**Algorithm**:
1. Extract base token (READ, CREATE, etc.) → HTTP method
2. Extract version token (VERSION_X) if present
3. Parse remaining tokens (BY, AND, OF, ANY) → build route path
4. Combine: METHOD + " " + ROUTE = pattern

