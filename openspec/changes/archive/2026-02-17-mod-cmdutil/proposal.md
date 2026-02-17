## Why

The six `mod` commands (apply, build, vet, delete, diff, status) share significant boilerplate: module path resolution, OPM config retrieval, Kubernetes config resolution, render pipeline execution with error handling, K8s client creation, and verbose output formatting. Each command re-implements these patterns independently with per-command flag variables, leading to ~200 lines of duplicated orchestration code that diverges subtly over time (e.g., `apply` wraps errors differently than `build`, `diff` omits render-error checking that `vet` has). A shared `internal/cmdutil` package will centralize these patterns, making commands shorter, more consistent, and easier to extend.

## What Changes

- **New `internal/cmdutil` package** providing:
  - Flag group helpers (render flags, K8s connection flags, output flags) that attach to cobra commands and return a resolved options struct
  - Module path resolver (args → path with "." default)
  - OPM config accessor with standardized error wrapping
  - Render pipeline executor that handles config loading, option building, validation, pipeline creation, render execution, error formatting (`printValidationError`, `printRenderErrors`), warning printing, and verbose/default match output — returning `*build.RenderResult` or an `ExitError`
  - K8s client factory that resolves kubeconfig/context/namespace, creates the client, and handles connectivity errors with proper exit codes
  - Shared output helpers already partially in `internal/output` but currently with command-package-scoped callers that duplicate the call pattern
- **Move `ExitError` from `mod_build.go` to `exit.go`** where it logically belongs alongside exit code constants
- **Move shared output functions** (`writeTransformerMatches`, `writeVerboseMatchLog`, `writeBuildVerboseJSON`, `printValidationError`, `printRenderErrors`) from `mod_build.go` into `internal/cmdutil` or `internal/output` as appropriate
- **Refactor all 6 mod commands** to use `cmdutil` helpers, reducing each command to its unique logic only
- **Remove per-command flag variable blocks** (e.g., `applyValuesFlags`, `buildValuesFlags`, `vetValuesFlags`) in favor of cmdutil-managed flag groups

## Capabilities

### New Capabilities

- `cmdutil`: Shared command utility package providing flag group management, render pipeline execution, K8s client creation, and output formatting helpers for `mod` subcommands. Defines the builder API contract and reusable patterns.

### Modified Capabilities

_(No behavioral requirement changes. All existing command specs describe WHAT commands do, not the internal structure. The refactoring changes HOW commands are wired, not their user-facing behavior, flags, exit codes, or output format.)_

## Impact

- **Affected code**: `internal/cmd/mod_apply.go`, `mod_build.go`, `mod_vet.go`, `mod_diff.go`, `mod_delete.go`, `mod_status.go`, `mod_build.go` (shared helpers moved out), `exit.go` (receives `ExitError`)
- **New code**: `internal/cmdutil/` package (flags.go, render.go, k8s.go, output.go)
- **APIs**: No public API changes. Internal package boundary only.
- **Dependencies**: No new external dependencies. Uses existing cobra, build, kubernetes, output, config packages.
- **Tests**: Existing command tests must continue passing. New unit tests for cmdutil helpers.
- **SemVer**: **PATCH** — internal refactoring only, no user-facing changes.
- **Risk**: Medium — touches all mod commands simultaneously. Mitigated by existing test coverage and behavioral equivalence verification.
