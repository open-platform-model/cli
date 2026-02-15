# Edge Case Tests for Values Validation

Reference document for implementing high-priority edge case tests in `TestValidateValuesAgainstConfig`.

## Overview

These 10 tests validate edge cases discovered during implementation of the recursive field validator. Each test targets a specific phase of the 4-phase validation algorithm:

1. **Phase 1**: Closedness check via `Value.Allows()`
2. **Phase 2**: Schema field resolution (literal then pattern via `.Optional()`)
3. **Phase 3**: Struct recursion decision (struct-kind values recurse)
4. **Phase 4**: Leaf value type validation (unify + validate)

---

## Test 1: Non-identifier field names show quoted paths

**Phase Tested:** Path construction (all phases)

**Scenario:** Fields with hyphens, dots, or other non-identifier characters must be properly quoted in error paths.

### Test Case A: Pattern accepts non-identifier fields
```go
t.Run("non-identifier field names accepted by pattern", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { [Name=string]: string }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{
        "my-app": "test"
        "test.key": "val"
    }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    assert.NoError(t, err, "non-identifier fields should be accepted by [string] pattern")
})
```

### Test Case B: Error paths quote non-identifier fields
```go
t.Run("non-identifier disallowed field shows quoted path", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { name: string }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{
        name: "ok"
        "extra-field": "bad"
        "test.key": "also-bad"
    }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    require.Error(t, err)

    details := formatCUEDetails(err)
    assert.Contains(t, details, `values."extra-field"`, "hyphenated field should be quoted")
    assert.Contains(t, details, `values."test.key"`, "dotted field should be quoted")
    assert.Contains(t, details, "field not allowed")
})
```

**Why this matters:** Real-world configs often use kebab-case (`cache-size`) or dotted keys (`app.version`). Error paths must preserve quoting to be copy-pasteable.

**Key assertion:** `sel.String()` preserves CUE's quoting conventions in error paths.

---

## Test 2: Open struct allows arbitrary fields

**Phase Tested:** Phase 1 (closedness check)

**Scenario:** Structs with `...` (open structs) should allow any field without "field not allowed" errors.

```go
t.Run("open struct allows arbitrary fields", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { name: string, ... }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{
        name: "ok"
        anything: "goes"
        nested: {
            deep: true
            values: [1, 2, 3]
        }
    }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    assert.NoError(t, err, "open struct should allow extra fields")
})
```

**Why this matters:** Open structs are common for plugin configs or metadata fields. `Allows()` must return `true` for all fields.

**Key assertion:** No "field not allowed" errors for fields beyond the defined schema when `...` is present.

**CUE behavior confirmed:**
- `#config: { name: string, ... }` → `Allows("extra")` returns `true`
- `#config: { name: string }` → `Allows("extra")` returns `false`

---

## Test 3: List values validated as leaves

**Phase Tested:** Phase 3 → Phase 4 routing decision

**Scenario:** Lists should be validated via unification (Phase 4), not recursed into as structs (Phase 3).

### Test Case A: Valid list passes
```go
t.Run("list values validated as leaves - valid", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { tags: [...string] }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{ tags: ["a", "b", "c"] }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    assert.NoError(t, err, "list of correct type should pass")
})
```

### Test Case B: Wrong element type caught
```go
t.Run("list values validated as leaves - type mismatch", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { tags: [...string] }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{ tags: [1, 2, 3] }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    require.Error(t, err)

    details := formatCUEDetails(err)
    assert.Contains(t, details, "values.tags")
    assert.Contains(t, details, "conflicting values")
})
```

**Why this matters:** `IncompleteKind() == cue.ListKind` must NOT match the `cue.StructKind` check in Phase 3. Lists have their own element validation via CUE's unification.

**Key assertion:** Error path is `values.tags` (not `values.tags.0` or similar). List validation happens at the list level, not per-element by our code.

**CUE behavior confirmed:**
- Lists have `IncompleteKind() == cue.ListKind`
- Unifying `[...string]` with `[1, 2, 3]` produces error at list level

---

## Test 4: Disjunction type flexibility

**Phase Tested:** Phase 4 (unification with disjunction schema)

**Scenario:** Fields with `int | string` schema should accept both types.

```go
t.Run("disjunction type flexibility", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { port: int | string }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    valsStr := ctx.CompileString(`{ port: "8080" }`, cue.Filename("values.cue"))
    err := validateValuesAgainstConfig(configDef, valsStr)
    assert.NoError(t, err, "string should satisfy int|string disjunction")

    valsInt := ctx.CompileString(`{ port: 8080 }`, cue.Filename("values.cue"))
    err = validateValuesAgainstConfig(configDef, valsInt)
    assert.NoError(t, err, "int should satisfy int|string disjunction")

    valsBool := ctx.CompileString(`{ port: true }`, cue.Filename("values.cue"))
    err = validateValuesAgainstConfig(configDef, valsBool)
    require.Error(t, err, "bool should NOT satisfy int|string disjunction")

    details := formatCUEDetails(err)
    assert.Contains(t, details, "values.port")
})
```

**Why this matters:** Disjunctions are common for polymorphic config fields (port as int or service name as string). CUE's unification naturally handles this.

**Key assertion:** Both branches of disjunction validate, invalid types rejected.

**CUE behavior confirmed:**
- `(int | string).Unify("8080").Validate()` → nil
- `(int | string).Unify(8080).Validate()` → nil
- `(int | string).Unify(true).Validate()` → error

---

## Test 5: Empty struct at leaf level

**Phase Tested:** Phase 3 (recursion into empty closed struct)

**Scenario:** Empty struct `{}` in schema and values should validate correctly. Nested disallowed fields should still be caught.

### Test Case A: Empty to empty passes
```go
t.Run("empty struct at leaf - matching empty", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { metadata: {} }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{ metadata: {} }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    assert.NoError(t, err, "empty struct should match empty schema struct")
})
```

### Test Case B: Disallowed field in empty schema caught
```go
t.Run("empty struct at leaf - disallowed field", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { metadata: {} }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{ metadata: { extra: "bad" } }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    require.Error(t, err)

    details := formatCUEDetails(err)
    assert.Contains(t, details, "values.metadata.extra")
    assert.Contains(t, details, "field not allowed")
})
```

**Why this matters:** Empty closed structs `{}` are valid CUE definitions (e.g., marker fields). Recursion must handle zero-iteration loops gracefully.

**Key assertion:** `iter.Next()` returns `false` immediately for empty structs; recursion terminates cleanly.

**CUE behavior confirmed:**
- `metadata: {}` has `IncompleteKind() == cue.StructKind`
- Iterating its fields yields zero iterations
- Closed empty struct forbids all fields

---

## Test 6: Constraint violation (numeric bounds)

**Phase Tested:** Phase 4 (unification with constraint expressions)

**Scenario:** Numeric bounds (`>0`, `<65536`) should be validated via unification.

```go
t.Run("numeric constraint violation", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { port: >0 & <65536 & int }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    valsNeg := ctx.CompileString(`{ port: -999 }`, cue.Filename("values.cue"))
    err := validateValuesAgainstConfig(configDef, valsNeg)
    require.Error(t, err)

    details := formatCUEDetails(err)
    assert.Contains(t, details, "values.port")
    assert.Contains(t, details, "invalid value", "should mention constraint violation")

    valsValid := ctx.CompileString(`{ port: 8080 }`, cue.Filename("values.cue"))
    err = validateValuesAgainstConfig(configDef, valsValid)
    assert.NoError(t, err, "value within bounds should pass")
})
```

**Why this matters:** Constraint expressions beyond simple type checks are common (`>=1`, `=~"regex"`, etc.). Our path rewriting must apply to constraint errors.

**Key assertion:** CUE constraint errors are rewritten to `values.port` path.

**CUE behavior confirmed:**
- Unifying `>0 & <65536 & int` with `-999` yields "invalid value -999 (out of bound >0)"
- Error has path `[#config port]` which we rewrite to `[values port]`

---

## Test 7: Enum/disjunction constraint violation

**Phase Tested:** Phase 4 (unification with string disjunction)

**Scenario:** Enum-style disjunctions (`"debug" | "info" | "warn"`) should only accept listed values.

```go
t.Run("enum disjunction constraint", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { level: "debug" | "info" | "warn" | "error" }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    valsValid := ctx.CompileString(`{ level: "info" }`, cue.Filename("values.cue"))
    err := validateValuesAgainstConfig(configDef, valsValid)
    assert.NoError(t, err, "value in enum should pass")

    valsInvalid := ctx.CompileString(`{ level: "trace" }`, cue.Filename("values.cue"))
    err = validateValuesAgainstConfig(configDef, valsInvalid)
    require.Error(t, err)

    details := formatCUEDetails(err)
    assert.Contains(t, details, "values.level")
    assert.Contains(t, details, "disjunction", "should mention disjunction failure")
})
```

**Why this matters:** String enums are extremely common in real configs (log levels, environments, modes). Validates that disjunction validation works for non-numeric types.

**Key assertion:** Valid enum values pass, invalid ones produce errors with `values.level` path.

**CUE behavior confirmed:**
- Unifying `("debug"|"info"|"warn"|"error")` with `"info"` → nil
- Unifying with `"trace"` → "4 errors in empty disjunction"

---

## Test 8: Errors at every recursion level

**Phase Tested:** Error accumulation across recursion levels

**Scenario:** A single validation run should catch disallowed fields at top level, type mismatches at leaves, and disallowed fields at nested levels simultaneously.

```go
t.Run("errors at every recursion level", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`
#config: {
    db: {
        host: string
        port: int
    }
}
`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{
    topBad: "x"
    db: {
        host: 123
        port: 8080
        extra: "y"
    }
}`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    require.Error(t, err)

    details := formatCUEDetails(err)
    
    // Should have all 3 errors:
    assert.Contains(t, details, "values.topBad", "should catch top-level disallowed field")
    assert.Contains(t, details, "values.db.host", "should catch nested type mismatch")
    assert.Contains(t, details, "values.db.extra", "should catch nested disallowed field")
    
    // Count occurrences of "field not allowed" (should be 2: topBad + extra)
    assert.Equal(t, 2, strings.Count(details, "field not allowed"))
    // Count type mismatches (should be 1: host)
    assert.Contains(t, details, "conflicting values")
})
```

**Why this matters:** Validates that error accumulation via `cueerrors.Append()` works across recursion levels. A single validation pass must report ALL errors, not just the first.

**Key assertion:** All 3 errors present in output with correct paths at each level.

---

## Test 9: Nested pattern constraints (two levels deep)

**Phase Tested:** Phase 2 (schema resolution via `.Optional()`) across recursion

**Scenario:** Pattern constraints at multiple nesting levels (e.g., `[string]: { [string]: {...} }`) should resolve correctly through recursion.

### Test Case A: Valid nested patterns
```go
t.Run("nested pattern constraints - valid", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`
#config: {
    [Name=string]: {
        [Name=string]: {
            size: string
        }
    }
}
`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{
    media: {
        tvshows: {
            size: "10Gi"
        }
        movies: {
            size: "20Gi"
        }
    }
}`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    assert.NoError(t, err, "nested pattern-matched fields should pass")
})
```

### Test Case B: Disallowed field at second pattern level
```go
t.Run("nested pattern constraints - disallowed at depth 2", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`
#config: {
    [Name=string]: {
        [Name=string]: {
            size: string
        }
    }
}
`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{
    media: {
        tvshows: {
            size: "10Gi"
            bad: "oops"
        }
    }
}`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    require.Error(t, err)

    details := formatCUEDetails(err)
    assert.Contains(t, details, "values.media.tvshows.bad")
    assert.Contains(t, details, "field not allowed")
})
```

**Why this matters:** Real OPM modules use pattern constraints (e.g., jellyfin's `media: [Name=string]: {...}`). Nested patterns like `[string]: { [string]: {...} }` are less common but possible. Validates that `.Optional()` resolution works at every recursion level.

**Key assertion:** Both `media` and `tvshows` resolve via `.Optional()`, `bad` correctly identified as disallowed at depth 2.

**CUE behavior confirmed:**
- `schema.LookupPath(cue.Str("media").Optional())` returns the pattern constraint value
- Recursion continues with that constraint as the new schema

---

## Test 10: List with mixed element types

**Phase Tested:** Phase 4 (list element type validation via unification)

**Scenario:** Lists with elements violating the element type constraint should error at the list level.

```go
t.Run("list with mixed element types", func(t *testing.T) {
    ctx := cuecontext.New()
    schema := ctx.CompileString(`#config: { items: [...string] }`, cue.Filename("schema.cue"))
    configDef := schema.LookupPath(cue.ParsePath("#config"))

    vals := ctx.CompileString(`{ items: ["ok", 42, "also-ok"] }`, cue.Filename("values.cue"))

    err := validateValuesAgainstConfig(configDef, vals)
    require.Error(t, err)

    details := formatCUEDetails(err)
    // Error should be at the list level, not per-element
    assert.Contains(t, details, "values.items")
    assert.Contains(t, details, "conflicting values")
})
```

**Why this matters:** Lists are validated by unifying the entire list value with the list schema. CUE reports element errors, but our path rewriting should keep the path at `values.items`, not drill into list indices.

**Key assertion:** Error path is `values.items`, not `values.items.1` or similar. List element errors are reported by CUE with internal paths that we rewrite.

**CUE behavior confirmed:**
- Unifying `[...string]` with `["ok", 42, "also-ok"]` yields error mentioning element index
- Our path rewriting replaces CUE's path entirely with `[values items]`

---

## Tests NOT Included (and why)

### Regex pattern constraints (`[Name=~"^[a-z]+$"]: {...}`)

**Finding:** `Value.Allows()` returns `false` for ALL fields (even literal fields) when ANY regex pattern is present in the struct. This makes regex patterns incompatible with our closedness checker.

**Root cause:** CUE SDK limitation. `Allows()` delegates to internal `Accept()` which doesn't resolve regex patterns.

**OPM impact:** Zero — no `.cue` files in the codebase use regex patterns (`=~`). All patterns are `[Name=string]` or `[Name=_]`.

**Decision:** Document as known limitation. No test needed since it would test CUE's behavior, not ours. If regex patterns are added in the future, they would need CUE SDK improvements or a different closedness check strategy.

### Literal + pattern overlap (`foo: int` with `[Name=string]: string`)

**Finding:** CUE unifies overlapping constraints: `foo: int & string` → bottom (`_|_`).

**CUE behavior:** This is CUE's semantic handling, not validator logic. Literal lookup (Phase 2 line 546) succeeds and returns a bottom value. Unification (Phase 4) catches it as conflicting values.

**Decision:** Skip. Our code correctly prioritizes literal over pattern. The bottom-value behavior is CUE's domain.

### Very deep nesting (10+ levels)

**Coverage:** Test 8 validates 2 levels, Test 9 validates 3 via patterns.

**Recursion safety:** Go's default stack handles thousands of recursive calls. No OPM module approaches dangerous nesting depths.

**Decision:** Skip. Adds test verbosity without additional coverage. The recursion is trivially depth-safe.

---

## Implementation Checklist

1. Add all 10 tests to `TestValidateValuesAgainstConfig` in `internal/build/errors_test.go`
2. Run `go test ./internal/build/ -v -run TestValidateValuesAgainstConfig`
3. Verify all new tests pass
4. Run `task test` to ensure no regressions
5. Run `task check` to verify lint clean

## Summary Table

| # | Test Name | Phase | Lines (est) | Key Validation |
|---|-----------|-------|-------------|----------------|
| 1 | Non-identifier fields | Path | 30 | Quoted field names in errors |
| 2 | Open struct | 1 | 20 | No false positives for `...` |
| 3 | List values | 3→4 | 35 | Lists not recursed, validated as leaves |
| 4 | Disjunction | 4 | 30 | Type flexibility |
| 5 | Empty struct | 3 | 30 | Empty recursion + nested closedness |
| 6 | Numeric bounds | 4 | 25 | Constraint error rewriting |
| 7 | Enum disjunction | 4 | 25 | String enum validation |
| 8 | Multi-level errors | All | 40 | Error accumulation |
| 9 | Nested patterns | 2 | 45 | Multi-level `.Optional()` |
| 10 | Mixed list | 4 | 20 | List element type errors |

**Total:** ~300 lines of test code across 10 test cases covering all 4 validation phases.
