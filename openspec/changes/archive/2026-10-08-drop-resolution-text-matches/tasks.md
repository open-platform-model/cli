## 1. Spike: pin today's answers (tests only)

- [x] 1.1 In `internal/config/platform_hint_pin_test.go`, add rows to `TestPlatformBuildHint_Pinned` (or a sibling test on the `cuemodtest` registry) for: a directly imported dependency whose module file does not parse; an import whose package name does not match; an ambiguous import; a registry that answers 401. Record the hint each gets at head. Verify: `go test ./internal/config/ -run TestPlatformBuildHint` passes with no code change.
- [x] 1.2 Make `TestBuildPlatformModule_KeyImportDriftNamesTheEntry` assert the hint text ("must equal the module path of the catalog it imports") and the validation exit class (`errors.Is(err, oerrors.ErrValidation)`). Add a hermetic unit test that feeds `platformBuildHint` a CUE error built with `cuecontext` at a path under `#registry` and one at another path. Verify: both pass at head.
- [x] 1.3 Try to reach the library's shape-check refusal of a `#registry` entry (an entry with no `#catalog`) through `BuildPlatformModule`. Pin the hint it gets, or write under "The shape check flattens its CUE cause" in `design.md` that the branch is not reachable from a platform file. Verify: the test passes at head, or the finding is in `design.md`.
- [x] 1.4 In `internal/config/loader_test.go`, add a test that prints nothing and asserts, for a config that sets `providers`, one that sets `cacheDir` and one with a `skewPolicy` outside its enum, that `cueerrors.Errors` of the validation error holds an error whose first path selector is that field. Add a case where an unrelated invalid field holds the string `cacheDir` in its value and pin the hint it gets at head. Verify: the test passes, or the row without a path is written into `design.md` ("Config field hints").
- [x] 1.5 Confirm that `TestLoadPublishedPackage_Pinned`, `TestUnprovidedImport_RegistryFailureIsNotAbsent`, `TestProbedPackageAbsent`, `TestCompatScan*` and `TestGateCompat*` in `internal/publish` pin "absent" against `*ConnectivityError` for both unprovided-import rows and every registry-failure row; add a row only where one is missing. Verify: `go test ./internal/publish/` passes.
- [x] 1.6 `task lint` and `task test:unit` green, then commit `test(config): pin platform and config hint answers before the typed checks`

## 2. The compat walk reads the resolution kind

- [x] 2.1 Replace the body of `unprovidedImport` in `internal/publish/compat.go` as `design.md` D1 says, and rewrite its doc comment and the sentence in `loadPublishedPackage`'s comment that calls it a text match. Verify: `grep -n 'strings.Contains(err.Error()' internal/publish/compat.go` prints nothing.
- [x] 2.2 Verify with no test edited: `go test ./internal/publish/ -run 'TestLoadPublishedPackage_Pinned|TestUnprovidedImport_RegistryFailureIsNotAbsent|TestProbedPackageAbsent|TestCompatScan|TestGateCompat'` passes, every row unchanged.
- [x] 2.3 `task lint` and `task test:unit` green, then commit `refactor(publish): read an unprovided import from the library's resolution kind`

## 3. Platform build hints read types and the CUE path

- [x] 3.1 Add `cueErrorUnder` to `internal/config` (`design.md` D3) with a unit test: a path under the selector, a path elsewhere, an error with no CUE error in its chain, nil.
- [x] 3.2 Rewrite `platformBuildHint` in `internal/config/platform.go` as `design.md` D2 says; drop the `msg` variable and the `strings` import if unused; rewrite the comment above the pin case. Verify: `grep -n 'strings.Contains' internal/config/platform.go` prints nothing.
- [x] 3.3 Flip only the rows the proposal names (direct-path module file to the pin hint; package-name mismatch to the default hint; the shape-check row to the default hint, if section 1 reached it). Replace `TestPlatformBuildHint_NotFoundWithoutImportPrefix` only if its text form no longer classifies. Verify: every other row of section 1 passes unedited, and `errors.Is(err, oerrors.ErrValidation)` holds on each.
- [x] 3.4 `task lint` and `task test:unit` green, then commit `fix(config): pick the platform build hint from the error type and the CUE path`

## 4. Config hints read the failing field

- [x] 4.1 Change `removedFieldHint` in `internal/config/loader.go` to take the error and use `cueErrorUnder`, in today's order; a row section 1 found without a path keeps its match and is noted for the allowlist. Verify: `TestValidateConfigSchema_ProvidersRejected`, `TestValidateConfigSchema_CacheDirRejected` and `TestLoadConfigFile_SkewPolicyInvalidValue` pass unedited, and the "word in a value" case of 1.4 now gets the generic hint.
- [x] 4.2 `task lint` and `task test:unit` green, then commit `fix(config): pick the config hint from the failing field, not from the message`

## 5. Guard and records

- [x] 5.1 Add the guard test of `design.md` D4 with the allowlist (`internal/cuemod/tidy.go` `classify`, `internal/publish/identity.go` `conformIdentity`, `pkg/errors/grouped_errors.go` `groupCUEErrors`, `internal/workflow/render/validation.go` `printValidationError`), each with its reason. Verify: it passes, and it fails when a `strings.Contains(err.Error(), "x")` is added to a scratch non-test file (remove the file after).
- [x] 5.2 Update the comment block at the top of `internal/cuemod/connectivity.go` and the `internal/publish` and `internal/config` lines of `AGENTS.md` where they describe a text match that is gone. Verify: `grep -rn 'message-text match\|text match left' internal cmd pkg AGENTS.md` shows only the allowlisted sites.
- [x] 5.3 Run `task openspec:check` and the three touched packages whole (`go test ./internal/publish/ ./internal/config/ ./internal/cuemod/`). Verify: both pass.
- [x] 5.4 `task lint` and `task test:unit` green, then commit `test(errors): refuse a new error-text match outside the named exceptions`
