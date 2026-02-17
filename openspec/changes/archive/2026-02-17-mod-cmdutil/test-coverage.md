# Test Coverage Analysis: mod-cmdutil

## Summary

Post-refactoring test coverage is strong for flag logic and output formatting, but thin for the main render pipeline orchestrator (`RenderModule`). This document describes what additional unit tests would improve coverage.

## Current Coverage

### ✅ Well-Covered

| Component | File | Tests | Coverage |
|-----------|------|-------|----------|
| Flag group structs | `cmdutil/flags.go` | `flags_test.go` | All `AddTo()`, `Validate()`, `LogName()` methods tested |
| K8s client factory | `cmdutil/k8s.go` | `k8s_test.go` | Error path tested (invalid kubeconfig) |
| Output formatters | `cmdutil/output.go` | `output_test.go` | All branches tested (8 tests covering error types, availability, multi-error) |
| ShowRenderOutput | `cmdutil/render.go` | `render_test.go` | Error path, default mode, warnings tested |
| Module path resolution | `cmdutil/render.go` | `render_test.go` | Empty args and single-arg cases tested |

### ⚠️ Thin Coverage

| Component | File | What's Tested | What's Missing |
|-----------|------|---------------|----------------|
| RenderModule | `cmdutil/render.go:60` | Nil OPMConfig (line 65) | K8s resolution failure (line 85), RenderOptions validation failure (line 124), pipeline render failures (lines 138-139), success path |

---

## RenderModule Test Coverage Gaps

The `RenderModule` function at `cmdutil/render.go:60` has **6 code paths** (5 error + 1 success), but only **1 is unit-tested**.

### Path 1: Nil OPMConfig ✅ TESTED

```go
if opts.OPMConfig == nil {
    return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, ...}
}
```

**Test:** `render_test.go:13` (`TestRenderModule_NilConfig`)

---

### Path 2: K8s Config Resolution Failure ❌ UNTESTED

```go
k8sConfig, err := ResolveKubernetes(...)
if err != nil {
    return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, ...}
}
```

**What would trigger this:**
- An OPMConfig with a provider flag value that doesn't match any configured provider
- Example: `providerFlag = "nonexistent"` but `opmConfig.Providers` only has `{"kubernetes": ...}`

**Recommended test:**

```go
func TestRenderModule_K8sResolutionFailure(t *testing.T) {
    cfg := &config.OPMConfig{
        Config: &config.Config{
            Providers: map[string]config.Provider{
                "kubernetes": {...},
            },
        },
        Providers: map[string]cue.Value{
            "kubernetes": {...},
        },
    }

    _, err := RenderModule(context.Background(), RenderModuleOpts{
        Args:      []string{"."},
        Render:    &RenderFlags{Provider: "nonexistent"},
        OPMConfig: cfg,
    })

    require.Error(t, err)
    var exitErr *oerrors.ExitError
    require.True(t, errors.As(err, &exitErr))
    assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
    assert.Contains(t, exitErr.Error(), "resolving kubernetes config")
}
```

**Complexity:** Medium. Requires constructing a valid `OPMConfig` with provider definitions.

---

### Path 3: RenderOptions Validation Failure ❌ UNTESTED

```go
if err := renderOpts.Validate(); err != nil {
    return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
}
```

**What would trigger this:**
- The `build.RenderOptions.Validate()` method checks:
  - `ModulePath` is not empty (always satisfied via `ResolveModulePath` default)
  - No additional checks in current implementation

**Current state:** `Validate()` is unlikely to fail in practice with the current implementation. This path may not be worth unit-testing unless `Validate()` logic becomes more complex.

---

### Path 4: Pipeline Render Failure (ReleaseValidationError) ❌ UNTESTED

```go
result, err := pipeline.Render(ctx, renderOpts)
if err != nil {
    PrintValidationError("render failed", err)
    return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
}
```

**What would trigger this:**
- A module with invalid CUE (e.g., non-concrete values in `#config`)
- Example: `values.cue` has `replicas: int` but no concrete value provided

**Why it's hard to unit-test:**
- Requires a real CUE module fixture with a CUE context, provider definitions, and transformers
- Essentially duplicates the integration-level command tests (`TestModVet_ValidModule`, etc.)

**Recommendation:** Skip unit-testing this path. It's already covered by command-level integration tests.

---

### Path 5: Pipeline Render Failure (Generic Error) ❌ UNTESTED

Same as Path 4 — generic errors from `pipeline.Render()` are handled the same way. Covered by integration tests.

---

### Path 6: Success Path ❌ UNTESTED

```go
result, err := pipeline.Render(ctx, renderOpts)
// No error
return result, nil
```

**Why it's hard to unit-test:**
- Requires a valid CUE module, OPM config, provider with transformers, and registry
- Essentially an integration test

**Recommendation:** Skip unit-testing this path. It's already covered by command-level integration tests:
- `TestModVet_ValidModule` (mod_vet_test.go)
- `TestModBuildCmd_FlagBindings` (mod_build_test.go)
- `TestModApplyCmd_FlagBindings` (mod_apply_test.go)

---

## Recommendations

### High Value (Worth Adding)

1. **Path 2: K8s Config Resolution Failure** — This is a pure logic error (invalid provider name) that doesn't require real CUE modules. A unit test here would catch regressions in the K8s resolution logic.

### Low Value (Skip)

2. **Path 3: RenderOptions Validation Failure** — Current `Validate()` logic is trivial. Not worth testing unless it becomes more complex.
3. **Paths 4-6: Pipeline Render Paths** — These require real CUE modules and are already covered by command-level integration tests. Unit-testing them would duplicate existing coverage.

---

## Action Items for Future

- [ ] Add `TestRenderModule_K8sResolutionFailure` to `render_test.go`
- [ ] If `build.RenderOptions.Validate()` adds non-trivial logic in the future, add a corresponding test
- [ ] No action needed for pipeline render paths (covered by integration tests)

---

## Test File Summary

| File | Lines | Tests | Coverage Notes |
|------|-------|-------|----------------|
| `flags_test.go` | 160 | 7 | All flag registration, validation, and composition paths covered |
| `k8s_test.go` | 23 | 1 | Error path tested; success requires real cluster (acceptable) |
| `output_test.go` | 214 | 8 | All output formatter branches tested (validation errors, render errors, multi-error) |
| `render_test.go` | 105 | 6 | Module path resolution, nil config, ShowRenderOutput error/success/warnings tested |

**Total:** 502 test lines across 22 tests

**Overall Assessment:** Test coverage is strong for the refactored code. The main gap (K8s resolution failure) is low-risk and can be added later if needed. All user-facing behavior is covered by integration tests.
