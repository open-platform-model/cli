## 1. Foundation — pkg/ Type Packages

- [x] 1.1 Create `pkg/core/resource.go`: Resource struct with `cue.Value`, provenance fields (`Release`, `Component`, `Transformer`), accessor methods (`Kind()`, `Name()`, `Namespace()`, `APIVersion()`, `GVK()`, `Labels()`, `Annotations()`)
- [x] 1.2 Create `pkg/core/convert.go`: Conversion methods on Resource — `MarshalJSON()`, `MarshalYAML()`, `ToUnstructured()`, `ToMap()`
- [x] 1.3 Create `pkg/core/labels.go`: Move all label constants from `internal/core/labels.go` (identical values)
- [x] 1.4 Create `pkg/core/weights.go`: Move `GetWeight()` and GVK weight map from `internal/core/weights.go` (identical values)
- [x] 1.5 Create `pkg/module/module.go`: Promote from `experiments/factory/pkg/module/module.go`; remove dead `pkgName` field (DEBT.md #4)
- [x] 1.6 Create `pkg/modulerelease/release.go`: Promote from factory; make `schema` and `dataComponents` unexported with typed accessors `MatchComponents()` and `ExecuteComponents()` (DEBT.md #8); fix value/pointer embedding inconsistency (DEBT.md #12)
- [x] 1.7 Create `pkg/bundle/bundle.go`: Promote from `experiments/factory/pkg/bundle/bundle.go` as-is
- [x] 1.8 Create `pkg/bundlerelease/release.go`: Promote from factory; remove unused `Schema` field (DEBT.md #14)
- [x] 1.9 Create `pkg/provider/provider.go`: Promote from factory — thin CUE wrapper, no `Match()` method
- [x] 1.10 Write unit tests for `pkg/core/` — test all accessor methods and conversion methods with a test CUE value representing a K8s Deployment

## 2. Error Types — pkg/errors/

- [x] 2.1 Move `internal/errors/errors.go` to `pkg/errors/errors.go`: `DetailError`, `ExitError`, exit code constants, helper functions — no behavior changes
- [x] 2.2 Move `internal/errors/domain.go` to `pkg/errors/domain.go`: `TransformError`, `ValidationError`, `ValuesValidationError`, `FieldError`, `ConflictError` — no behavior changes
- [x] 2.3 Move `internal/errors/sentinel.go` to `pkg/errors/sentinel.go`: `ErrValidation`, `ErrConnectivity`, `ErrPermission`, `ErrNotFound` — no behavior changes
- [x] 2.4 Add `ConfigError` type to `pkg/errors/`: carry `Context`, `Name`, `RawError`; implement `Error()`, `Unwrap()`, `FieldErrors() []FieldError` (merge factory `ConfigError` with CLI error parsing from `builder/validation.go`)
- [x] 2.5 Move `internal/errors/errors_test.go` to `pkg/errors/errors_test.go` — update import paths, ensure all tests pass
- [x] 2.6 Verify: `go test ./pkg/errors/...` passes

## 3. Loader — pkg/loader/

- [x] 3.1 Create `pkg/loader/module_release.go`: Promote from factory — `LoadReleasePackage()`, `DetectReleaseKind()`, `LoadModuleReleaseFromValue()`; fix: use `os.Stat()` + `IsDir()` in `resolveReleaseFile()` instead of extension check (DEBT.md #10); remove dead `LoadRelease()` function (DEBT.md #9)
- [x] 3.2 Create `pkg/loader/finalize.go`: Extract `finalizeValue()` from factory `module_release.go` — `Syntax(cue.Final()) + BuildExpr` with clear error for non-expr result
- [x] 3.3 Create `pkg/loader/bundle_release.go`: Promote from factory — `LoadBundleReleaseFromValue()`, `extractBundleReleases()`; unify duplicated extract pattern into single `extractReleaseMetadata()` (DEBT.md #13)
- [x] 3.4 Create `pkg/loader/provider.go`: Promote from factory — `LoadProvider()` with auto-selection for single provider
- [x] 3.5 Create `pkg/loader/validate.go`: Merge factory `validateConfig()` with CLI error parsing — return `*errors.ConfigError` that supports `FieldErrors()` for structured output; integrate CUE `errors.Errors()` walking with `FieldError` construction from `builder/validation.go`
- [x] 3.6 Write unit tests for `pkg/loader/validate.go` — test Module Gate with valid values, type mismatch, missing required field; verify `FieldErrors()` returns structured output
- [x] 3.7 Write unit tests for `pkg/loader/module_release.go` — test `LoadReleasePackage()` with valid module, `DetectReleaseKind()`, `resolveReleaseFile()` with directory vs file vs nonexistent path
- [x] 3.8 Write unit tests for `pkg/loader/provider.go` — test named loading and auto-selection
- [x] 3.9 Verify: `go test ./pkg/loader/...` passes

## 4. Engine — pkg/engine/

- [ ] 4.1 Create `pkg/engine/matchplan.go`: Promote from factory — `MatchPlan`, `MatchResult`, `MatchedPair`, `buildMatchPlan()`, `sortMatchedPairs()`; fix: sort keys in `Warnings()` for deterministic output (DEBT.md #3)
- [ ] 4.2 Create `pkg/engine/execute.go`: Promote from factory — `executeTransforms()`, `executePair()`, `injectContext()`, `collectResourceList()`, `collectResourceMap()`; fix: propagate metadata decode errors instead of silent discard (DEBT.md #1) — log at WARN level or return error in strict mode; fix: define explicit output contract for `isSingleResource` (DEBT.md #6)
- [ ] 4.3 Create `pkg/engine/module_renderer.go`: Promote from factory — `ModuleRenderer`, `NewModuleRenderer()`, `Render()`, `RenderResult`, `UnmatchedComponentsError`; use `errors.Join` (already fixed in factory, DEBT.md #2)
- [ ] 4.4 Create `pkg/engine/bundle_renderer.go`: Promote from factory — `BundleRenderer`, `BundleRenderResult`; fix: fail-slow at bundle level — collect all release errors instead of stopping on first (DEBT.md #5)
- [ ] 4.5 Create `pkg/engine/errors.go`: Engine-specific error types — `UnmatchedComponentsError` with structured diagnostics, integrate with `pkg/errors.TransformError` wrapping
- [ ] 4.6 Write unit tests for `pkg/engine/matchplan.go` — test `MatchedPairs()` sorting, `Warnings()` deterministic output
- [ ] 4.7 Write unit tests for `pkg/engine/execute.go` — test three output forms (list, single resource, map), metadata decode error propagation
- [ ] 4.8 Verify: `go test ./pkg/engine/...` passes

## 5. Adapt internal/output/

- [ ] 5.1 Update `internal/output/manifest.go`: Replace `ResourceInfo` interface — use `*core.Resource` methods directly (`MarshalYAML()` for YAML output, `Kind()`, `Name()`, `Namespace()` for display); update compile-time assertions; replace `core.GetWeight()` import to `pkg/core`
- [ ] 5.2 Update `internal/output/split.go` if it references old core types
- [ ] 5.3 Verify: `go build ./internal/output/...` compiles

## 6. Adapt internal/inventory/

- [ ] 6.1 Update `internal/inventory/entry.go`: Change `NewEntryFromResource()` to accept `*unstructured.Unstructured` instead of `*core.Resource`
- [ ] 6.2 Update `internal/inventory/digest.go`: Change `ComputeManifestDigest()` to accept `[]*unstructured.Unstructured` instead of `[]*core.Resource`
- [ ] 6.3 Update `internal/inventory/crud.go`, `secret.go`, `list.go`, `stale.go`: Change label constant imports from `internal/core` to `pkg/core`
- [ ] 6.4 Update `internal/inventory/digest_test.go`, `types_test.go`: Rewrite struct literals to use `*unstructured.Unstructured` instead of `*core.Resource`
- [ ] 6.5 Verify: `go test ./internal/inventory/...` passes

## 7. Adapt internal/kubernetes/

- [ ] 7.1 Update `internal/kubernetes/apply.go`: Change signatures to accept `[]*unstructured.Unstructured` and simple strings (name, namespace) instead of `[]*core.Resource` and `modulerelease.ReleaseMetadata`
- [ ] 7.2 Update `internal/kubernetes/diff.go`: Same signature changes as apply.go
- [ ] 7.3 Update `internal/kubernetes/delete.go`: Change `GetWeight` import to `pkg/core`
- [ ] 7.4 Update `internal/kubernetes/diff_integration_test.go`: Rewrite struct literals
- [ ] 7.5 Verify: `go build ./internal/kubernetes/...` compiles (integration tests need cluster)

## 8. Adapt internal/cmdutil/ — The Integration Point

- [ ] 8.1 Rewrite `internal/cmdutil/render.go`: Replace `pipeline.NewPipeline().Render()` with new orchestration — call `loader.LoadReleasePackage()`, `loader.DetectReleaseKind()`, `loader.LoadModuleReleaseFromValue()`, `engine.NewModuleRenderer().Render()`; convert `[]*core.Resource` to `[]*unstructured.Unstructured` at this layer; keep values file resolution logic
- [ ] 8.2 Rewrite `internal/cmdutil/output.go`: Replace `TransformerMatchPlan` references with `engine.MatchPlan`; replace `ComponentSummary` construction (derive from MatchPlan or CUE iteration); update `UnmatchedComponentError` handling to use `engine.UnmatchedComponentsError`; replace `TransformerRequirements` interface usage with `engine.MatchResult` fields
- [ ] 8.3 Update `internal/cmdutil/flags.go`: Update import paths if needed
- [ ] 8.4 Update `internal/cmdutil/inventory.go`: Update import paths for label constants
- [ ] 8.5 Update `internal/cmdutil/k8s.go`: Update import paths if needed
- [ ] 8.6 Rewrite `internal/cmdutil/render_test.go`: Full rewrite of struct literals — use new types from `pkg/`
- [ ] 8.7 Rewrite `internal/cmdutil/output_test.go`: Update mock types and struct literals for new engine types
- [ ] 8.8 Verify: `go test ./internal/cmdutil/...` passes

## 9. Adapt internal/cmd/mod/ — Commands

- [ ] 9.1 Update `internal/cmd/mod/apply.go`: Adapt to new RenderResult; convert resources before K8s calls; update metadata field accesses (`result.Release.Name`, `.UUID`, etc.)
- [ ] 9.2 Update `internal/cmd/mod/build.go`: Adapt to new RenderResult; use `r.MarshalYAML()` for output
- [ ] 9.3 Update `internal/cmd/mod/diff.go`: Adapt to new RenderResult; convert resources before K8s calls
- [ ] 9.4 Update `internal/cmd/mod/vet.go`: Update method calls on Resource
- [ ] 9.5 Update `internal/cmd/mod/delete.go`: Update import paths for label constants
- [ ] 9.6 Update `internal/cmd/mod/status.go`: Update import paths
- [ ] 9.7 Update `internal/cmd/mod/list.go`: Update import paths
- [ ] 9.8 Update `internal/cmd/mod/events.go`: Update import paths if needed
- [ ] 9.9 Update `internal/cmd/mod/tree.go`: Update import paths if needed
- [ ] 9.10 Rewrite `internal/cmd/mod/verbose_output_test.go`: Full rewrite of struct literals for new types
- [ ] 9.11 Update remaining test files in `internal/cmd/mod/`: adapt to new types where needed
- [ ] 9.12 Verify: `go test ./internal/cmd/mod/...` passes

## 10. Cleanup — Remove Old Packages

- [ ] 10.1 Delete `internal/builder/` (entire directory)
- [ ] 10.2 Delete `internal/pipeline/` (entire directory)
- [ ] 10.3 Delete `internal/loader/` (entire directory)
- [ ] 10.4 Delete `internal/core/component/` (entire directory)
- [ ] 10.5 Delete `internal/core/transformer/` (entire directory)
- [ ] 10.6 Delete `internal/core/provider/` (entire directory)
- [ ] 10.7 Delete `internal/core/module/` (entire directory)
- [ ] 10.8 Delete `internal/core/modulerelease/` (entire directory)
- [ ] 10.9 Delete `internal/core/resource.go`
- [ ] 10.10 Delete `internal/core/labels.go` (moved to `pkg/core/`)
- [ ] 10.11 Delete `internal/core/weights.go` (moved to `pkg/core/`)
- [ ] 10.12 Delete `internal/errors/` (moved to `pkg/errors/`)
- [ ] 10.13 Run `go mod tidy`

## 11. Validation Gates

- [ ] 11.1 Run `task fmt` — all Go files formatted
- [ ] 11.2 Run `task vet` — go vet passes
- [ ] 11.3 Run `task lint` — golangci-lint passes
- [ ] 11.4 Run `task test:unit` — all unit tests pass
- [ ] 11.5 Run `task test:e2e` — all e2e tests pass
- [ ] 11.6 Run `task cluster:create` then `task test:integration` — all integration tests pass
- [ ] 11.7 Run `task build` — binary compiles successfully
- [ ] 11.8 Smoke test: `./bin/opm mod build` on an example module produces identical YAML output as before migration
