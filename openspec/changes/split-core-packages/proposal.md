## Why

`internal/core` has grown into a grab-bag package that mixes pure domain types, CUE-heavy extraction logic, matching algorithms, and execution logic — all under a package doc that incorrectly claims it has no CUE dependency. This makes it hard to understand ownership, navigate the code, and reason about dependencies. Splitting it into focused subpackages mirrors the structure already established in the CUE catalog (`catalog/v0/core`) and aligns with the existing phase-based decomposition (`loader`, `builder`, `pipeline`).

## What Changes

- `internal/core` is split into five focused subpackages under `internal/core/`
- `internal/transformer` (warnings only) is absorbed into `internal/core/transformer`
- `internal/core/errors.go` is moved into `internal/errors/` alongside existing CLI errors, split across two files
- All consumers of `internal/core` update their import paths to the appropriate subpackage
- No public APIs, CLI behavior, or CUE semantics change — pure internal refactor

## Capabilities

### New Capabilities

- `core-component`: `Component`, `ComponentMetadata`, `ExtractComponents` — own subpackage mirroring `component.cue`
- `core-module`: `Module`, `ModuleMetadata` — own subpackage mirroring `module.cue`
- `core-modulerelease`: `ModuleRelease`, `ReleaseMetadata`, CUE validation helpers — own subpackage mirroring `module_release.cue`
- `core-transformer`: `Transformer`, `TransformerMetadata`, `TransformerContext`, `TransformerMatchPlan`, `Execute`, `CollectWarnings` — own subpackage mirroring `transformer.cue`
- `core-provider`: `Provider`, `ProviderMetadata`, `Match` — own subpackage mirroring `provider.cue`
- `errors-domain`: `TransformError`, `ValidationError` consolidated into `internal/errors/domain.go`

### Modified Capabilities

<!-- No spec-level behavior changes. This is a pure internal restructure. -->

## Impact

- **`internal/core`**: Retains only `Resource`, label constants, weight constants
- **`internal/errors`**: Gains `TransformError` and `ValidationError` in new `domain.go`; existing `errors.go` split into `errors.go` + `sentinel.go`
- **`internal/transformer`**: Deleted; absorbed into `internal/core/transformer`
- **`internal/loader`, `internal/builder`, `internal/pipeline`, `internal/cmdutil`**: Import paths updated, no logic changes
- **`internal/kubernetes`, `internal/inventory`, `internal/output`**: Minimal or no changes (only use `Resource` and label constants which stay in `internal/core`)
- SemVer: PATCH — no external API or behavior changes
