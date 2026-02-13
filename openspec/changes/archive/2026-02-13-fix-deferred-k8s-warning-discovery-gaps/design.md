## Context

The `improve-k8s-api-warnings-and-discovery` change introduced a custom `opmWarningHandler` (routing K8s API warnings through charmbracelet/log), `ExcludeOwned` filtering in discovery, and a config option for warning behavior. During verification, several gaps were identified and deferred:

1. **Warning handler tests only assert no-panic** — the `wantLevel` field in the test table is declared but never used. The routing logic is untested because `output.Warn()` / `output.Debug()` are package-level functions on an unexported logger.

2. **Discovery silently swallows unavailable API groups** — `discoverAPIResources()` at `discovery.go:196-201` handles `IsGroupDiscoveryFailedError` by continuing, but never logs a warning. The spec requires logging.

3. **Config init omits `cue mod tidy` hint** — the generated config imports `opmodel.dev/providers@v0`, but the success output only suggests `opm config vet`.

## Goals / Non-Goals

**Goals:**

- Make warning handler routing logic testable and tested (warn/debug/suppress paths)
- Comply with the discovery-owned-filter spec by logging unavailable API groups
- Improve config init UX with a dependency resolution hint

**Non-Goals:**

- Adding a public `SetLogger()` API to the output package (too invasive for this scope)
- Automating `cue mod tidy` in config init (tracked separately in TODO.md)
- Adding mock-based unit tests for `ServerPreferredResources()` call selection (low risk, deferred)

## Decisions

### Decision 1: Inject warningLogger interface into opmWarningHandler

**Choice**: Define an unexported `warningLogger` interface in `warnings.go` with `Warn()` and `Debug()` methods. Inject it into `opmWarningHandler` as a field. Provide a default `outputWarningLogger` struct that delegates to `output.Warn`/`output.Debug`.

**Rationale**:

- Keeps the interface unexported — it's a test seam, not a public API
- Follows the Go pattern of "accept interfaces, return structs" (AGENTS.md)
- Zero behavioral change in production — the default adapter calls the same functions
- The test table's existing `wantLevel` field can finally be asserted against a mock

**Alternatives considered**:

- Export a `SetLogger()` on the output package — too broad a change for this scope
- Capture stderr in tests — fragile, timing-dependent, couples to log format
- Accept the gap — the routing logic is simple, but untested code rots

### Decision 2: Log unavailable groups with output.Warn

**Choice**: Add a single `output.Warn("some API groups unavailable during discovery, results may be incomplete", "err", err)` after the `IsGroupDiscoveryFailedError` guard in `discoverAPIResources()`.

**Rationale**:

- The `discovery.ErrGroupDiscoveryFailed` error embeds which groups failed, so passing `err` gives useful detail in verbose mode
- Matches the spec: "log a warning about the unavailable groups"
- Single line change, zero risk

**Alternatives considered**:

- Log at Debug level — spec says "log a warning", not debug
- Parse the error to list individual groups — over-engineering for a rare scenario

### Decision 3: Minimal config init hint

**Choice**: Add `output.Println("Next: run 'cue mod tidy' in " + paths.HomeDir + " to resolve dependencies")` before the existing `Validate with: opm config vet` line.

**Rationale**:

- The generated config imports `opmodel.dev/providers@v0` which requires resolution
- Without this hint, `opm config vet` will fail on a fresh install with an opaque CUE error
- A hint is the simplest intervention; automating the step is tracked in TODO.md

## Risks / Trade-offs

**[Risk] warningLogger interface adds indirection**
→ Mitigation: Unexported, single-use, two methods. The cognitive overhead is minimal and confined to one file.

**[Trade-off] Discovery warning may be noisy on clusters with known unavailable groups**
→ Accepted: Users can set `log.kubernetes.apiWarnings: "suppress"` if needed. The warning only fires during discovery, not per-request.

**[Trade-off] Config init hint assumes user knows `cue mod tidy`**
→ Accepted: The CUE toolchain is a prerequisite for OPM. Users who don't know CUE will need to learn it regardless.
