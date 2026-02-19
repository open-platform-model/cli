## Why

Three separate Go types (`Config`, `OPMConfig`, `GlobalConfig`) represent what is conceptually one thing: the CLI's runtime configuration. `Config` is never stored or passed independently — it's only ever extracted from `OPMConfig.Config` as a transient local to feed resolver helpers. This split forces awkward patterns: resolver options take `*Config` + `[]string` for provider names, a `cmdutil.ResolveKubernetes` wrapper exists solely to unpack `OPMConfig` into those fields, and accessing log settings requires `cfg.OPMConfig.Config.Log.Kubernetes.APIWarnings` (4 levels deep). The root command also redundantly re-resolves the registry via `ResolveBase` after `LoadOPMConfig` already resolved it.

## What Changes

- **Merge `Config` and `OPMConfig` into `GlobalConfig`** in the `config` package. Promote `Kubernetes`, `Log` fields directly onto `GlobalConfig`. Remove the `Config` and `OPMConfig` types entirely.
- **Add `GlobalFlags` substruct** to `GlobalConfig` holding raw flag values (`--config`, `--registry`, `--verbose`, `--timestamps`). Replace the current loose fields (`RegistryFlag`, `Verbose`).
- **Remove `RegistrySource` field** — unused outside of construction.
- **Update `Resolve*Options` structs** to accept `*GlobalConfig` directly instead of `*Config` + `ProviderNames []string` (Option C approach).
- **Delete `cmdutil.ResolveKubernetes`** wrapper — callers use `config.ResolveKubernetes` directly.
- **Delete `ResolveBase` and `ResolveAll`** — zero production callers after root.go simplification. Keep the underlying helpers (`ResolveConfigPath`, `ResolveRegistry`, `resolveStringField`, `resolveProvider`).
- **Simplify `root.go`** — `config.Load` sets `Registry`, `ConfigPath`, and all config fields on `GlobalConfig` directly. No second resolution pass.
- **Drop `Registry` field from `RenderReleaseOpts`** — pipeline reads it from `GlobalConfig`.
- **Update `build.NewPipeline`** to accept `*config.GlobalConfig` instead of `*config.OPMConfig`.

## Capabilities

### New Capabilities

- `config-types`: Consolidated configuration type system — single `GlobalConfig` type with `GlobalFlags` substruct, unified loader, and resolver accepting `*GlobalConfig` directly.

### Modified Capabilities

- `config`: Resolution precedence chain implementation changes (resolver accepts `*GlobalConfig`, `ResolveBase`/`ResolveAll` removed, loader populates `GlobalConfig` directly). Requirements unchanged.
- `cmdutil`: `RenderModule` accepts `*config.GlobalConfig` instead of separate `OPMConfig` + `Registry` fields. `ResolveKubernetes` wrapper removed. Requirements updated for new signatures.

## Impact

- **Packages modified**: `config`, `cmdtypes`, `cmdutil`, `build`, `cmd/root.go`, `cmd/mod/*`, `cmd/config/vet.go`
- **Types deleted**: `config.Config`, `config.OPMConfig`, `config.ResolveBaseOptions`, `config.ResolveAllOptions`, `config.ResolvedBaseConfig`, `config.ResolvedConfig`
- **Functions deleted**: `cmdutil.ResolveKubernetes`, `config.ResolveBase`, `config.ResolveAll`, `config.DefaultConfig`
- **API change**: `build.NewPipeline(*config.OPMConfig)` becomes `build.NewPipeline(*config.GlobalConfig)`
- **SemVer**: PATCH — all changes are internal. No user-facing behavior, flags, output, or exit codes change.
- **`cmdtypes` package**: Shrinks to exit-related utilities only (loses `GlobalConfig`, `*config.OPMConfig` import).
- **Dependency graph**: `config` gains no new internal imports. `build` continues to import `config`. `cmdtypes` drops its `config` import.
