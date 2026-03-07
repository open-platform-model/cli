# CUE SDK Go — API Reference

Complete reference for the CUE Go SDK packages used in this project.
Covers every exported function and method with a description and a working example.

> See also: [cue-in-go.md](./cue-in-go.md) for architecture concepts,
> copy/clone strategies, AST manipulation, and the decision guide.

---

## Table of Contents

1. [Entry Point — `cuelang.org/go/cue/cuecontext`](#1-entry-point--cuelangorggoguecuecontext)
2. [Context — `cuelang.org/go/cue` (`Context`)](#2-context--cuelangorggoguecue-context)
3. [Value — `cuelang.org/go/cue` (`Value`)](#3-value--cuelangorggoguecue-value)
4. [Path & Selector — `cuelang.org/go/cue`](#4-path--selector--cuelangorggoguecue)
5. [Iterator — `cuelang.org/go/cue`](#5-iterator--cuelangorggoguecue)
6. [Attribute — `cuelang.org/go/cue`](#6-attribute--cuelangorggoguecue)
7. [Loading — `cuelang.org/go/cue/load`](#7-loading--cuelangorggoguecueload)
8. [Formatting — `cuelang.org/go/cue/format`](#8-formatting--cuelangorggocueformat)
9. [Errors — `cuelang.org/go/cue/errors`](#9-errors--cuelangorggoguecueerrors)
10. [Go Codec — `cuelang.org/go/encoding/gocode/gocodec`](#10-go-codec--cuelangorggoencoding-gocode-gocodec)
11. [JSON Encoding — `cuelang.org/go/encoding/json`](#11-json-encoding--cuelangorggoencoding-json)
12. [YAML Encoding — `cuelang.org/go/encoding/yaml`](#12-yaml-encoding--cuelangorggoencoding-yaml)
13. [Kind & Op Constants](#13-kind--op-constants)
14. [Selector Types](#14-selector-types)

---

## 1. Entry Point — `cuelang.org/go/cue/cuecontext`

The only purpose of this package is to create a `*cue.Context`. Always start here.
Never use the zero value of `cue.Context` directly.

---

### `New`

```go
func New(options ...Option) *cue.Context
```

Creates the root evaluation context. Reads `CUE_EXPERIMENT` and `CUE_DEBUG`
environment variables by default. A context may only grow — create one per
command invocation, not per operation.

```go
import "cuelang.org/go/cue/cuecontext"

ctx := cuecontext.New()
```

With options:

```go
ctx := cuecontext.New(
    cuecontext.EvaluatorVersion(cuecontext.EvalV3),
)
```

---

### `EvaluatorVersion`

```go
func EvaluatorVersion(v EvalVersion) Option
```

Pins which evaluator version to use. Useful to opt in to the latest stable
evaluator explicitly rather than relying on environment variables.

| Constant | Description |
|---|---|
| `EvalDefault` | Based on `CUE_EXPERIMENT` env var |
| `EvalStable` | Latest stable (currently V3) |
| `EvalExperiment` | In-development; may change without notice |
| `EvalV3` | Recommended: new disjunction + closedness algorithm |

```go
ctx := cuecontext.New(cuecontext.EvaluatorVersion(cuecontext.EvalV3))
```

---

### `CUE_DEBUG`

```go
func CUE_DEBUG(s string) Option
```

Pass debug flags programmatically instead of via the `CUE_DEBUG` env var.
Panics for unknown or malformed options. Primarily for testing.

```go
ctx := cuecontext.New(cuecontext.CUE_DEBUG("verboseerrors=1"))
```

---

## 2. Context — `cuelang.org/go/cue` (`Context`)

`*cue.Context` is the factory for all `Value`s. Values from different contexts
are incompatible — mixing them in `Unify` or `FillPath` panics.

---

### `CompileString`

```go
func (c *Context) CompileString(src string, opts ...BuildOption) Value
```

Parses and evaluates a CUE source string. No import resolution.
Best for inline CUE, tests, and single-file fragments.

Always pass `cue.Filename` so that error messages include a file name.

```go
v := ctx.CompileString(`
    #Config: {
        image:    string
        replicas: int & >0
    }
`, cue.Filename("schema.cue"))
if v.Err() != nil {
    log.Fatal(v.Err())
}
```

---

### `CompileBytes`

```go
func (c *Context) CompileBytes(b []byte, opts ...BuildOption) Value
```

Same as `CompileString` but takes a `[]byte`. Preferred when reading from disk.

```go
src, _ := os.ReadFile("values.cue")
v := ctx.CompileBytes(src, cue.Filename("values.cue"))
```

---

### `BuildInstance`

```go
func (c *Context) BuildInstance(i *build.Instance, opts ...BuildOption) Value
```

Builds a `Value` from a `*build.Instance` returned by `load.Instances`.
This is the entry point for full module loading with import resolution.

```go
instances := load.Instances([]string{"."}, &load.Config{Dir: moduleDir})
v := ctx.BuildInstance(instances[0])
if v.Err() != nil {
    log.Fatal(v.Err())
}
```

---

### `BuildInstances`

```go
func (c *Context) BuildInstances(instances []*build.Instance) ([]Value, error)
```

Builds multiple instances at once and returns all errors combined.

```go
instances := load.Instances([]string{"./..."}, cfg)
values, err := ctx.BuildInstances(instances)
if err != nil {
    log.Fatal(err)
}
```

---

### `BuildFile`

```go
func (c *Context) BuildFile(f *ast.File, opts ...BuildOption) Value
```

Builds a `Value` from a parsed `*ast.File`. Does not resolve external imports.
Used after AST manipulation when the CUE is self-contained.

```go
f, _ := parser.ParseFile("modified.cue", src, parser.ParseComments)
v := ctx.BuildFile(f)
```

---

### `BuildExpr`

```go
func (c *Context) BuildExpr(x ast.Expr, opts ...BuildOption) Value
```

Builds a `Value` from an `ast.Expr` (a single expression rather than a full file).

```go
expr, _ := parser.ParseExpr("inline", `{host: "localhost", port: 8080}`)
v := ctx.BuildExpr(expr)
```

---

### `Encode`

```go
func (c *Context) Encode(x any, opts ...EncodeOption) Value
```

Converts a Go value to a CUE `Value`. Traverses structs, maps, slices, and
scalars recursively. Respects `json:` struct tags for field naming and
`omitempty`. Handles `json.Marshaler` and `encoding.TextMarshaler`.

```go
type Config struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

v := ctx.Encode(Config{Host: "localhost", Port: 8080})
// CUE: {host: "localhost", port: 8080}

// Inject at a specific path
base := ctx.CompileString(`#release: {}`)
filled := base.FillPath(cue.ParsePath("#release.config"), v)
```

With `NilIsAny` — treat Go `nil` as CUE `_` (top/any) instead of `null`:

```go
v := ctx.Encode(nil, cue.NilIsAny(true))
// CUE: _ (top)
```

---

### `EncodeType`

```go
func (c *Context) EncodeType(x any, opts ...EncodeOption) Value
```

Converts a Go *type* to a CUE schema `Value`. Ignores concrete field values —
produces the type constraints, not the data.

```go
type Module struct {
    Name    string `json:"name"`
    Version string `json:"version"`
    Tags    []string `json:"tags,omitempty"`
}

schema := ctx.EncodeType(Module{})
// CUE: {name: string, version: string, tags?: [...string]}
```

---

### `NewList`

```go
func (c *Context) NewList(v ...Value) Value
```

Creates a CUE list value from a set of existing `Value`s. All values must
originate from the same context.

```go
a := ctx.CompileString(`"foo"`)
b := ctx.CompileString(`"bar"`)
list := ctx.NewList(a, b)
// CUE: ["foo", "bar"]
```

---

### `BuildOption` constructors

These are passed to `CompileString`, `CompileBytes`, `BuildInstance`, etc.

#### `Filename`

```go
func Filename(filename string) BuildOption
```

Attaches a filename to the compiled source. Used for positions in error messages.
Always provide this when compiling bytes/strings.

```go
v := ctx.CompileBytes(src, cue.Filename("config.cue"))
```

#### `Scope`

```go
func Scope(scope Value) BuildOption
```

Resolves unresolved identifiers against the given value. Useful when compiling
CUE expressions that reference definitions from another loaded value.

```go
schema := ctx.CompileString(`#Port: int & >0 & <=65535`)
expr := ctx.CompileString(`#Port`, cue.Scope(schema))
```

#### `ImportPath`

```go
func ImportPath(path string) BuildOption
```

Sets the import path for the compiled value. Used when the compiled source
needs to be referenced as a package.

```go
v := ctx.CompileString(src, cue.ImportPath("example.com/mymod/config"))
```

#### `InferBuiltins`

```go
func InferBuiltins(elide bool) BuildOption
```

Allows unresolved references to bind to CUE builtin packages by unique name.
Useful when evaluating expressions without explicit import statements.

```go
v := ctx.CompileString(`strings.ToUpper("hello")`, cue.InferBuiltins(true))
```

---

## 3. Value — `cuelang.org/go/cue` (`Value`)

`cue.Value` is the core type. It is a small struct (not a pointer) — copying is
free and safe. All operations are pure and return new values. Nil/zero `Value`
is valid but represents a missing value.

---

### Type Inspection

#### `Kind`

```go
func (v Value) Kind() Kind
```

Returns the concrete kind. Returns `BottomKind` for non-concrete values
(e.g. bounds like `>=0`) and error values.

```go
v := ctx.CompileString(`42`)
fmt.Println(v.Kind() == cue.IntKind) // true

v2 := ctx.CompileString(`>=0`)
fmt.Println(v2.Kind()) // BottomKind — not concrete
```

#### `IncompleteKind`

```go
func (v Value) IncompleteKind() Kind
```

Returns a bitmask of all kinds the value *could* be. Works for non-concrete
values where `Kind` would return `BottomKind`.

```go
v := ctx.CompileString(`>=0`)
fmt.Println(v.IncompleteKind() == cue.IntKind) // true

// Disjunction: string | int
v2 := ctx.CompileString(`string | int`)
fmt.Println(v2.IncompleteKind() == (cue.StringKind | cue.IntKind)) // true
```

#### `IsConcrete`

```go
func (v Value) IsConcrete() bool
```

Reports whether the value is fully concrete — no unresolved references,
no constraints, no bounds. Does not check list/struct element values recursively.

```go
ctx.CompileString(`42`).IsConcrete()         // true
ctx.CompileString(`int`).IsConcrete()        // false
ctx.CompileString(`>=0`).IsConcrete()        // false
ctx.CompileString(`{a: 1}`).IsConcrete()     // true (top-level only)
```

#### `Exists`

```go
func (v Value) Exists() bool
```

Reports whether the value exists in the configuration. Returns false when
`LookupPath` or a field accessor found no matching field.

```go
v := ctx.CompileString(`{a: 1}`)
fmt.Println(v.LookupPath(cue.ParsePath("a")).Exists()) // true
fmt.Println(v.LookupPath(cue.ParsePath("b")).Exists()) // false
```

#### `IsClosed`

```go
func (v Value) IsClosed() bool
```

Reports if the struct/value is closed at the top level — no additional fields
are allowed. Structs referenced as `#Definitions` are closed.

```go
open := ctx.CompileString(`{a: 1}`)
closed := ctx.CompileString(`close({a: 1})`)
fmt.Println(open.IsClosed())   // false
fmt.Println(closed.IsClosed()) // true
```

#### `IsNull`

```go
func (v Value) IsNull() bool
```

Reports whether the value is exactly `null`.

```go
v := ctx.CompileString(`null`)
fmt.Println(v.IsNull()) // true
```

#### `Err`

```go
func (v Value) Err() error
```

Returns the error if the value represents an error, nil otherwise.
Always check this after compilation, build, or unification.

```go
v := ctx.CompileString(`{a: 1}`)
if v.Err() != nil {
    log.Fatalf("compile error: %v", v.Err())
}

// After unification that results in a conflict
bad := ctx.CompileString(`string`).Unify(ctx.CompileString(`42`))
fmt.Println(bad.Err()) // error: conflicting values
```

---

### Scalar Extraction

#### `Bool`

```go
func (v Value) Bool() (bool, error)
```

Extracts the boolean value. Returns an error if v is not a concrete boolean.

```go
v := ctx.CompileString(`true`)
b, err := v.Bool()
// b == true, err == nil
```

#### `String`

```go
func (v Value) String() (string, error)
```

Extracts the string value. Returns an error if v is not a concrete string.

```go
v := ctx.CompileString(`"hello"`)
s, err := v.String()
// s == "hello", err == nil
```

#### `Bytes`

```go
func (v Value) Bytes() ([]byte, error)
```

Extracts bytes. Accepts both CUE `bytes` and `string` kinds.

```go
v := ctx.CompileString(`'hello'`) // CUE bytes literal
b, err := v.Bytes()
// b == []byte("hello")
```

#### `Int64`

```go
func (v Value) Int64() (int64, error)
```

Extracts an integer as `int64`. Returns `(math.MaxInt64, ErrBelow)` if value
exceeds int64 range, `(math.MinInt64, ErrAbove)` if below.

```go
v := ctx.CompileString(`42`)
n, err := v.Int64()
// n == 42, err == nil
```

#### `Uint64`

```go
func (v Value) Uint64() (uint64, error)
```

Extracts an integer as `uint64`. Returns `(0, ErrAbove)` for negative values.

```go
v := ctx.CompileString(`255`)
n, err := v.Uint64()
// n == 255, err == nil
```

#### `Float64`

```go
func (v Value) Float64() (float64, error)
```

Extracts a float as `float64`. Returns `ErrAbove`/`ErrBelow` on overflow.

```go
v := ctx.CompileString(`3.14`)
f, err := v.Float64()
// f == 3.14, err == nil
```

#### `Int`

```go
func (v Value) Int(z *big.Int) (*big.Int, error)
```

Extracts an arbitrary-precision integer. Pass `nil` to allocate a new `big.Int`,
or pass an existing one to reuse it.

```go
v := ctx.CompileString(`99999999999999999999`)
n, err := v.Int(nil)
```

#### `Float`

```go
func (v Value) Float(f *big.Float) (*big.Float, error)
```

Extracts an arbitrary-precision float.

```go
v := ctx.CompileString(`1.23456789012345678901234567890`)
f, err := v.Float(nil)
```

#### `Null`

```go
func (v Value) Null() error
```

Returns nil if v is `null`, an error otherwise. Used as a type assertion.

```go
v := ctx.CompileString(`null`)
if err := v.Null(); err == nil {
    fmt.Println("value is null")
}
```

#### `AppendInt`

```go
func (v Value) AppendInt(buf []byte, base int) ([]byte, error)
```

Appends the string representation of the integer to buf in the given base.
Avoids allocation when building strings with an existing buffer.

```go
v := ctx.CompileString(`255`)
buf, _ := v.AppendInt(nil, 16)
// string(buf) == "ff"
```

#### `AppendFloat`

```go
func (v Value) AppendFloat(buf []byte, fmt byte, prec int) ([]byte, error)
```

Appends the float string representation to buf using Go's `strconv` format
conventions (`'f'`, `'e'`, `'g'`, etc.).

```go
v := ctx.CompileString(`3.14`)
buf, _ := v.AppendFloat(nil, 'f', 2)
// string(buf) == "3.14"
```

#### `MantExp`

```go
func (v Value) MantExp(mant *big.Int) (exp int, err error)
```

Decomposes a CUE number into mantissa × 10^exp. Pass `nil` for mant to get
just the exponent without allocation.

```go
v := ctx.CompileString(`1234`)
exp, _ := v.MantExp(nil)
// exp == 0 (1234 × 10^0)
```

---

### Traversal

#### `LookupPath`

```go
func (v Value) LookupPath(p Path) Value
```

Returns the sub-value at path p. Returns a non-existing value (check with
`Exists()`) if the path does not exist.

Use `cue.AnyString` to get the element type of an open struct `[string]: T`,
and `cue.AnyIndex` to get the element type of an open list `[...T]`.

```go
v := ctx.CompileString(`{
    metadata: {
        name:    "my-app"
        version: "1.0.0"
    }
}`)

name, _ := v.LookupPath(cue.ParsePath("metadata.name")).String()
// name == "my-app"

// Nested definition
v2 := ctx.CompileString(`#Release: spec: module: string`)
moduleType := v2.LookupPath(cue.ParsePath("#Release.spec.module"))
fmt.Println(moduleType.IncompleteKind()) // StringKind

// Open struct element type
v3 := ctx.CompileString(`labels: [string]: string`)
elemType := v3.LookupPath(cue.MakePath(cue.Str("labels"), cue.AnyString))
fmt.Println(elemType.IncompleteKind()) // StringKind
```

#### `FillPath`

```go
func (v Value) FillPath(p Path, x interface{}) Value
```

Returns a new value with x unified into v at path p. Does not mutate v.
x can be a Go value (encoded via `Context.Encode`), an `ast.Expr`, or
a `cue.Value` from the same context. The resulting value is not validated —
call `Err()` on the result.

```go
base := ctx.CompileString(`#Release: {module: string, replicas: int}`)

// Inject a Go string
v := base.FillPath(cue.ParsePath("#Release.module"), "my-app")

// Inject an encoded struct
type Meta struct{ Version string `json:"version"` }
meta := ctx.Encode(Meta{Version: "1.2.3"})
v2 := base.FillPath(cue.ParsePath("#Release.meta"), meta)

// Check for conflicts
if v.Err() != nil {
    log.Fatalf("fill error: %v", v.Err())
}
```

#### `Fields`

```go
func (v Value) Fields(opts ...Option) (*Iterator, error)
```

Returns an iterator over the struct fields of v. By default omits definitions,
hidden fields, and optional fields. Returns an error if v is not a struct.

```go
v := ctx.CompileString(`{a: 1, b: "hello", c?: true}`)

// Default: regular fields only
iter, _ := v.Fields()
for iter.Next() {
    fmt.Printf("%s: %v\n", iter.Selector(), iter.Value())
}
// a: 1
// b: "hello"

// Include optional and definitions
iter, _ = v.Fields(cue.Optional(true), cue.Definitions(true))

// Include hidden fields too
iter, _ = v.Fields(cue.All())
```

#### `List`

```go
func (v Value) List() (Iterator, error)
```

Returns an iterator over the list elements. Returns an error if v is not a list.

```go
v := ctx.CompileString(`["a", "b", "c"]`)
iter, _ := v.List()
for iter.Next() {
    s, _ := iter.Value().String()
    fmt.Println(s)
}
// a
// b
// c
```

#### `Len`

```go
func (v Value) Len() Value
```

Returns the length as a CUE value. For lists: capacity. For structs: field
count. For bytes/strings: byte count. Extract with `Int64()`.

```go
v := ctx.CompileString(`[1, 2, 3]`)
n, _ := v.Len().Int64()
// n == 3
```

#### `Walk`

```go
func (v Value) Walk(before func(Value) bool, after func(Value))
```

Performs a depth-first traversal of the data model values. The `before`
function is called before descending; return false to skip children. `after`
is called after. Does not visit definitions, hidden, optional, or required
fields — only regular data fields.

```go
v := ctx.CompileString(`{a: {b: {c: 42}}}`)

v.Walk(func(v cue.Value) bool {
    fmt.Printf("enter: %v (kind=%v)\n", v.Path(), v.Kind())
    return true
}, func(v cue.Value) {
    fmt.Printf("leave: %v\n", v.Path())
})
```

#### `Path`

```go
func (v Value) Path() Path
```

Returns the path from the root to this value within its original instance.
Only defined for values with a fixed path; returns empty path for computed values.

```go
v := ctx.CompileString(`{a: {b: 1}}`)
sub := v.LookupPath(cue.ParsePath("a.b"))
fmt.Println(sub.Path()) // a.b
```

---

### Unification & Validation

#### `Unify`

```go
func (v Value) Unify(w Value) Value
```

Returns the greatest lower bound (CUE `&` operator) of v and w. Both must
come from the same context. This is the primary way to merge constraints
with data.

```go
schema := ctx.CompileString(`{port: >0 & <=65535}`)
data   := ctx.CompileString(`{port: 8080}`)

merged := schema.Unify(data)
if err := merged.Validate(cue.Concrete(true)); err != nil {
    log.Fatal(err) // nil — 8080 satisfies >0 & <=65535
}

// Conflict
bad := ctx.CompileString(`{port: 99999}`).Unify(schema)
fmt.Println(bad.Err()) // error: port out of range
```

#### `UnifyAccept`

```go
func (v Value) UnifyAccept(w Value, accept Value) Value
```

Like `Unify(w)` but disregards closedness rules — only allows fields present
in `accept`. Used for incremental unification of conjuncts while preserving
structural constraints.

```go
schema := ctx.CompileString(`close({a: int, b: string})`)
part1  := ctx.CompileString(`{a: 1}`)
part2  := ctx.CompileString(`{b: "hello"}`)

v := schema.UnifyAccept(part1, schema)
v  = v.UnifyAccept(part2, schema)
```

#### `Validate`

```go
func (v Value) Validate(opts ...Option) error
```

Validates the value recursively and returns aggregated errors. By default
accepts non-concrete values. Use `Concrete(true)` to require all values
to be fully resolved.

```go
v := ctx.CompileString(`{
    name:     "my-app"
    replicas: 3
    port:     int & >0
}`)

// Schema-level validation (no concrete requirement)
if err := v.Validate(); err != nil {
    log.Fatalf("invalid schema: %v", err)
}

// Concrete validation — fails because port is not concrete
if err := v.Validate(cue.Concrete(true)); err != nil {
    fmt.Println("not fully concrete:", err)
}

// Final — closes structs and selects defaults
if err := v.Validate(cue.Final()); err != nil {
    fmt.Println("missing required fields:", err)
}
```

#### `Subsume`

```go
func (v Value) Subsume(w Value, opts ...Option) error
```

Returns nil if w is an instance of v (v subsumes w). In other words: v is
"at least as general" as w. Useful for backwards-compatibility checks.

```go
v1 := ctx.CompileString(`{a: int}`)          // v1 is more general
v2 := ctx.CompileString(`{a: int, b: string}`) // v2 has extra field

err := v1.Subsume(v2) // nil: v1 subsumes v2 (v2 is a valid instance of v1)
err  = v2.Subsume(v1) // error: v1 doesn't have field b
```

#### `Equals`

```go
func (v Value) Equals(other Value) bool
```

Reports whether two values are structurally equal, ignoring optional fields.
Result is undefined for incomplete/non-concrete values.

```go
a := ctx.CompileString(`{x: 1, y: 2}`)
b := ctx.CompileString(`{x: 1, y: 2}`)
fmt.Println(a.Equals(b)) // true
```

#### `Allows`

```go
func (v Value) Allows(sel Selector) bool
```

Reports if a field with the given selector is permitted by the value's
constraints. Useful for checking whether adding a field would violate closedness.

```go
open   := ctx.CompileString(`{a: 1}`)
closed := ctx.CompileString(`close({a: 1})`)

fmt.Println(open.Allows(cue.Str("b")))   // true
fmt.Println(closed.Allows(cue.Str("b"))) // false
fmt.Println(closed.Allows(cue.Str("a"))) // true
```

#### `Default`

```go
func (v Value) Default() (Value, bool)
```

Returns the default value and whether a default exists. If no default,
returns the value itself.

```go
v := ctx.CompileString(`*3 | int`) // default 3
def, ok := v.Default()
// ok == true, def == 3
```

#### `Eval`

```go
func (v Value) Eval() Value
```

Resolves references within v. In most cases this is a no-op since the SDK
evaluates lazily. Use only when you need to force resolution before inspection.

```go
v := ctx.CompileString(`x: 1, y: x + 1`)
sub := v.LookupPath(cue.ParsePath("y")).Eval()
n, _ := sub.Int64()
// n == 2
```

---

### Encoding & Serialization

#### `Decode`

```go
func (v Value) Decode(x interface{}) error
```

Populates a Go value pointed to by x with the contents of v. x must be a
non-nil pointer. Checks interfaces in priority order:
1. `cue.Unmarshaler` (custom CUE decoding)
2. `json.Unmarshaler`
3. `encoding.TextUnmarshaler`

For structs, validates constraints from `cue:` field tags.
A field of type `cue.Value` receives the raw CUE value without conversion.

```go
type Config struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

v := ctx.CompileString(`{host: "localhost", port: 8080}`)

var cfg Config
if err := v.Decode(&cfg); err != nil {
    log.Fatal(err)
}
// cfg.Host == "localhost", cfg.Port == 8080

// Decode into a map
var m map[string]interface{}
v.Decode(&m)

// Decode into cue.Value (keeps it as CUE)
var raw cue.Value
v.LookupPath(cue.ParsePath("host")).Decode(&raw)
```

#### `MarshalJSON`

```go
func (v Value) MarshalJSON() (b []byte, err error)
```

Marshals the CUE value to JSON bytes. The value must be concrete.

```go
v := ctx.CompileString(`{name: "my-app", replicas: 3}`)
b, err := v.MarshalJSON()
// b == []byte(`{"name":"my-app","replicas":3}`)
```

#### `Syntax`

```go
func (v Value) Syntax(opts ...Option) ast.Node
```

Converts the value to an AST node for inspection or formatting.
The returned node can be formatted with `format.Node`.

```go
v := ctx.CompileString(`{
    #Config: {host: string}
    config: #Config & {host: "localhost"}
}`)

// Full value with definitions
node := v.Syntax()

// Only concrete data
node = v.Syntax(cue.Final(), cue.Concrete(true))

// Everything including hidden fields and docs
node = v.Syntax(cue.All(), cue.Docs(true))

// Format to bytes
b, _ := format.Node(node)
fmt.Println(string(b))
```

---

### Metadata & References

#### `Attribute`

```go
func (v Value) Attribute(key string) Attribute
```

Returns the attribute with the given key. If no such attribute exists,
all methods on the returned `Attribute` return errors.

```go
v := ctx.CompileString(`
    config: {
        port: 8080 @env(PORT,type=int)
    }
`)
port := v.LookupPath(cue.ParsePath("config.port"))
attr := port.Attribute("env")
key, val := attr.Arg(0) // key == "PORT", val == ""
key2, val2 := attr.Arg(1) // key2 == "type", val2 == "int"
```

#### `Attributes`

```go
func (v Value) Attributes(mask AttrKind) []Attribute
```

Returns all attributes matching the kind mask. Use `cue.FieldAttr` for
field-level attributes, `cue.DeclAttr` for declaration-level.

```go
v := ctx.CompileString(`port: 8080 @json("port") @env("PORT")`)
port := v.LookupPath(cue.ParsePath("port"))
attrs := port.Attributes(cue.FieldAttr)
for _, a := range attrs {
    fmt.Println(a.Name()) // "json", "env"
}
```

#### `Doc`

```go
func (v Value) Doc() []*ast.CommentGroup
```

Returns documentation comments for the field from which this value originates.

```go
v := ctx.CompileString(`
    // The HTTP port to listen on.
    port: 8080
`, cue.Filename("config.cue"))

port := v.LookupPath(cue.ParsePath("port"))
for _, cg := range port.Doc() {
    for _, c := range cg.List {
        fmt.Println(c.Text) // "// The HTTP port to listen on."
    }
}
```

#### `Pos`

```go
func (v Value) Pos() token.Pos
```

Returns the source position of the value for use in error messages.

```go
v := ctx.CompileString(`{a: 1}`, cue.Filename("test.cue"))
fmt.Println(v.LookupPath(cue.ParsePath("a")).Pos())
// test.cue:1:5
```

#### `Expr`

```go
func (v Value) Expr() (Op, []Value)
```

Decomposes the expression tree: returns the operator and its operands.
Used for inspecting the structure of constraints without evaluating them.

```go
v := ctx.CompileString(`int & >0 & <=100`)
op, args := v.Expr()
// op == cue.AndOp
// args[0] == int, args[1] == >0, args[2] == <=100

// Disjunction
v2 := ctx.CompileString(`"a" | "b" | "c"`)
op2, args2 := v2.Expr()
// op2 == cue.OrOp, len(args2) == 3
```

#### `Source`

```go
func (v Value) Source() ast.Node
```

Returns the original AST node for this value. Returns nil for computed or
programmatically constructed values. Useful for getting precise source positions.

```go
v := ctx.CompileString(`port: 8080`, cue.Filename("cfg.cue"))
port := v.LookupPath(cue.ParsePath("port"))
node := port.Source()
if node != nil {
    fmt.Println(node.Pos()) // cfg.cue:1:7
}
```

#### `ReferencePath`

```go
func (v Value) ReferencePath() (root Value, p Path)
```

If v is a reference (e.g. a field that refers to another field), returns the
root value and path of the target. Returns zero values if v is not a reference.

```go
v := ctx.CompileString(`{a: 1, b: a}`)
b := v.LookupPath(cue.ParsePath("b"))
root, path := b.ReferencePath()
// path == a
// root.LookupPath(path) == 1
```

#### `Context`

```go
func (v Value) Context() *Context
```

Returns the `*Context` that created this value. Use this to avoid passing
the context as a parameter — just retrieve it from any value you already have.

```go
func process(v cue.Value) {
    ctx := v.Context() // no need to pass ctx separately
    extra := ctx.CompileString(`extra: "field"`)
    result := v.Unify(extra)
    _ = result
}
```

---

### `Option` constructors (for `Validate`, `Fields`, `Syntax`, `Subsume`)

| Function | Description |
|---|---|
| `cue.Concrete(bool)` | Require all values to be concrete |
| `cue.Final()` | Close structs/lists, select defaults |
| `cue.Schema()` | Treat as schema — ignore closedness (for `Subsume`) |
| `cue.Optional(bool)` | Include optional fields in `Fields()` |
| `cue.Hidden(bool)` | Include hidden fields in `Fields()` |
| `cue.Definitions(bool)` | Include `#definitions` in `Fields()` |
| `cue.Patterns(bool)` | Include pattern constraints (`[string]: T`) in `Fields()` |
| `cue.All()` | Include all fields and values |
| `cue.Docs(bool)` | Include doc comments in `Syntax()` |
| `cue.Attributes(bool)` | Include attributes in `Syntax()` |
| `cue.InlineImports(bool)` | Inline imported references in `Syntax()` |
| `cue.DisallowCycles(bool)` | Error on cycles even without `Concrete` |
| `cue.ErrorsAsValues(bool)` | Render errors inline in `Syntax()` output |
| `cue.Raw()` | Generate raw unsimplified AST in `Syntax()` |

---

### Utility functions

#### `Dereference`

```go
func Dereference(v Value) Value
```

If v is a reference, returns the target. Otherwise returns v unchanged.

```go
v := ctx.CompileString(`{a: 1, b: a}`)
b := v.LookupPath(cue.ParsePath("b"))
target := cue.Dereference(b)
n, _ := target.Int64()
// n == 1
```

#### `IsIncomplete`

```go
func IsIncomplete(err error) bool
```

Reports whether an error is an "incomplete" error — the value has unresolved
references or unfilled constraints but is not structurally invalid. These
errors are acceptable in non-concrete (schema) contexts.

```go
v := ctx.CompileString(`{port: >0}`)
err := v.Validate(cue.Concrete(true))
if cue.IsIncomplete(err) {
    fmt.Println("schema valid but not yet concrete")
}
```

#### `LanguageVersion`

```go
func LanguageVersion() string
```

Returns the CUE language version string supported by the current SDK.

```go
fmt.Println(cue.LanguageVersion()) // e.g. "v0.12.0"
```

---

### `Unmarshaler` interface

```go
type Unmarshaler interface {
    UnmarshalCUE(v Value) error
}
```

Implement this interface for custom CUE → Go decoding.
Checked before `json.Unmarshaler` and `encoding.TextUnmarshaler` by `Value.Decode`.

```go
type Duration struct {
    d time.Duration
}

func (dur *Duration) UnmarshalCUE(v cue.Value) error {
    s, err := v.String()
    if err != nil {
        return err
    }
    dur.d, err = time.ParseDuration(s)
    return err
}

v := ctx.CompileString(`"1h30m"`)
var dur Duration
v.Decode(&dur)
// dur.d == 90 * time.Minute
```

---

## 4. Path & Selector — `cuelang.org/go/cue`

A `Path` is a sequence of `Selector`s that addresses a location within a CUE value.

---

### `MakePath`

```go
func MakePath(selectors ...Selector) Path
```

Constructs a path from explicit selectors. More type-safe than `ParsePath`
because each selector's type is set explicitly.

```go
// metadata.name
p := cue.MakePath(cue.Str("metadata"), cue.Str("name"))

// #Release.spec.components[0]
p2 := cue.MakePath(cue.Def("Release"), cue.Str("spec"),
    cue.Str("components"), cue.Index(0))

v.LookupPath(p)
```

---

### `ParsePath`

```go
func ParsePath(s string) Path
```

Parses a CUE path string. Check `.Err()` for parse errors.
Cannot represent hidden fields — use `MakePath(Hid(...))` for those.

```go
p := cue.ParsePath("metadata.name")
p2 := cue.ParsePath("#Release.spec.module")
p3 := cue.ParsePath("items[0].value")

if err := p.Err(); err != nil {
    log.Fatal(err)
}
```

---

### `Path` methods

| Method | Description |
|---|---|
| `Selectors() []Selector` | Returns the path components |
| `Append(sel ...Selector) Path` | Adds selectors; returns new path |
| `Optional() Path` | Converts all selectors to optional form (`foo?`) |
| `String() string` | CUE string representation |
| `Err() error` | Error from `ParsePath`, if any |

```go
p := cue.ParsePath("spec.module")
p2 := p.Append(cue.Str("version"))
fmt.Println(p2.String()) // spec.module.version

fmt.Println(p.Optional().String()) // spec?.module?
```

---

### Selector constructors

#### `Str`

```go
func Str(s string) Selector
```

Creates a regular string field selector (non-definition).

```go
cue.Str("metadata")   // metadata
cue.Str("my-field")   // "my-field" (quoted if not a valid identifier)
```

#### `Def`

```go
func Def(s string) Selector
```

Creates a definition selector. Adds `#` prefix if not already present.
Panics if s cannot be a valid identifier.

```go
cue.Def("Release")  // #Release
cue.Def("#Config")  // #Config (# already present)
```

#### `Hid`

```go
func Hid(name, pkg string) Selector
```

Creates a hidden field selector scoped to a package. Use `"_"` for anonymous
packages. Required: `pkg` must not be empty.

```go
cue.Hid("_internal", "github.com/myorg/mymod")
cue.Hid("_secret", "_") // anonymous package
```

#### `Index`

```go
func Index[T interface{ int | int64 }](x T) Selector
```

Creates a list index selector.

```go
cue.Index(0)  // [0]
cue.Index(2)  // [2]
```

#### `Label`

```go
func Label(label ast.Label) Selector
```

Converts an AST label node to a Selector. Useful when working with AST.

```go
ident := ast.NewIdent("host")
sel := cue.Label(ident)
```

#### `AnyIndex`, `AnyString`

```go
var AnyIndex = Selector{...}  // [_] — element type of [...T]
var AnyString = Selector{...} // any string field — element type of [string]: T
```

```go
// Get element type of open list
v := ctx.CompileString(`items: [...string]`)
elemType := v.LookupPath(cue.MakePath(cue.Str("items"), cue.AnyIndex))
fmt.Println(elemType.IncompleteKind()) // StringKind

// Get value type of open struct
v2 := ctx.CompileString(`labels: [string]: string`)
valType := v2.LookupPath(cue.MakePath(cue.Str("labels"), cue.AnyString))
fmt.Println(valType.IncompleteKind()) // StringKind
```

---

### `Selector` methods

| Method | Description |
|---|---|
| `String() string` | CUE representation of the selector |
| `Type() SelectorType` | Full type (label + constraint) |
| `LabelType() SelectorType` | Just the label kind |
| `ConstraintType() SelectorType` | Just the constraint kind |
| `IsConstraint() bool` | True for optional and pattern constraint selectors |
| `IsString() bool` | True for regular/optional/required member fields |
| `IsDefinition() bool` | True for non-hidden definition selectors |
| `Optional() Selector` | Convert to optional form (`foo?`) |
| `Required() Selector` | Convert to required form (`foo!`) |
| `Unquoted() string` | Unquoted name (only for `StringLabel`) |
| `PkgPath() string` | Package path for hidden labels |
| `Index() int` | Index value (panics unless `IndexLabel`) |
| `Pattern() Value` | Pattern value (only for pattern constraint selectors) |

```go
iter, _ := v.Fields(cue.Definitions(true), cue.Optional(true))
for iter.Next() {
    sel := iter.Selector()
    fmt.Printf("name=%s isDef=%v isOpt=%v\n",
        sel.String(), sel.IsDefinition(), sel.IsConstraint())
}
```

---

## 5. Iterator — `cuelang.org/go/cue`

Returned by `Value.Fields()` and `Value.List()`. Always call `Next()` before
the first access.

| Method | Description |
|---|---|
| `Next() bool` | Advance to the next element. Must be called first. |
| `Value() Value` | Current element's value |
| `Selector() Selector` | Current element's label/selector |
| `IsOptional() bool` | Whether the current field is optional |
| `FieldType() SelectorType` | The selector type of the current field |

```go
v := ctx.CompileString(`{
    name:     "my-app"
    version?: "1.0.0"
    #Meta: {}
}`)

// Struct iteration
iter, _ := v.Fields(cue.Optional(true), cue.Definitions(true))
for iter.Next() {
    sel  := iter.Selector()
    val  := iter.Value()
    kind := sel.Type()
    fmt.Printf("%-10s  opt=%-5v  def=%-5v  val=%v\n",
        sel.String(),
        iter.IsOptional(),
        sel.IsDefinition(),
        val,
    )
}
// name        opt=false  def=false  val="my-app"
// version     opt=true   def=false  val="1.0.0"
// #Meta       opt=false  def=true   val={}

// List iteration
list := ctx.CompileString(`[10, 20, 30]`)
li, _ := list.List()
for li.Next() {
    n, _ := li.Value().Int64()
    fmt.Println(n)
}
```

---

## 6. Attribute — `cuelang.org/go/cue`

CUE attributes are `@key(arg1, key=value, ...)` annotations on fields.
Retrieve an `Attribute` via `Value.Attribute(key)` or `Value.Attributes(mask)`.

| Method | Description |
|---|---|
| `Name() string` | Attribute key (e.g. `"json"` for `@json(...)`) |
| `Contents() string` | Full contents inside the parentheses |
| `NumArgs() int` | Number of comma-separated arguments |
| `Arg(i int) (key, val string)` | ith argument; splits on `=` if present |
| `RawArg(i int) string` | Raw ith argument including whitespace |
| `Kind() AttrKind` | Where the attribute appears |
| `Err() error` | Error if attribute is invalid or not found |
| `String(pos int) (string, error)` | String value at position pos |
| `Int(pos int) (int64, error)` | Integer value at position pos |
| `Flag(pos int, key string) (bool, error)` | Whether key entry exists at/after pos |
| `Lookup(pos int, key string) (val, found, err)` | Find `key=value` entry |

```go
v := ctx.CompileString(`
    port: 8080 @tag(PORT,type=int,var=env)
`)
port := v.LookupPath(cue.ParsePath("port"))
attr := port.Attribute("tag")

fmt.Println(attr.Name())       // "tag"
fmt.Println(attr.NumArgs())    // 3

key0, val0 := attr.Arg(0)     // "PORT", ""
key1, val1 := attr.Arg(1)     // "type", "int"
key2, val2 := attr.Arg(2)     // "var", "env"

typeVal, found, _ := attr.Lookup(0, "type")
// typeVal == "int", found == true

hasVar, _ := attr.Flag(0, "var")
// hasVar == true
```

---

## 7. Loading — `cuelang.org/go/cue/load`

Loads CUE modules from disk into `*build.Instance`s. Handles `cue.mod/`,
module resolution, and external registry dependencies.

---

### `Instances`

```go
func Instances(args []string, c *Config) []*build.Instance
```

The primary entry point for loading CUE modules. Pass `[]string{"."}` for
the current directory. Errors loading an instance are on `instance.Err`.

```go
import (
    "cuelang.org/go/cue/load"
    "cuelang.org/go/cue/cuecontext"
)

ctx := cuecontext.New()

cfg := &load.Config{Dir: "/path/to/module"}
instances := load.Instances([]string{"."}, cfg)

if instances[0].Err != nil {
    log.Fatalf("load: %v", instances[0].Err)
}

v := ctx.BuildInstance(instances[0])
if v.Err() != nil {
    log.Fatalf("build: %v", v.Err())
}
```

With overlay — inject generated CUE without writing to disk:

```go
generated := []byte(`_generated: {image: "nginx:1.25"}`)
cfg := &load.Config{
    Dir: modulePath,
    Overlay: map[string]load.Source{
        filepath.Join(modulePath, "_gen.cue"): load.FromBytes(generated),
    },
}
instances := load.Instances([]string{"."}, cfg)
```

---

### `load.Config` key fields

```go
cfg := &load.Config{
    // Working directory for relative path resolution
    Dir: "/path/to/module",

    // Load a specific package (default: all)
    Package: "mypkg",

    // Inject @tag values: "key=value" or boolean "key"
    Tags: []string{"env=prod", "debug"},

    // Dynamic injection variables for @tag(key,var=name)
    TagVars: load.DefaultTagVars(),

    // Virtual filesystem overlay (absolute path → Source)
    Overlay: map[string]load.Source{
        "/abs/path/to/file.cue": load.FromString(`extra: "value"`),
    },

    // Skip all import resolution (no registry access)
    SkipImports: true,

    // Include _test.cue files
    Tests: true,
}
```

---

### Source constructors

#### `FromString`

```go
func FromString(s string) Source
```

Creates an overlay source from a string literal.

```go
overlay := map[string]load.Source{
    "/my/module/inject.cue": load.FromString(`injected: "value"`),
}
```

#### `FromBytes`

```go
func FromBytes(b []byte) Source
```

Creates an overlay source from a byte slice. Contents are not copied.

```go
src, _ := os.ReadFile("template.cue")
overlay := map[string]load.Source{
    "/my/module/template.cue": load.FromBytes(src),
}
```

#### `FromFile`

```go
func FromFile(f *ast.File) Source
```

Creates an overlay source from an `*ast.File`. The file must be error-free.

```go
f, _ := parser.ParseFile("gen.cue", generatedCUE, parser.ParseComments)
overlay := map[string]load.Source{
    "/my/module/gen.cue": load.FromFile(f),
}
```

---

### `DefaultTagVars`

```go
func DefaultTagVars() map[string]TagVar
```

Returns built-in injection variables for use with `@tag(key,var=name)`.

| Key | Value |
|---|---|
| `now` | Current time in RFC3339Nano |
| `os` | `runtime.GOOS` |
| `arch` | `runtime.GOARCH` |
| `cwd` | Current working directory |
| `username` | Current OS username |
| `hostname` | Hostname |
| `rand` | Random 128-bit hex string |

```go
cfg := &load.Config{
    Dir:     modulePath,
    TagVars: load.DefaultTagVars(),
}
// CUE: buildTime: string @tag(now,var=now)
```

---

### `GenPath`

```go
func GenPath(root string) string
```

Returns the directory for generated files within a module root.
Typically `<root>/cue.mod/gen`.

```go
genDir := load.GenPath("/path/to/module")
// genDir == "/path/to/module/cue.mod/gen"
```

---

## 8. Formatting — `cuelang.org/go/cue/format`

Formats CUE AST nodes and source bytes in canonical `cue fmt` style.

---

### `Node`

```go
func Node(node ast.Node, opt ...Option) ([]byte, error)
```

Formats an AST node. Accepts `*ast.File`, `[]ast.Decl`, `ast.Expr`,
`ast.Decl`, or `ast.Spec`.

```go
import "cuelang.org/go/cue/format"

v := ctx.CompileString(`{name:"my-app",replicas:3}`)
node := v.Syntax()
b, err := format.Node(node)
// b == []byte("{\n\tname:     \"my-app\"\n\treplicas: 3\n}")
```

With simplification:

```go
b, _ := format.Node(node, format.Simplify())
```

---

### `Source`

```go
func Source(b []byte, opt ...Option) ([]byte, error)
```

Formats raw CUE source bytes. Handles both full files and partial source.

```go
raw := []byte(`{name:"my-app",  replicas:3}`)
formatted, err := format.Source(raw)
```

---

### Format options

| Function | Description |
|---|---|
| `Simplify()` | Allow simplifications (remove unnecessary quotes, etc.) |
| `UseSpaces(tabwidth int)` | Convert tabs to spaces with given width |
| `TabIndent(indent bool)` | Control tab-based indentation |
| `IndentPrefix(n int)` | Add n tab stops of prefix to every line |

```go
// Format with 4-space indentation
b, _ := format.Node(node, format.UseSpaces(4), format.TabIndent(false))

// Indent by 2 extra tab stops
b, _ = format.Node(node, format.IndentPrefix(2))
```

---

## 9. Errors — `cuelang.org/go/cue/errors`

CUE errors carry structured information: position, path in the data tree,
and support for multiple errors from a single validation pass.

---

### `Newf`

```go
func Newf(p token.Pos, format string, args ...interface{}) Error
```

Creates a positioned CUE error.

```go
import cueerrors "cuelang.org/go/cue/errors"

err := cueerrors.Newf(token.NoPos, "invalid module: %s", name)
```

---

### `Wrapf`

```go
func Wrapf(err error, p token.Pos, format string, args ...interface{}) Error
```

Wraps an existing error with position and message context.

```go
if err := v.Validate(); err != nil {
    return cueerrors.Wrapf(err, v.Pos(), "validating module %s", name)
}
```

---

### `Errors`

```go
func Errors(err error) []Error
```

Splits a combined error into its individual components. CUE validation
often returns multiple errors in a single value — use this to iterate them.

```go
err := v.Validate(cue.Concrete(true))
for _, e := range cueerrors.Errors(err) {
    fmt.Printf("  path=%v  pos=%v  msg=%v\n",
        e.Path(), e.Position(), e.Error())
}
```

---

### `Append`

```go
func Append(a, b Error) Error
```

Combines two errors into a list. Either can be nil.

```go
var combined cueerrors.Error
for _, v := range values {
    if err := v.Validate(); err != nil {
        combined = cueerrors.Append(combined, cueerrors.Promote(err, ""))
    }
}
return combined
```

---

### `Sanitize`

```go
func Sanitize(err Error) Error
```

Sorts errors and removes duplicates on a best-effort basis.

```go
err = cueerrors.Sanitize(err)
```

---

### `Promote`

```go
func Promote(err error, msg string) Error
```

Converts a plain Go error to a CUE `Error`. No-op if already a CUE error.

```go
if err := someGoFunc(); err != nil {
    return cueerrors.Promote(err, "calling someGoFunc")
}
```

---

### `Positions`

```go
func Positions(err error) []token.Pos
```

Returns all positions from an error, sorted by relevance.

```go
for _, pos := range cueerrors.Positions(err) {
    fmt.Println(pos) // file.cue:12:5
}
```

---

### `Print` / `Details`

```go
func Print(w io.Writer, err error, cfg *Config)
func Details(err error, cfg *Config) string
```

Format errors for human consumption. `Details` returns a string.
The `Config` controls path prefix, cwd-relative paths, and custom format functions.

```go
// Print all errors with file paths relative to cwd
cueerrors.Print(os.Stderr, err, &cueerrors.Config{
    Cwd: "/path/to/module",
})

// Get as string
msg := cueerrors.Details(err, nil)
```

---

### `Is` / `As`

```go
func Is(err, target error) bool
func As(err error, target interface{}) bool
```

Standard Go `errors.Is`/`errors.As` wrappers that understand CUE error chains.

```go
if cueerrors.Is(err, io.EOF) { ... }

var cueErr cueerrors.Error
if cueerrors.As(err, &cueErr) {
    fmt.Println(cueErr.Path())
}
```

---

## 10. Go Codec — `cuelang.org/go/encoding/gocode/gocodec`

Bidirectional Go ↔ CUE conversion with constraint validation and auto-completion
via `cue:` struct field tags.

> **Naming inversion warning:** In this package, `Decode` means Go → CUE and
> `Encode` means CUE → Go. This is the opposite of `cue.Context`.

---

### `New`

```go
func New[Ctx *cue.Runtime | *cue.Context](ctx Ctx, c *Config) *Codec
```

Creates a Codec. Safe for concurrent `Decode`, `Validate`, and `Complete` calls.

```go
import "cuelang.org/go/encoding/gocode/gocodec"

ctx := cuecontext.New()
codec := gocodec.New(ctx, nil)
```

---

### `ExtractType`

```go
func (c *Codec) ExtractType(x interface{}) (cue.Value, error)
```

Extracts a CUE schema from a Go type's `cue:` struct field tags. The value
of x is ignored — only the type is used.

```go
type Config struct {
    Host string `json:"host"`
    Port int    `json:"port" cue:">0 & <=65535"`
    Mode string `json:"mode" cue:"\"http\" | \"https\""`
}

schema, err := codec.ExtractType(Config{})
// schema: {host: string, port: (>0 & <=65535), mode: ("http" | "https")}
```

---

### `Decode` (Go → CUE)

```go
func (c *Codec) Decode(x interface{}) (cue.Value, error)
```

Converts a Go value to a CUE `Value`. Use `ctx.Encode` for simple cases;
use this when you want `cue:` tag constraints included automatically.

```go
cfg := Config{Host: "localhost", Port: 8080, Mode: "http"}
v, err := codec.Decode(cfg)
// v: {host: "localhost", port: 8080, mode: "http"}
```

---

### `Encode` (CUE → Go)

```go
func (c *Codec) Encode(v cue.Value, x interface{}) error
```

Converts a CUE `Value` to a Go value. This is a thin wrapper around `v.Decode(x)`.

```go
v := ctx.CompileString(`{host: "localhost", port: 8080}`)
var cfg Config
err := codec.Encode(v, &cfg)
```

---

### `Validate`

```go
func (c *Codec) Validate(v cue.Value, x interface{}) error
```

Checks that a Go value satisfies the constraints in a CUE value. Typically
v is obtained from `ExtractType`, and x is the instance to check.

```go
schema, _ := codec.ExtractType(Config{})

good := Config{Host: "localhost", Port: 8080, Mode: "http"}
bad  := Config{Host: "localhost", Port: 99999, Mode: "http"}

codec.Validate(schema, good) // nil
codec.Validate(schema, bad)  // error: port out of range
```

---

### `Complete`

```go
func (c *Codec) Complete(v cue.Value, x interface{}) error
```

Fills in undefined fields in x that can be uniquely determined from v's
constraints. Only modifies nil pointers and zero-value fields with `json:",omitempty"`.
Performs a JSON round-trip — data not round-trippable through JSON (e.g.
`time.Time` timezone) is not preserved.

```go
type Sum struct {
    A int `cue:"C-B" json:",omitempty"`
    B int `cue:"C-A" json:",omitempty"`
    C int `cue:"A+B & >=5" json:",omitempty"`
}

schema, _ := codec.ExtractType(Sum{})

s := &Sum{A: 1, B: 4} // C is not set
err := codec.Complete(schema, s)
// s.C == 5 (computed from A+B)
```

---

### `Validate` (package-level)

```go
func Validate(x interface{}) error
```

Convenience function that uses a default Codec. Extracts the CUE type from
x, encodes x as a value, unifies, and validates.

```go
err := gocodec.Validate(Config{Host: "localhost", Port: 8080, Mode: "http"})
```

---

### `cue:` struct tag

Fields can be annotated with CUE expression constraints. These are used by
`ExtractType`, `Validate`, and `Complete`.

```go
type Config struct {
    // Simple constraint
    Port int `cue:">0 & <=65535" json:"port"`

    // Enumeration (disjunction)
    Mode string `cue:"\"http\" | \"https\"" json:"mode"`

    // Computed relationship
    Sum int `cue:"A+B" json:"sum,omitempty"`

    // Mark field optional in extracted type
    Name string `cue:",opt" json:"name,omitempty"`
}
```

---

## 11. JSON Encoding — `cuelang.org/go/encoding/json`

Converts between JSON and CUE AST. Used when you need to read JSON data
into CUE for unification or validation.

---

### `Extract`

```go
func Extract(path string, data []byte) (ast.Expr, error)
```

Parses JSON bytes into a CUE AST expression. The `path` is used for
position information in errors.

```go
import cuejson "cuelang.org/go/encoding/json"

data := []byte(`{"host": "localhost", "port": 8080}`)
expr, err := cuejson.Extract("values.json", data)
if err != nil {
    log.Fatal(err)
}
v := ctx.BuildExpr(expr)
```

---

### `Valid`

```go
func Valid(b []byte) bool
```

Reports whether the bytes are valid JSON.

```go
fmt.Println(cuejson.Valid([]byte(`{"a": 1}`))) // true
fmt.Println(cuejson.Valid([]byte(`{bad}`)))     // false
```

---

### `Validate`

```go
func Validate(b []byte, v cue.Value) error
```

Validates JSON bytes against a CUE value's constraints.

```go
schema := ctx.CompileString(`{port: int & >0 & <=65535}`)
data   := []byte(`{"port": 8080}`)

if err := cuejson.Validate(data, schema); err != nil {
    log.Fatal(err)
}
```

---

### `NewDecoder` / `(*Decoder).Extract`

```go
func NewDecoder(r *cue.Runtime, path string, src io.Reader) *Decoder
func (d *Decoder) Extract() (ast.Expr, error)
```

Streaming JSON → CUE decoder. `r` is unused (historical) and may be nil.
Call `Extract()` in a loop until `io.EOF`.

```go
dec := cuejson.NewDecoder(nil, "stream.json", reader)
for {
    expr, err := dec.Extract()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    v := ctx.BuildExpr(expr)
    // process v
}
```

---

### `PointerFromCUEPath`

```go
func PointerFromCUEPath(p cue.Path) (Pointer, error)
```

Converts a CUE path to a JSON Pointer (RFC 6901). Only `StringLabel`
and `IndexLabel` selectors are supported.

```go
p := cue.ParsePath("metadata.name")
ptr, err := cuejson.PointerFromCUEPath(p)
// ptr == "/metadata/name"
```

---

## 12. YAML Encoding — `cuelang.org/go/encoding/yaml`

Converts between YAML and CUE. Preserves comments and source positions.
Used for validating YAML config files against CUE schemas.

---

### `Extract`

```go
func Extract(filename string, src interface{}) (*ast.File, error)
```

Parses YAML into a CUE AST file. `src` can be `nil` (reads from filename),
`string`, `[]byte`, or `io.Reader`. Multi-document YAML (separated by `---`)
becomes a CUE list.

```go
import cueyaml "cuelang.org/go/encoding/yaml"

data, _ := os.ReadFile("values.yaml")
file, err := cueyaml.Extract("values.yaml", data)
if err != nil {
    log.Fatal(err)
}
v := ctx.BuildFile(file)
```

---

### `Encode`

```go
func Encode(v cue.Value) ([]byte, error)
```

Marshals a concrete CUE value to YAML bytes. Requires concrete value.

```go
v := ctx.CompileString(`{
    name:     "my-app"
    replicas: 3
    image:    "nginx:1.25"
}`)

b, err := cueyaml.Encode(v)
// b:
// name: my-app
// replicas: 3
// image: nginx:1.25
```

---

### `EncodeStream`

```go
func EncodeStream(iter cue.Iterator) ([]byte, error)
```

Encodes multiple CUE values to a YAML stream with `---` separators.

```go
list := ctx.CompileString(`[{a: 1}, {b: 2}]`)
iter, _ := list.List()
b, err := cueyaml.EncodeStream(iter)
// b:
// a: 1
// ---
// b: 2
```

---

### `Validate`

```go
func Validate(b []byte, v cue.Value) error
```

Validates YAML bytes against a CUE value's constraints. For YAML streams,
all documents must match.

```go
schema := ctx.CompileString(`{
    name:     string
    replicas: int & >0
}`)
data, _ := os.ReadFile("values.yaml")
if err := cueyaml.Validate(data, schema); err != nil {
    log.Fatalf("invalid values: %v", err)
}
```

---

### `NewDecoder` / `(*Decoder).Extract`

```go
func NewDecoder(path string, src io.Reader) *Decoder
func (d *Decoder) Extract() (ast.Expr, error)
```

Streaming YAML → CUE decoder. `Extract()` returns one document at a time
from a multi-document YAML stream. Returns `io.EOF` when exhausted.

```go
f, _ := os.Open("multi-doc.yaml")
dec := cueyaml.NewDecoder("multi-doc.yaml", f)
for {
    expr, err := dec.Extract()
    if err == io.EOF {
        break
    }
    v := ctx.BuildExpr(expr)
    // process each YAML document as a CUE value
}
```

---

## 13. Kind & Op Constants

### `Kind` constants

Used with `Value.Kind()` and `Value.IncompleteKind()`.

| Constant | Description |
|---|---|
| `BottomKind` | Error or non-concrete constraint value |
| `NullKind` | `null` |
| `BoolKind` | `bool` |
| `IntKind` | Integer (`int`, `uint`, etc.) |
| `FloatKind` | Decimal float (cannot be an integer) |
| `StringKind` | `string` |
| `BytesKind` | `bytes` |
| `StructKind` | Struct / object |
| `ListKind` | List / array |
| `NumberKind` | `IntKind | FloatKind` |
| `TopKind` | Top (`_`) — any value |

```go
switch v.IncompleteKind() {
case cue.StructKind:
    // handle struct
case cue.ListKind:
    // handle list
case cue.StringKind:
    s, _ := v.String()
case cue.IntKind:
    n, _ := v.Int64()
}
```

### `Op` constants

Returned by `Value.Expr()`.

| Constant | Description |
|---|---|
| `NoOp` | Leaf or single value |
| `AndOp` | Unification `&` |
| `OrOp` | Disjunction `\|` |
| `SelectorOp` | Field selector `.` |
| `IndexOp` | Index `[i]` |
| `CallOp` | Function call |
| `EqualOp` | `==` |
| `NotEqualOp` | `!=` |
| `LessThanOp` | `<` |
| `LessThanEqualOp` | `<=` |
| `GreaterThanOp` | `>` |
| `GreaterThanEqualOp` | `>=` |
| `RegexMatchOp` | `=~` |
| `NotRegexMatchOp` | `!~` |
| `AddOp` | `+` |
| `SubtractOp` | `-` |
| `MultiplyOp` | `*` |
| `FloatQuotientOp` | `/` |
| `IntQuotientOp` | `div` |
| `NotOp` | `!` |
| `InterpolationOp` | String interpolation |

---

## 14. Selector Types

### `SelectorType` constants

| Constant | Description |
|---|---|
| `StringLabel` | Regular field (`foo:`) |
| `IndexLabel` | List index (`[0]`) |
| `DefinitionLabel` | Definition (`#Foo:`) |
| `HiddenLabel` | Hidden field (`_foo:`) |
| `HiddenDefinitionLabel` | Hidden definition (`_#Foo:`) |
| `OptionalConstraint` | Optional field (`foo?:`) |
| `RequiredConstraint` | Required field (`foo!:`) |
| `PatternConstraint` | Pattern constraint (`[string]: T`) |

### `AttrKind` constants

| Constant | Description |
|---|---|
| `FieldAttr` | Attribute on a field: `foo: bar @attr()` |
| `DeclAttr` | Attribute on a declaration: `{ @attr() }` |
| `ValueAttr` | Bitmask: any attribute locally on a field |
