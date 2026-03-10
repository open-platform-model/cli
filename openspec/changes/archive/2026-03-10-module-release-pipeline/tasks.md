## 1. Release Type Refactor

- [x] 1.1 Update `pkg/modulerelease` to expose `RawCUE`, `DataComponents`, `Config`, and `Values` on `ModuleRelease`
- [x] 1.2 Update `pkg/bundlerelease` to expose `RawCUE`, `Releases`, `Config`, and `Values` on `BundleRelease`
- [x] 1.3 Update constructors and receiver methods in release packages to match the new field layout and remove obsolete schema/data access patterns
- [x] 1.4 Update existing release-type unit tests to reflect the new parse-stage and process-stage contracts

## 2. Internal Release Parsing

- [x] 2.1 Add a new internal package for parse-only release-file handling and define the `GetReleaseFile` return type
- [x] 2.2 Implement `GetReleaseFile` for `ModuleRelease` files with best-effort metadata, module, raw CUE, and config extraction without validation
- [x] 2.3 Implement `GetReleaseFile` for `BundleRelease` files with best-effort metadata, bundle, raw CUE, and config extraction without validation
- [x] 2.4 Add tests covering module release parsing, bundle release parsing, unresolved `#module` / `#bundle`, and unknown release kinds

## 3. Release Processing API

- [x] 3.1 Create a new public `pkg/releaseprocess` package and add `ValidateConfig(schema cue.Value, values []cue.Value)`
- [x] 3.2 Implement value unification and structured config-error handling for conflicting value inputs and schema mismatches
- [x] 3.3 Implement `ProcessModuleRelease` to validate values, fill `RawCUE`, derive concrete components, finalize `DataComponents`, compute a match plan, and call the engine renderer
- [x] 3.4 Implement a gate-validating `ProcessBundleRelease` stub that stores validated values and returns a not-yet-implemented error
- [x] 3.5 Add unit tests for `ValidateConfig`, successful module processing, validation failure short-circuiting, and bundle-processing stub behavior

## 4. Go Match Plan

- [x] 4.1 Create a new public `pkg/match` package and move or redefine `MatchPlan`, `MatchResult`, matched-pair helpers, and warning helpers there
- [x] 4.2 Implement `match.Match` to reproduce the logic in `catalog/v1alpha1/core/matcher/matcher.cue`
- [x] 4.3 Add deterministic sorting for matched pairs, non-matched pairs, unmatched components, and unhandled trait warnings
- [x] 4.4 Add matcher parity tests covering missing labels/resources/traits, unmatched components, unhandled traits, and deterministic output ordering

## 5. Engine Execution Refactor

- [x] 5.1 Refactor `pkg/engine.ModuleRenderer` so it accepts a precomputed match plan instead of constructing one internally
- [x] 5.2 Remove or replace CUE matcher-definition dependencies from `pkg/engine` while preserving transform execution semantics
- [x] 5.3 Update `pkg/engine.BundleRenderer` to render processed module releases using the refactored module renderer contract
- [x] 5.4 Update engine tests for execute-only rendering, unmatched-component errors, output decoding, and fail-slow behavior

## 6. CLI Orchestration Migration

- [x] 6.1 Refactor `internal/cmdutil/render.go` to use parse -> process -> execute for release-file rendering
- [x] 6.2 Refactor synthesized module-release rendering paths to use the new processing APIs instead of loader-owned validation and matching
- [x] 6.3 Preserve existing namespace override, `--module` injection, and values-file resolution behavior in the new orchestration flow
- [x] 6.4 Update command-level tests or integration coverage for release-file and synthesized-module render flows

## 7. Legacy Loader Cleanup

- [x] 7.1 Reduce or remove legacy `pkg/loader` functions whose validation or matching responsibilities have moved into `pkg/releaseprocess` and `pkg/match`
- [x] 7.2 Update provider-loading and release-loading call sites to use the new packages consistently
- [x] 7.3 Remove dead matcher-loading code paths and update related documentation/comments to describe the new pipeline

## 8. Validation Gates

- [x] 8.1 Run targeted unit tests for release parsing, release processing, matcher, and engine packages
- [x] 8.2 Run `task fmt` and fix any formatting issues introduced by the refactor
- [x] 8.3 Run `task lint` and address linter failures
- [x] 8.4 Run `task test` and resolve any regressions in module, release, engine, and command behavior
