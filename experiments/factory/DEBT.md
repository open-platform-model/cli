# Technical Debt Register — `experiments/factory`

This file records issues identified during code review that are intentionally left
unresolved because this directory is an experiment. Each item must be addressed before
any of this code is promoted to the production CLI.

Items are grouped by severity: **High → Medium → Low → Architectural**.

---

## High

### Silent error discard on metadata decode

**File:** `pkg/engine/execute.go:195–199`

`Decode()` errors for `metadata.labels` and `metadata.annotations` are silently
discarded via `_ =`. If a CUE value exists but cannot be decoded (e.g. unexpected
schema shape), the transformer receives empty maps with no indication that anything
went wrong. This can silently produce incorrectly labelled Kubernetes manifests.

**Production fix:** Propagate or log the error so the operator is informed. A
structured log entry at `WARN` level with the component name, field path, and
underlying CUE error is the minimum bar. Consider returning the error if strict mode
is enabled.

---

## Medium

### `joinErrors` loses error identity

**File:** `pkg/engine/module_renderer.go:161–166`

The function collapses a slice of errors into a flat string via `fmt.Errorf("%s", ...)`.
The resulting error cannot be unwrapped: `errors.Is`, `errors.As`, and any type-based
error handling downstream silently fail (e.g. checking for typed render errors).

**Production fix:** Use `errors.Join(errs...)` (Go 1.20+). This was fixed in the
experiment itself — carry the same change forward.

---

### Non-deterministic `Warnings()` output

**File:** `pkg/engine/matchplan.go:60–71`

`MatchPlan.Warnings()` iterates `p.UnhandledTraits` (a `map[string][]string`) without
sorting. Warning output order differs on every run, complicating diffs and testing.
`MatchedPairs()` in the same file is carefully sorted; `Warnings()` should be too.

**Production fix:** Collect map keys, sort them, then iterate. Sort the inner trait
slice as well.

---

### Dead `pkgName` field / missing `PkgName()`

**File:** `internal/core/module/module.go:29–32`

The `pkgName` field has a doc comment claiming it is "set by `module.Load()`" and
"accessed via `PkgName()`", but neither function exists anywhere in this experiment.
The `ModulePath` field is similarly never set.

**Production fix:** Either implement `module.Load()` and `PkgName()`, or remove the
field and its stale comment. Dead fields with misleading documentation are worse than
no documentation.

---

### Fail-fast vs. fail-slow asymmetry across pipeline levels

**Files:** `pkg/engine/bundle_renderer.go:80–83`, `pkg/engine/execute.go:53–70`

`executeTransforms` (pair level) collects all errors before returning. `BundleRenderer`
(release level) stops on the first failed `ModuleRelease`. The two levels have opposite
failure semantics with no documentation or rationale.

**Production fix:** Decide on a single consistent strategy and document it. For user
experience, fail-slow (collect all errors) at both levels is generally preferable so
the operator sees all failures in one pass.

---

### `isSingleResource` heuristic is fragile

**File:** `pkg/engine/execute.go:211–214`

The entire dispatch between "single resource" and "map of named resources" rests on
whether the output struct has an `apiVersion` field. This is a Kubernetes-specific
convention baked into a general-purpose abstraction. A transformer output that happens
to include `apiVersion` in a map value would be incorrectly classified.

**Production fix:** Define an explicit CUE output contract (a disjunction in the
schema) and validate the output shape against it rather than inferring by field
presence. This enforces the three supported forms at definition time.

---

### `pkg/loader` has too many responsibilities

**File:** `pkg/loader/`

The package handles: CUE instance loading, value finalisation, schema validation,
Go struct extraction/decoding, and kind detection. These are five distinct concerns.
`validateConfig` / `ConfigError` is already coherent enough to stand alone.

**Production fix:** Split into `loader` (CUE loading only), `validator` (config gates),
and let decoding helpers live closer to the types they decode (e.g. on the core types
themselves).

---

### Dual `Schema` + `DataComponents` on `ModuleRelease`

**File:** `internal/core/modulerelease/release.go`, `pkg/engine/execute.go`

`ModuleRelease` carries two `cue.Value` fields with the same Go type but different
invariants: `Schema` retains definition fields (used in match phase) and `DataComponents`
has them stripped (used in execute phase). Misuse is invisible to the compiler.

**Production fix:** Encapsulate both behind typed accessors (`MatchComponents()`,
`ExecuteComponents()`) or a dedicated struct, making correct usage the only easy path.

---

## Low

### `LoadRelease` is dead code

**File:** `pkg/loader/module_release.go:105–111`

`LoadRelease` is a public convenience wrapper that is never called — `cmd/main.go` calls
the two underlying functions directly. It has no tests.

**Production fix:** Remove it, or make it the canonical entry point and update all
callers.

---

### Extension-based directory detection

**File:** `pkg/loader/module_release.go:183–192`

`resolveReleaseFile` checks `filepath.Ext(path) != ".cue"` to decide whether the
argument is a directory. A missing path with no extension is silently treated as a
directory; the failure is only caught later by CUE with a less clear message.

**Production fix:** Use `os.Stat()` and check `IsDir()`. Return a clear error for
non-existent paths.

---

### `fmt.Printf` mixed into business logic / stdout conflict

**File:** `cmd/main.go`

Diagnostic progress messages and rendered YAML resource output both go to stdout.
Structured logging (`charmbracelet/log`, used everywhere else in the CLI) is not used.

**Production fix:** Use `charmbracelet/log` for diagnostics (stderr). Resource output
should go to a configurable `io.Writer`, defaulting to stdout, so callers can redirect
independently.

---

### Value vs pointer embedding inconsistency

**Files:** `internal/core/modulerelease/release.go`, `internal/core/bundlerelease/release.go`

`Module` and `Bundle` are embedded by value while `Metadata` fields on the same structs
are pointers. There is no rationale for the asymmetry. This produces two different zero
values for "not set": a zero struct vs. a nil pointer.

**Production fix:** Pick one convention (prefer pointers for optional/large nested
structs) and apply it consistently. Document the choice.

---

### Partial duplication of extract pattern

**Files:** `pkg/loader/module_release.go:151`, `pkg/loader/bundle_release.go:224–231`

`decodeModuleReleaseEntry` manually constructs `ReleaseMetadata` and calls `Decode` in
the same way as `extractReleaseMetadata` — they could share the same function.

**Production fix:** Unify into a single `extractReleaseMetadata` call from both sites.

---

### `BundleRelease.Schema` is never consumed

**File:** `internal/core/bundlerelease/release.go:32`

`BundleRelease.Schema` holds a reference to the entire top-level CUE package value
but is never read by any engine code. It holds the entire CUE evaluation tree in memory.

**Production fix:** Remove the field, or document and test a concrete use case for it.
