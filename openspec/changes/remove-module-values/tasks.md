## 1. Module Struct Cleanup

- [x] 1.1 Remove `Values cue.Value` field from `Module` struct in `internal/core/module/module.go`
- [x] 1.2 Remove `HasValuesCue bool` field from `Module` struct
- [x] 1.3 Remove `SkippedValuesFiles []string` field from `Module` struct
- [x] 1.4 Update `Module` struct comments to reflect v1alpha1 model

## 2. Loader Simplification

- [x] 2.1 Remove Step 8 from `LoadModule` in `internal/loader/module.go` (Pattern A values.cue loading and Pattern B inline values fallback)
- [x] 2.2 Remove references to `mod.Values`, `mod.HasValuesCue`, `mod.SkippedValuesFiles` from `LoadModule`
- [x] 2.3 Keep values*.cue file filtering in Step 3 (required for closed `#Module` definition)
- [x] 2.4 Update `LoadModule` function comments and doc string to reflect simplified behavior

## 3. Builder Values Rewrite

- [x] 3.1 Rewrite `selectValues` in `internal/builder/values.go`: when no `--values` files, discover `values.cue` from `mod.ModulePath`, load it, extract `values` field
- [x] 3.2 Remove `mod.Values` fallback from `selectValues`
- [x] 3.3 Add debug logging in `selectValues` for which values source was used (moved from pipeline)
- [x] 3.4 Remove `modCopy.Values = selectedValues` from `Build` in `internal/builder/builder.go` (line 152)
- [x] 3.5 Update builder comments and doc strings

## 4. Pipeline Simplification

- [x] 4.1 Remove values debug logging from `pipeline.prepare()` in `internal/pipeline/pipeline.go` (lines 182-217)
- [x] 4.2 Remove references to `mod.HasValuesCue` and `mod.SkippedValuesFiles` from `prepare()`
- [x] 4.3 Simplify `prepare()` to: load module, validate, resolve name/namespace

## 5. Test Fixture Updates

- [x] 5.1 Remove `internal/loader/testdata/inline-values-module/` fixture (Pattern B dead)
- [x] 5.2 Update remaining loader test fixtures if they reference `mod.Values`

## 6. Loader Tests

- [x] 6.1 Remove `TestLoadModule_InlineValues_PopulatesModValues` test
- [x] 6.2 Remove `TestLoadModule_InlineValues_NoSeparateValuesFile` test
- [x] 6.3 Remove `TestLoadModule_ApproachA_DefaultValuesLoadedSeparately` test
- [x] 6.4 Remove `TestLoadModule_ExtraValuesFilesFilteredSilently` test (or rewrite to only check filtering, not `mod.Values`)
- [x] 6.5 Update `TestLoadModule_Success` — remove `mod.Values.Exists()` assertion
- [x] 6.6 Update `TestLoadModule_PartialMetadata` — remove `mod.Values` assertions
- [x] 6.7 Update `TestLoadModule_NoValues` — remove `mod.Values` assertion (test may still verify loader handles missing values gracefully)
- [x] 6.8 Update `TestLoadModule_NoComponents` — remove `mod.Values` assertion
- [x] 6.9 Update `TestLoadModule_ApproachA_ModuleRawHasNoConcreteValues` — keep (still valid: Raw should not have `values` field)

## 7. Builder Tests

- [x] 7.1 Add test: `selectValues` discovers `values.cue` from module directory when no `--values` provided
- [x] 7.2 Add test: `selectValues` errors when no `--values` and no `values.cue` exists
- [x] 7.3 Add test: `selectValues` ignores `values.cue` when `--values` files are provided
- [x] 7.4 Update `internal/builder/values_test.go` — remove tests that reference `mod.Values`

## 8. Integration Test Updates

- [x] 8.1 Update `tests/integration/values-flow/main.go` — remove `mod.Values` assertions, keep `rel.Values` assertions

## 9. Validation Gates

- [x] 9.1 Run `task fmt` — all Go files formatted
- [x] 9.2 Run `task lint` — golangci-lint passes
- [x] 9.3 Run `task test:unit` — all unit tests pass
