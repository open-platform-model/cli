## Why

`opm mod init` reads module identity off the raw CUE value twice although the library has already decoded it. `assertDerives` (internal/scaffold/scaffold.go) and `statedVersion` (internal/scaffold/repair.go) both acquire a `*module.Module` through `Kernel.AcquireModuleFromDir`, then look up `metadata.modulePath` and `metadata.version` in `mod.Package` with hand-built CUE paths. The kernel decodes those same fields into `mod.Metadata` (`ModulePath`, `Version`) during the acquire, and the `pkg-types` spec already says the CLI uses the library's metadata types wherever it holds decoded metadata.

The owner settled this in the beta.1 kernel walkthrough (task d2): "cli scaffold sites switch to mod.Metadata now". The rest of d2 (the instance's module metadata, its merged values and the module's `debugValues`) needs new library accessors and is a separate cli change after that library release. This change is the scaffold half. It needs no library release.

## What Changes

- `assertDerives` compares `mod.Metadata.ModulePath` and `mod.Metadata.Version` against the expected values instead of looking up `metadata.<field>` in `mod.Package`. Its "does not evaluate" internal-error branch becomes a nil-`Metadata` check. The two fields are checked in a fixed order (modulePath, then version) instead of map order.
- `statedVersion` reads `mod.Metadata.Version`. A nil `Metadata` or an empty version refuses as "not stated", as an absent field does today.
- The `cuelang.org/go/cue` import goes from both files once nothing else uses it.
- The `pkg-types` requirement that the CLI uses the library's metadata types gains a scenario: the scaffold reads an acquired module's identity from `Metadata`, not from `Package`.

Not in this change: the six render, vet and instance-init sites, which wait for library accessors (`Instance.ModuleMetadata()`, `Instance.Values()`, `Module.DebugValues()`). No `Module.InitValues()` accessor either (supervisor decision SD9).

SemVer class: none for users. This is a refactor with no change to commands, flags, output or exit codes, so after GA it would ship in a PATCH. During beta it rides the next `-beta.N` and does not trigger a release by itself (`refactor`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pkg-types`: the "Core types exported in pkg/" requirement adds that a held `*module.Module`'s identity is read from its decoded `Metadata`, with a scenario for the scaffold.

## Impact

- Code: `internal/scaffold/scaffold.go` (`assertDerives`), `internal/scaffold/repair.go` (`statedVersion`).
- Commands: `opm module init` (scaffold and `--from` clone, the post-rewrite assertion) and `opm module init` in repair mode (identity creation). Same refusals, same headlines.
- Tests: the existing `DetectRepair` unit tests and the `TestE2E_ModInit_*` e2e tests (including `TestE2E_ModInit_NonDerivingDonorRefuses`) are the regression net. No new tests.
- Dependencies: none. Library pin unchanged (`v1.0.0-beta.4` already exposes `Module.Metadata`).
