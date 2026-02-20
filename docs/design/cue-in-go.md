# CUE in Go: Loading, Storing, and Manipulating Values

Research report based on the `experiments/ast-pipeline` spike and analysis of the
production build pipeline (`internal/build/`).

---

## The Mental Model

`cue.Value` is not CUE source text. It is a small struct — roughly an index plus a
pointer back to a `*cue.Context` — that acts as a handle into an immutable evaluation
graph managed by the context.

```
┌──────────────────────────────────────────────────────────────────┐
│                        *cue.Context                              │
│                    (the evaluation engine)                       │
│                                                                  │
│   ┌────────────────────────────────────────────────────────┐     │
│   │                 Internal Value Graph                   │     │
│   │                                                        │     │
│   │   node_1 ──── node_2 ──── node_3 ──── node_4          │     │
│   │      │                      │            │             │     │
│   │    "foo"                  #Def          ∧             │     │
│   │                                                        │     │
│   └────────────────────────────────────────────────────────┘     │
│         ▲               ▲               ▲                        │
│     cue.Value       cue.Value       cue.Value                    │
│    (just a ref)   (just a ref)   (just a ref)                    │
└──────────────────────────────────────────────────────────────────┘
```

Two rules govern everything:

1. **All `cue.Value` operations are pure** — `Unify`, `FillPath`, `LookupPath` always
   return a new value. The original is never mutated.
2. **All values in a pipeline must share one `*cue.Context`** — mixing values from
   different contexts panics with "values are not from the same runtime."

---

## Loading CUE into Go

Three entry points cover every case.

### A. From a module on disk

Use `load.Instances` + `ctx.BuildInstance`. This resolves imports, reads `cue.mod/`,
and handles external registry dependencies.

```go
import (
    "cuelang.org/go/cue/cuecontext"
    "cuelang.org/go/cue/load"
)

ctx := cuecontext.New()

cfg := &load.Config{Dir: "/path/to/module"}
instances := load.Instances([]string{"."}, cfg)
if instances[0].Err != nil {
    return fmt.Errorf("load: %w", instances[0].Err)
}

value := ctx.BuildInstance(instances[0])
if value.Err() != nil {
    return fmt.Errorf("build: %w", value.Err())
}
```

This is the heaviest path. Use it when you need full import resolution (e.g., modules
that import `opmodel.dev/core@v0`).

**Overlay injection** — inject generated CUE files without writing to disk:

```go
cfg := &load.Config{
    Dir: modulePath,
    Overlay: map[string]load.Source{
        modulePath + "/generated.cue": load.FromBytes(generatedBytes),
    },
}
```

### B. From bytes or a string

Use `ctx.CompileBytes` or `ctx.CompileString`. No import resolution. Good for embedded
schemas, single-file fragments, and tests.

```go
// From bytes (preferred — always pass Filename for error messages)
content, _ := os.ReadFile("schema.cue")
value := ctx.CompileBytes(content, cue.Filename("schema.cue"))
if value.Err() != nil { ... }

// From a string literal (tests and inline CUE)
value := ctx.CompileString(`
    #Config: {
        image:    string
        replicas: int & >0
    }
`)
```

### C. From a Go value

Wrap a Go struct, map, or scalar into CUE via `ctx.Encode`. Used for injecting
concrete data into CUE definitions.

```go
type Meta struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

meta := Meta{Name: "my-app", Version: "1.0.0"}
metaVal := ctx.Encode(meta)

// Scalars work too
nameVal := ctx.Encode("my-app")

// Inject into a CUE value at a path
filled := baseVal.FillPath(cue.ParsePath("#context.name"), nameVal)
```

---

## Storing `cue.Value`

`cue.Value` is a value type (a struct, not a pointer). Storing it in any container is
free and safe — no heap allocation, no reference counting.

```go
// Direct field — most common
type Component struct {
    Value     cue.Value
    Spec      cue.Value
    Resources map[string]cue.Value
    Traits    map[string]cue.Value
}

// Map — FQN → value, stored by value
resources := map[string]cue.Value{}
iter, _ := val.LookupPath(cue.ParsePath("#resources")).Fields()
for iter.Next() {
    resources[iter.Label()] = iter.Value() // independent copy
}

// Slice — ordered collection
var declared []cue.Value
iter2, _ := val.List()
for iter2.Next() {
    declared = append(declared, iter2.Value())
}
```

Do not store `*cue.Context` in every struct that holds a value. Retrieve it from a
value when needed:

```go
// Recover the context from any value you already have
ctx := someValue.Context()
newVal := ctx.CompileString(`extra: "field"`)
```

---

## Copying and Cloning

### Copy by assignment (free, always correct)

Since `cue.Value` is a value type, any assignment or parameter pass is already a copy:

```go
original := ctx.CompileString(`x: 1, y: 2`)

a := original           // copy — points to same immutable node
b := original           // another copy
m["key"] = original     // stored in map by value

someFunc(original)      // passed by value, caller's copy unchanged

// a, b, m["key"], and original are all independent handles.
// Manipulating one does not affect the others.
```

### Divergent copies via operations

Operations always produce new values without touching the original:

```go
// FillPath — produces a new value with the path filled
filled := base.FillPath(cue.ParsePath("#config.image"), ctx.Encode("nginx:1.0"))
// base is unchanged; filled has the new field

// Unify — CUE lattice meet (∧)
merged := schema.Unify(concrete)
// schema and concrete are unchanged; merged is their intersection

// LookupPath — extract a sub-value
spec := component.LookupPath(cue.ParsePath("spec"))
// component is unchanged; spec is a sub-handle into the same graph
```

You can hold both the "before" and "after" simultaneously:

```go
base := ctx.CompileString(`replicas: 1`)
withTwo := base.FillPath(cue.ParsePath("replicas"), ctx.Encode(2))
withThree := base.FillPath(cue.ParsePath("replicas"), ctx.Encode(3))
// base still has replicas: 1, withTwo has 2, withThree has 3
```

### Cross-context copy (for parallel goroutines)

When you need a genuinely independent copy in a new context — for example to run
concurrent transformer jobs — you must round-trip through source bytes. This is because
**FillPath across different contexts panics**.

```go
// Proven in: experiments/ast-pipeline/cross_context_test.go

// Step 1 (single-threaded): serialize inst.Files to immutable bytes
type fileSource struct {
    filename string
    content  []byte
}
var sources []fileSource
for _, file := range inst.Files {
    b, err := format.Node(file)
    if err != nil { ... }
    sources = append(sources, fileSource{file.Filename, b})
}

// Step 2 (per goroutine): parse fresh, build in own context
go func() {
    ctx := cuecontext.New()
    var val cue.Value
    for i, src := range sources {
        f, _ := parser.ParseFile(src.filename, src.content, parser.ParseComments)
        if i == 0 {
            val = ctx.BuildFile(f)
        } else {
            val = val.Unify(ctx.BuildFile(f))
        }
    }
    // Now compile transformer in the SAME context — cross-context FillPath panics
    tfVal := ctx.CompileString(transformerCUE)
    unified := tfVal.FillPath(cue.ParsePath("#component"), componentVal)
    // ...
}()
```

For modules with external imports (e.g., `opmodel.dev/core@v0`), `ctx.BuildFile` cannot
resolve those imports. Use the reload-per-goroutine strategy instead:

```go
go func() {
    ctx := cuecontext.New()
    instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
    val := ctx.BuildInstance(instances[0])
    // ... completely independent, no shared state
}()
```

---

## `cue.Value` → AST → `cue.Value`

The full round-trip works for self-contained CUE. Use this when you need structural
manipulation: adding fields, removing definitions, renaming labels — things that
`FillPath` and `Unify` cannot express.

### Step 1: Value to AST

```go
import "cuelang.org/go/cue/format"

// Default: includes hidden fields, definitions, optional fields, attributes.
// Only omits doc comments.
node := val.Syntax()

// With doc comments
node = val.Syntax(cue.Docs(true))

// Only concrete values (defaults resolved, constraints removed)
node = val.Syntax(cue.Final(), cue.Concrete(true))

// Explicit: show absolutely everything
node = val.Syntax(cue.All(), cue.Docs(true))
```

`Syntax()` returns `ast.Node` — type-switch to get the concrete type:

```go
switch n := node.(type) {
case *ast.File:
    // Full file with package clause
case *ast.StructLit:
    // Struct literal (no package)
default:
    // Expression
}
```

### Step 2: Manipulate the AST

**Walk and modify with `astutil.Apply`:**

```go
import "cuelang.org/go/cue/ast/astutil"

// Change a field value
astutil.Apply(file, func(c astutil.Cursor) bool {
    field, ok := c.Node().(*ast.Field)
    if !ok {
        return true // descend
    }
    ident, ok := field.Label.(*ast.Ident)
    if ok && ident.Name == "replicas" {
        c.Replace(&ast.Field{
            Label: ast.NewIdent("replicas"),
            Value: ast.NewLit(token.INT, "5"),
        })
    }
    return true
}, nil)

// Delete a field
astutil.Apply(file, func(c astutil.Cursor) bool {
    if field, ok := c.Node().(*ast.Field); ok {
        if ident, ok := field.Label.(*ast.Ident); ok && ident.Name == "deprecated" {
            c.Delete()
        }
    }
    return true
}, nil)
```

**Append fields directly to `Decls`:**

```go
// Add a regular field
file.Decls = append(file.Decls, &ast.Field{
    Label: ast.NewIdent("injected"),
    Value: ast.NewString("value"),
})

// Add a definition
file.Decls = append([]ast.Decl{
    &ast.Field{
        Label: ast.NewIdent("#MyDef"),
        Value: ast.NewStruct(
            "field", ast.NewString("value"),
        ),
    },
}, file.Decls...)
```

**Merge two files:**

```go
f1.Decls = append(f1.Decls, f2.Decls...)
```

### How hidden fields and definitions appear in the AST

There is no separate node type. All fields are `*ast.Field`. The label's `Name` string
determines the kind:

```
Label string → field kind:

  "foo"   → regular field          foo: value
  "#Foo"  → definition             #Foo: value
  "_foo"  → hidden field           _foo: value
  "_#Foo" → hidden definition      _#Foo: value

Optional fields:
  field.Constraint == token.OPTION  → foo?: value
```

Identifying them while walking:

```go
astutil.Apply(file, func(c astutil.Cursor) bool {
    field, ok := c.Node().(*ast.Field)
    if !ok {
        return true
    }
    ident, ok := field.Label.(*ast.Ident)
    if !ok {
        return true
    }
    switch {
    case strings.HasPrefix(ident.Name, "_#"):
        // hidden definition
    case strings.HasPrefix(ident.Name, "#"):
        // definition
    case strings.HasPrefix(ident.Name, "_"):
        // hidden field
    default:
        // regular field
    }
    return true
}, nil)
```

### Step 3: AST back to `cue.Value`

```go
// Path A: BuildFile directly (no import resolution)
val := ctx.BuildFile(file)
if val.Err() != nil { ... }

// Path B: Format → bytes → CompileBytes
b, err := format.Node(file)
if err != nil { ... }
val := ctx.CompileBytes(b, cue.Filename("modified.cue"))

// Path C: Format → bytes → ParseFile → BuildFile
// (Useful when you need a fresh *ast.File with scope resolved)
b, _ := format.Node(file)
fresh, _ := parser.ParseFile("modified.cue", b, parser.ParseComments)
val := ctx.BuildFile(fresh)
```

**Limitation:** `ctx.BuildFile` does not resolve external imports. If the CUE source
contains `import "opmodel.dev/core@v0"`, BuildFile will fail. For those cases, format
the modified AST to bytes, write it as an overlay, and use `load.Instances`:

```go
b, _ := format.Node(file)
cfg := &load.Config{
    Dir: modulePath,
    Overlay: map[string]load.Source{
        overlayPath: load.FromBytes(b),
    },
}
instances := load.Instances([]string{"."}, cfg)
val := ctx.BuildInstance(instances[0])
```

---

## Validation

Four levels, applied at different pipeline stages:

```go
// 1. Did compilation succeed?
val := ctx.CompileBytes(src, cue.Filename("f.cue"))
if val.Err() != nil {
    return fmt.Errorf("compile: %w", val.Err())
}

// 2. Is the schema structurally valid? (no concrete requirement)
if err := val.Validate(); err != nil {
    return fmt.Errorf("schema: %w", err)
}

// 3. Are all values concrete? (everything filled in)
if err := val.Validate(cue.Concrete(true)); err != nil {
    return fmt.Errorf("incomplete: %w", err)
}

// 4. After FillPath — inline error check
unified := base.FillPath(cue.ParsePath("#config"), configVal)
if unified.Err() != nil {
    return fmt.Errorf("fill: %w", unified.Err())
}
```

---

## Gotchas

### 1. `ast.NewStruct` creates quoted string labels

```go
// BAD — creates "name": "value" (quoted label, not scope-visible)
ast.NewStruct("name", ast.NewString("value"))

// GOOD — creates name: "value" (identifier label, scope-visible)
&ast.Field{
    Label: ast.NewIdent("name"),
    Value: ast.NewString("value"),
}
```

Quoted labels break cross-scope reference resolution. Any label that another field
might reference by name must be an identifier, not a quoted string.

### 2. Programmatic AST needs `astutil.Resolve`

The parser wires up `Ident.Scope` automatically. Programmatically constructed nodes
do not have this. Call `astutil.Resolve` after any construction that involves
cross-scope references:

```go
import "cuelang.org/go/cue/ast/astutil"

astutil.Resolve(file, func(pos token.Pos, msg string, args ...interface{}) {
    log.Printf("resolve error at %v: "+msg, append([]interface{}{pos}, args...)...)
})
```

Safe to call unconditionally — it is a no-op on already-resolved nodes.

### 3. FillPath across different contexts panics

```go
ctxA := cuecontext.New()
valA := ctxA.CompileString(`x: 1`)

ctxB := cuecontext.New()
valB := ctxB.CompileString(`y: 2`)

// PANICS: "values are not from the same runtime"
valA.FillPath(cue.ParsePath("y"), valB)
```

All values in any `Unify`, `FillPath`, or comparison operation must share the same
`*cue.Context`. Retrieve the context from an existing value with `val.Context()` rather
than creating new contexts in helper functions.

### 4. `ast.NewStruct` variadic label ordering

`ast.NewStruct` takes alternating `label, value` pairs. Both must be `ast.Node`:

```go
// Labels must be ast.Node — use ast.NewString for quoted, ast.NewIdent for bare
ast.NewStruct(
    ast.NewIdent("name"), ast.NewString("value"),   // name: "value"
    ast.NewString("x-header"), ast.NewString("v"),  // "x-header": "v"
)
```

Passing a raw Go `string` as a label creates a `*ast.BasicLit`, which becomes a
quoted string label.

### 5. `Syntax()` options interact with each other

`cue.Concrete(true)` implies `final=true` and forces `omitHidden=true` and
`omitDefinitions=true`. You cannot get concrete values and hidden fields at the same
time. If you need both concrete data and structure inspection, do two separate
`Syntax()` calls with different options.

### 6. Round-trip fidelity with external imports

```
val.Syntax() → format.Node() → CompileBytes()
    [x] Works for self-contained CUE
    [ ] Fails for CUE with unresolved external imports
        (BuildFile/CompileBytes do not resolve registry packages)

For modules with external imports, use the overlay approach:
    format.Node() → []byte → load.Config.Overlay → load.Instances → BuildInstance
```

---

## Decision Guide

```
┌──────────────────────────────────────────────────────────────────┐
│  Goal                     Approach                               │
├──────────────────────────────────────────────────────────────────┤
│  Load a module from disk  load.Instances + ctx.BuildInstance     │
│  Load inline CUE source   ctx.CompileString / ctx.CompileBytes   │
│  Wrap a Go value          ctx.Encode(goValue)                    │
│                                                                  │
│  Inject/override a field  val.FillPath(path, newVal)             │
│  Merge two values         a.Unify(b)                             │
│  Navigate into a value    val.LookupPath(path)                   │
│  Iterate struct fields    val.Fields() iterator                  │
│                                                                  │
│  Add/remove/rename fields val.Syntax() → astutil.Apply →         │
│   structurally             ctx.BuildFile()                       │
│                                                                  │
│  "Clone" for a goroutine  format.Node(inst.Files) → []byte →     │
│   (self-contained module)  parser.ParseFile → ctx.BuildFile      │
│                                                                  │
│  "Clone" for a goroutine  load.Instances + ctx.BuildInstance     │
│   (module with imports)    per goroutine (fully independent)     │
│                                                                  │
│  Extract data to Go       val.Decode(&goStruct)                  │
│  Extract concrete source  val.Syntax(cue.Final(), Concrete(true))│
│  Extract full schema      val.Syntax() or val.Syntax(cue.All())  │
└──────────────────────────────────────────────────────────────────┘
```

---

## Experiments Reference

All claims in this document are validated by tests in
`experiments/ast-pipeline/`. Key test files:

| File | What it validates |
|------|-------------------|
| `ast_basics_test.go` | Construction, Value → AST, round-trips, comment preservation |
| `ast_manipulation_test.go` | Add/remove/change fields via AST, overlay injection |
| `ast_inspection_test.go` | Walking AST, extracting metadata without evaluation |
| `cross_context_test.go` | Cross-context FillPath panic, race-free parallel strategies |
| `parallel_test.go` | Concurrent evaluation, shared vs independent contexts |
| `overlay_test.go` | AST-based overlay generation vs fmt.Sprintf |
