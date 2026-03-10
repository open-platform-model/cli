## Why

The current ModuleRelease and BundleRelease flow mixes file loading, values resolution, validation, CUE evaluation, component extraction, transformer matching, and rendering across `pkg/loader`, `pkg/engine`, and `internal/cmdutil`. This makes the release pipeline hard to reason about, blocks clean BundleRelease support, and keeps core matching logic hidden in CUE rather than in explicit Go code.

## What Changes

- Introduce an internal parse-only `GetReleaseFile` method that reads an absolute `release.cue` path, detects whether it contains `#ModuleRelease` or `#BundleRelease`, and returns a barebones release object without validating values
- Refactor `modulerelease.ModuleRelease` and `bundlerelease.BundleRelease` to carry `RawCUE`, extracted `Config`, concrete `Values`, and finalized component/release data needed by later processing stages
- Add a public release-processing API that separates config validation and release processing from file loading, including `ProcessModuleRelease` and a gate-only stub for `ProcessBundleRelease`
- Reimplement transformer matching in Go via a new public `match.Match` API that mirrors the current behavior in `catalog/v1alpha1/core/matcher/matcher.cue`
- Refactor the render pipeline so matching happens before engine execution, leaving `pkg/engine` responsible for executing matched transformers rather than building the match plan itself
- Rewire CLI rendering orchestration to use the new parse -> process -> execute pipeline for release files and synthesized module releases

## Capabilities

### New Capabilities
- `module-release-processing`: Parse, validate, concretize, match, and render `ModuleRelease` instances through explicit Go pipeline stages
- `bundle-release-processing`: Parse and validate `BundleRelease` instances through the same pipeline shape, with bundle rendering prepared behind a public processing API

### Modified Capabilities
- `release-file-loading`: Change release file loading from eager validation to parse-only best-effort extraction that tolerates unresolved `#module` or `#bundle` references
- `transformer-match-plan-execute`: Move match-plan construction from CUE evaluation into Go while preserving current matching semantics and execution behavior
- `engine-rendering`: Change engine responsibilities so it executes a supplied match plan rather than creating one internally

## Impact

- Affected packages: `internal/` release file parsing, `pkg/modulerelease`, `pkg/bundlerelease`, `pkg/loader`, `pkg/engine`, `pkg/provider`, `internal/cmdutil`
- New public packages are expected for release processing and Go-side matching
- Existing CLI render flows for module directories, release files, and later bundle releases will be rewired to use the new APIs
- This is a MINOR change: it introduces new internal/public pipeline APIs and refactors behavior without intentionally removing user-facing commands
