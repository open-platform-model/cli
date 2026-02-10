## 1. ReleaseBuilder: Overlay-based metadata (`internal/build/release_builder.go`)

> **Note:** Original design targeted full `#ModuleRelease` overlay. Actual implementation uses a **hybrid overlay** approach — overlay computes release metadata (`#opmReleaseMeta`), while `FillPath(#config, values)` makes components concrete. This is due to CUE SDK limitation: `close()` + pattern constraints in `#ModuleRelease` panics.

- [x] 1.1 Add `modulePath string` parameter to `ReleaseBuilder.Build()` (needed for overlay file path). Update `NewReleaseBuilder` to accept module path or pass it via `ReleaseOptions`.
  > **Done.** `Build(modulePath string, opts ReleaseOptions, valuesFiles []string)` — modulePath is a direct parameter.
- [x] 1.2 Add `import "cuelang.org/go/cue/format"` and `"cuelang.org/go/cue/load"` to release_builder.go imports.
  > **Partially done.** `"cuelang.org/go/cue/load"` added. `format` not needed (no serialization in overlay approach).
- [x] 1.3 Implement `generateOverlayCUE(opts ReleaseOptions) []byte` that generates the virtual overlay content.
  > **Done with deviation.** Method is `generateOverlayCUE(pkgName string, opts ReleaseOptions) []byte`. Generates `#opmReleaseMeta` definition (not `_opmRelease: core.#ModuleRelease`). Does NOT import `opmodel.dev/core` — instead computes release identity via `uuid.SHA1` directly. File named `opm_release_overlay.cue` (not `_opm_release.cue` — files starting with `_` are excluded by CUE loader).
- [x] 1.4 Implement `buildWithOverlay(modulePath string, overlay []byte) (cue.Value, error)`.
  > **Done inline.** Logic is inline in `Build()` rather than a separate method. Uses `load.Config{Dir: modulePath, Overlay: map[string]load.Source{...}}`, `load.Instances`, and `cueCtx.BuildInstance`.
- [x] 1.5 Rewrite `Build()` to use the overlay approach.
  > **Done with deviation.** Build uses overlay for metadata but KEEPS `FillPath(#config, values)` for component concreteness (hybrid approach). Does NOT remove FillPath — CUE SDK limitation prevents full `#ModuleRelease` evaluation.
- [x] 1.6 Update `extractComponents()` to work on release components.
  > **Done with deviation.** Method is `extractComponentsFromDefinition(concreteModule)` — still reads from `#components` definition (not `_opmRelease.components`), since we use hybrid approach.
- [x] 1.7 Update `extractMetadata()` to read from overlay-computed fields.
  > **Done.** `extractReleaseMetadata()` reads from `#opmReleaseMeta` for fqn, version, identity, and labels. Falls back to `extractMetadataFallback()` when overlay is not available.
- [x] 1.8 Remove the `LabelReleaseID` constant and the manual label extraction in `extractMetadata()`.
  > **Done.** Removed `LabelReleaseID` constant (duplicate of `kubernetes.LabelReleaseID`). Removed manual `Labels[LabelReleaseID]` → `ReleaseIdentity` extraction from both `extractMetadataFallback()` and `extractMetadataFromModule()`. Release identity is now only computed by the CUE overlay in `Build()`. Updated test `TestReleaseBuilder_ExtractMetadata_ReleaseIdentityFromLabels` → `TestReleaseBuilder_ExtractMetadata_LegacyPathNoReleaseIdentity` to assert empty `ReleaseIdentity` in legacy path.

## 2. ReleaseBuilder: Update callers (`internal/build/pipeline.go`)

- [x] 2.1 Update `pipeline.Render()` to pass `module.Path` to `ReleaseBuilder.Build()`.
  > **Done differently.** Pipeline now calls `resolveModulePath()` to get `modulePath string`, then passes it directly to `releaseBuilder.Build(modulePath, opts, valuesFiles)`. No `LoadedModule` struct involved.
- [x] 2.2 Verify `LoadedModule` exposes `Path` field.
  > **Superseded.** `LoadedModule` type was removed entirely. Pipeline uses `modulePath` string directly. `module.go` now only contains `LoadedComponent`.

## 3. Executor: Pre-execution serialization (`internal/build/executor.go`)

> **ALL SUPERSEDED.** Original plan was to serialize CUE values via `format.Node(value.Syntax())` to CUE source text, then re-compile in fresh `*cue.Context` per worker goroutine. This approach failed: `Syntax()` panics on transformer values with complex cross-package references (`adt.Vertex` "unreachable" in `cuelang.org/go@v0.15.4/internal/core/export/self.go:379`). Sequential execution was chosen instead.

- [x] ~~3.1 Add `import "cuelang.org/go/cue/format"` and `"cuelang.org/go/cue/cuecontext"` to executor.go imports.~~
  > **Superseded.** Neither import needed — executor runs sequentially.
- [x] ~~3.2 Implement `serializeValue(v cue.Value) (string, error)` helper.~~
  > **Superseded.** No serialization — sequential execution.
- [x] ~~3.3 Add `SerializedJob` struct.~~
  > **Superseded.** Kept original `Job` struct. No serialized variant needed.
- [x] ~~3.4 In `ExecuteWithTransformers()`, add a pre-execution serialization phase.~~
  > **Superseded.** No serialization phase — jobs execute sequentially in a simple loop.

## 4. Executor: Isolated CUE context per job (`internal/build/executor.go`)

> **ALL SUPERSEDED.** See Section 3 note. Sequential execution eliminates the need for isolated contexts.

- [x] ~~4.1 Update `runWorker()` to receive `SerializedJob`.~~
  > **Superseded.** `runWorker` removed. No worker pool.
- [x] ~~4.2 Rewrite `executeJob()` to accept `SerializedJob` and create fresh context.~~
  > **Superseded.** `executeJob` takes `Job` directly. Uses shared context (safe because single-threaded).
- [x] ~~4.3 Remove `cueCtx := job.Transformer.Value.Context()` line.~~
  > **Not applicable.** Line still exists (line 130) and is correct — it obtains the context for `Encode()` calls, which is safe in sequential execution.
- [x] ~~4.4 Remove `Job` struct's direct CUE value references.~~
  > **Superseded.** `Job` struct still has `*LoadedTransformer` and `*LoadedComponent` — safe in sequential execution.

## 5. ReleaseBuilder: Tests (`internal/build/release_builder_test.go`)

- [ ] 5.1 Write `TestGenerateOverlayCUE` — verify generated CUE contains correct package, import, field name, release name, and namespace.
- [ ] 5.2 Write `TestBuildWithOverlay_ValidModule` — use test fixture to verify overlay produces valid release with concrete components and computed metadata.
- [ ] 5.3 Write `TestBuildWithOverlay_InvalidValues` — provide values that violate `#config` schema and verify error contains CUE file:line:col information.
- [ ] 5.4 Write `TestBuildWithOverlay_MetadataExtraction` — verify `ReleaseMetadata` contains `fqn`, `version`, `identity`, and all standard release labels from CUE.
- [x] 5.5 Update or remove `TestReleaseBuilder_ExtractMetadata_ReleaseIdentityFromLabels` and `TestReleaseBuilder_ExtractMetadata_NoReleaseIdentity`.
  > **Done.** Existing tests updated to use `BuildFromValue()` (legacy path). They still test the fallback metadata extraction.
- [ ] 5.6 Remove `TestReleaseBuilder_Build_*` tests that test the old `FillPath(#config, values)` approach.
  > **Not applicable.** `FillPath(#config, values)` is still used in the hybrid approach. These tests remain valid.

## 6. Executor: Tests (`internal/build/executor_test.go`)

> **Serialization tests superseded.** Existing `TestExecuteJob_*` tests cover the sequential execution path.

- [x] ~~6.1 Write `TestSerializeValue`.~~
  > **Superseded.** No serialization.
- [x] ~~6.2 Write `TestSerializeValue_RoundTrip`.~~
  > **Superseded.** No serialization.
- [x] ~~6.3 Write `TestExecuteJob_IsolatedContext`.~~
  > **Superseded.** No isolated contexts — sequential execution.
- [x] ~~6.4 Write `TestExecuteWithTransformers_Parallel_NoPanic`.~~
  > **Superseded.** No parallel execution. Concurrency panic is eliminated by design (sequential).
- [ ] 6.5 Write `TestExecuteWithTransformers_DeterministicOutput` — run the same execution twice and verify the resources are identical (preserves FR-B-053).

## 7. Integration: End-to-end pipeline (`internal/build/pipeline_test.go`)

- [ ] 7.1 Write `TestPipeline_Render_MultiComponent` — render a multi-component module through the full pipeline and verify: no panic, all resources present, correct labels.
- [ ] 7.2 Write `TestPipeline_Render_OutputMatchesPreviousBehavior` — render the jellyfin module and compare output against a golden file.

## 8. Cleanup

- [ ] 8.1 Remove `internal/build/release_builder_identity_test.go` if `TestOPMNamespaceUUIDCorrect` is no longer relevant.
  > **Still relevant.** The test verifies the OPM namespace UUID constant matches the CUE overlay's hardcoded UUID string (`c1cbe76d-5687-5a47-bfe6-83b081b15413`). Keeps Go and CUE in sync. **Keep this test.**
- [x] 8.2 Review and remove unused imports in `release_builder.go`.
  > **Done.** Imports are clean: `fmt`, `os`, `path/filepath`, `cue`, `load`, `output`.
- [x] 8.3 Update `internal/build/pipeline.go` comments to reference the new architecture.
  > **Done.** Pipeline comments updated (phases 1-6, overlay references).

## 9. Validation Gates

- [ ] 9.1 Run `task fmt` — all Go files formatted
- [x] 9.2 Run `task check` — fmt + vet + test all pass
  > **Equivalent done.** `go test ./...` passes, `go vet ./...` clean, `go build ./...` clean.
- [x] 9.3 Manually test `opm mod build` against the jellyfin example module and verify YAML output is correct
  > **Done.** Jellyfin renders with the same 4 pre-existing transform errors as `main` branch. No panics. No regressions.
- [ ] 9.4 Manually test `opm mod build` against a multi-component module (blog or multi-tier-module) and verify no panic
- [ ] 9.5 Manually test `opm mod apply` against a live cluster to verify end-to-end behavior is preserved
