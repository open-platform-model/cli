## Why

A deep codebase audit identified dead code (functions with no production callers,
struct fields never populated, no-op flag registrations), copy-paste duplication
across command implementations, and one correctness bug in `inventory/stale.go`
where a naive pluralizer produces incorrect Kubernetes GVRs (e.g. `NetworkPolicy →
"networkpolicys"`). The debt is spread enough to warrant a dedicated sweep now,
before it compounds across the several in-progress changes.

## What Changes

- **Bug fix**: Replace broken `kindToResource` in `inventory/stale.go` with the
  comprehensive implementation from `kubernetes/resource.go` (export it).
- **Dead code removal**:
  - Delete `ComputeReleaseUUID` from `core/labels.go`; remove `github.com/google/uuid` dep
  - Delete `ValidFormats` from `output/format.go`
  - Delete `ListTemplateFiles` from `templates/embed.go`
  - Move `SetLogWriter` from `output/log.go` to test-only scope
  - Move `rewriteErrorPath` from `modulerelease/validation.go` to test file
  - Delete dead no-op block (`_ = s`) in `builder/builder.go`
  - Remove `ModuleMetadata.Annotations`, `ReleaseMetadata.Annotations`,
    `ModuleMetadata.Labels` (never populated or consumed)
  - **BREAKING** (CLI surface): Remove `--wait` / `--timeout` from `mod apply`;
    remove `--wait` from `mod delete` (flags registered but had no effect)
  - Use `Info.String()` in `runVersion` instead of manual field formatting
- **DRY / extraction**:
  - Export `KindToResource` / `HeuristicPluralize` from `kubernetes/resource.go`
  - Add `cmdutil.ResolveInventory` helper (eliminates copy-paste in `delete` + `status`)
  - Extract `extractCUEStringMap` into `loader/cue_util.go` (eliminates triplication in `loader` + `builder`)
  - Fix `buildMatchReason` to call `tf.GetRequiredResources()` / `tf.GetRequiredTraits()`
  - Fix `execute.go` to call `match.Transformer.GetFQN()` (eliminates inline nil-guard)
  - Remove duplicate `boolPtr`; use `output.BoolPtr` in `kubernetes/apply.go`
  - Consolidate duplicate resource sort: `pipeline.go` calls `inventory.SortResources`
  - Standardize `"resolving kubernetes config"` error message across all 6 command files
  - Fix `apply.go` to use `cmdutil.ResolveModulePath(args)` consistently
  - Extract `styledGreenCheck` var in `output/styles.go` (used in 2 functions)
  - Remove duplicate `ResolveModulePath` test assertions from `cmdutil/flags_test.go`
- **Structural cleanup**:
  - Remove `TransformerMetadata.APIVersion` (duplicates `Transformer.APIVersion`)
  - Replace `TransformerComponentMetadata` with embedded `*component.ComponentMetadata`
  - Delete 4 orphaned doc comments in `output/manifest.go` and `output/split.go`
  - Investigate and resolve `DeleteInventory` (either wire to `mod delete` or remove)

## Capabilities

### New Capabilities

- `cmdutil-inventory-resolver`: New `cmdutil.ResolveInventory` helper that encapsulates
  inventory lookup by release ID or release name, not-found handling, and resource
  discovery — shared by `mod delete` and `mod status`.
- `loader-cue-helpers`: New `extractCUEStringMap` utility in `loader/cue_util.go`
  for extracting `map[string]string` from a CUE value field — shared by loader and builder.

### Modified Capabilities

_(none — flag removals are removing non-functional stubs; no behavioral spec changes)_

## Impact

- **Packages modified**: `internal/core` (labels, module, modulerelease, transformer),
  `internal/inventory`, `internal/kubernetes`, `internal/loader`, `internal/builder`,
  `internal/pipeline`, `internal/output`, `internal/cmdutil`, `internal/cmd/mod`,
  `internal/templates`, `internal/version`
- **Dependencies**: `github.com/google/uuid` removed from `go.mod`
- **CLI surface**: `--wait`/`--timeout` removed from `mod apply`; `--wait` removed
  from `mod delete` (these flags had no effect — not functional)
- **SemVer**: PATCH (the removed flags were non-functional no-ops)
