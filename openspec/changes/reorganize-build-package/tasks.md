## 1. Phase 1: Module Package

- [x] 1.1 Create `internal/build/module/` directory
- [x] 1.2 Create `module/loader.go` with resolveModulePath and extractModuleMetadata functions (from pipeline.go:200-217, 221-272)
- [x] 1.3 Create `module/inspector.go` with extractMetadataFromAST and extractFieldsFromMetadataStruct (from module.go:22-79)
- [x] 1.4 Create `module/types.go` with ModuleInspection and moduleMetadataPreview types
- [x] 1.5 Create `internal/build/component.go` at root with LoadedComponent type (shared across packages)
- [x] 1.6 Create `module/loader_test.go` with tests for module path resolution and metadata extraction
- [x] 1.7 Update `pipeline.go` to import module package and use module.ResolvePath, module.ExtractMetadata
- [x] 1.8 Update `release_builder.go` to import module package and use module.ExtractMetadataFromAST
- [x] 1.9 Delete `internal/build/module.go`
- [x] 1.10 Run `go test ./internal/build/module -v` to verify module tests pass
- [x] 1.11 Run `task test` to verify all tests still pass

## 2. Phase 2: Release Package

- [x] 2.1 Create `internal/build/release/` directory
- [x] 2.2 Create `release/builder.go` with ReleaseBuilder type, NewReleaseBuilder, Build, detectPackageName, InspectModule, loadValuesFile (from release_builder.go:24-39, 89-257, 259-277, 279-317, 428-449)
- [x] 2.3 Create `release/overlay.go` with opmNamespaceUUID constant and generateOverlayAST method (from release_builder.go:22, 332-425)
- [x] 2.4 Create `release/validation.go` with validateValuesAgainstConfig, validateFieldsRecursive, collectAllCUEErrors, findSourcePosition, pathRewrittenError, rewriteErrorPath (from errors.go:476-561, 383-385, 440-457, 393-430)
- [x] 2.5 Create `release/metadata.go` with extractReleaseMetadata, extractMetadataFallback, extractComponentsFromDefinition, extractComponent, extractAnnotations (from release_builder.go:560-648, 452-474, 476-536, 539-553)
- [x] 2.6 Create `release/types.go` with ReleaseOptions, BuiltRelease, ReleaseMetadata types (note: ModuleReleaseMetadata stays in root types.go as public API)
- [x] 2.7 Move `release_builder_test.go` to `release/builder_test.go`
- [x] 2.8 Move `release_builder_annotations_test.go` to `release/`
- [x] 2.9 Move `release_builder_ast_test.go` to `release/`
- [x] 2.10 Move `release_builder_identity_test.go` to `release/`
- [ ] 2.11 Move `values_resolution_test.go` to `release/` — kept in root build (tests pipeline integration via adapters)
- [x] 2.12 Update `errors.go` to remove validation functions (lines 476-561) but keep ReleaseValidationError type
- [x] 2.13 Update `pipeline.go` to import release package and use release.NewBuilder, release.Build
- [x] 2.14 Update `executor.go` to import release package and use release.BuiltRelease
- [x] 2.15 Delete `internal/build/release_builder.go`
- [x] 2.16 Run `go test ./internal/build/release -v` to verify release tests pass
- [x] 2.17 Run `task test` to verify all tests still pass

## 3. Phase 3: Transform Package

- [x] 3.1 Create `internal/build/transform/` directory
- [x] 3.2 Create `internal/build/transformer_adapter.go` at root with type aliases for transform package types (LoadedProvider, LoadedTransformer, ProviderLoader, Matcher, Executor, TransformerContext, TransformError, TransformerSummary, MatchResult)
- [x] 3.3 Create `transform/provider.go` with ProviderLoader type, NewProviderLoader, Load, extractTransformer, extractLabelsField, extractMapKeys, ToSummaries, BuildFQN (from provider.go)
- [x] 3.4 Create `transform/matcher.go` with Matcher type, NewMatcher, Match, evaluateMatch, buildReason, ToMatchPlan (from matcher.go)
- [x] 3.5 Create `transform/executor.go` with Executor type, NewExecutor, ExecuteWithTransformers, executeJob, isSingleResource, decodeResource (from executor.go)
- [x] 3.6 Create `transform/context.go` with TransformerContext, TransformerModuleReleaseMetadata, TransformerComponentMetadata, NewTransformerContext, ToMap (from context.go)
- [x] 3.7 Create `transform/types.go` with Job, JobResult, ExecuteResult, MatchResult, MatchDetail types (internal to transform)
- [x] 3.8 Move `matcher_test.go` to `transform/`
- [x] 3.9 Move `executor_test.go` to `transform/`
- [x] 3.10 Move `context_test.go` and `context_annotations_test.go` to `transform/`
- [x] 3.11 Update `pipeline.go` to import transform package and use transform.NewProviderLoader, transform.NewMatcher, transform.NewExecutor
- [x] 3.12 Update `types.go`: MatchPlan and TransformerMatch kept as concrete types; pipeline converts transform.MatchPlan to build.MatchPlan
- [x] 3.13 Delete `internal/build/provider.go`
- [x] 3.14 Delete `internal/build/matcher.go`
- [x] 3.15 Delete `internal/build/executor.go`
- [x] 3.16 Delete `internal/build/context.go`
- [x] 3.17 Run `go test ./internal/build/transform -v` to verify transform tests pass
- [x] 3.18 Run `task test` to verify all tests still pass

## 4. Phase 4: Orchestration Package

> **SKIPPED**: Creating a separate `orchestration` package would introduce import cycles.
> `pipeline.go` stays in the root `build` package with full orchestration logic,
> calling into `module`, `release`, and `transform` subpackages directly.

- [ ] 4.1 Create `internal/build/orchestration/` directory — SKIPPED (import cycle)
- [ ] 4.2 Create `orchestration/pipeline.go` — SKIPPED
- [ ] 4.3 Create `orchestration/helpers.go` — SKIPPED
- [ ] 4.4 Note: releaseToModuleReleaseMetadata takes moduleName parameter — implemented in pipeline.go
- [x] 4.5 CRITICAL: Preserve deterministic 5-key resource sorting (weight → group → kind → namespace → name) — preserved in pipeline.go
- [ ] 4.6 Move `pipeline_test.go` to `orchestration/` — SKIPPED (stays in root build)
- [ ] 4.7 Rewrite root `pipeline.go` as thin facade — SKIPPED (pipeline.go kept as is)
- [ ] 4.8 Run `go test ./internal/build/orchestration -v` — SKIPPED
- [x] 4.9 Run `task test` to verify all tests still pass

## 5. Phase 5: Cleanup and Documentation

- [x] 5.1 Verify `internal/build/module.go` is deleted
- [x] 5.2 Verify `internal/build/release_builder.go` is deleted
- [x] 5.3 Verify `internal/build/provider.go` is deleted
- [x] 5.4 Verify `internal/build/matcher.go` is deleted
- [x] 5.5 Verify `internal/build/executor.go` is deleted
- [x] 5.6 Verify `internal/build/context.go` is deleted
- [x] 5.7 Verify old test files are deleted from root (moved to subpackages)
- [ ] 5.8 Create `internal/build/README.md` documenting new package structure
- [x] 5.9 Update `AGENTS.md` project structure section (lines 40-55) with new build/ organization
- [x] 5.10 Run `task check` (fmt + vet + test)
- [x] 5.11 Run `task build` to verify binary builds successfully
- [ ] 5.12 Run manual end-to-end test: `./bin/opm mod apply ./tests/fixtures/simple-module`
- [ ] 5.13 Run `task test:coverage` to verify test coverage maintained
- [ ] 5.14 Commit changes: `refactor(build): reorganize into focused subpackages`

## 6. Verification and Validation

- [x] 6.1 Verify public API unchanged: commands still import `build.NewPipeline()` successfully
- [x] 6.2 Verify all subpackage imports use correct paths (github.com/opmodel/cli/internal/build/module, etc.)
- [x] 6.3 Verify no circular import dependencies between subpackages
- [x] 6.4 Verify shared types (LoadedComponent, LoadedTransformer) accessible from all subpackages
- [x] 6.5 Verify testdata/ directory still accessible from all test files
- [x] 6.6 Run regression test suite on real modules if available
- [ ] 6.7 Verify no duplicate code exists between root and subpackages
- [ ] 6.8 Review file sizes: ensure no file exceeds 300 lines
