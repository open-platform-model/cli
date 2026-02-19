## Context

The CLI configuration is split across three Go types:

- `config.Config` — plain deserialization of `~/.opm/config.cue` (registry, kubernetes, log)
- `config.OPMConfig` — wraps `*Config` and adds runtime state (resolved registry, providers, CUE context)
- `cmdtypes.GlobalConfig` — wraps `*OPMConfig` and adds CLI flags (verbose, registry flag)

This three-layer nesting forces resolver helpers to accept `*Config` + separate provider-name slices, a `cmdutil.ResolveKubernetes` wrapper to exist solely for unpacking, and call sites to write `cfg.OPMConfig.Config.Log.Kubernetes.APIWarnings`. The root command also redundantly re-resolves the registry via `ResolveBase` after the loader already resolved it.

The CUE config file (`~/.opm/config.cue`) defines `registry`, `providers`, `kubernetes`, and `log` as siblings. The Go type split doesn't mirror this structure.

## Goals / Non-Goals

**Goals:**

- Single configuration type (`GlobalConfig`) in the `config` package
- `GlobalFlags` substruct for raw CLI flag values
- Resolver functions accept `*GlobalConfig` directly
- Eliminate redundant `ResolveBase`/`ResolveAll`, the `cmdutil.ResolveKubernetes` wrapper, and the `RegistrySource` field
- Simplify root.go initialization to a single `config.Load` call
- No user-observable behavior change (PATCH)

**Non-Goals:**

- Changing the config file format or CUE schema
- Changing CLI flags, output, or exit codes
- Changing the precedence resolution logic (flag > env > config > default)
- Eliminating the `cmdtypes` package entirely (it still holds exit-related utilities)
- Refactoring the build pipeline beyond updating the config type it accepts

## Decisions

### Decision 1: GlobalConfig lives in `config`, not `cmdtypes`

**Choice**: Define `GlobalConfig` and `GlobalFlags` in `internal/config`.

**Rationale**: The `config` package already owns `KubernetesConfig`, `LogConfig`, the loader, and the resolvers — everything that reads and writes `GlobalConfig` fields. Placing the type here avoids moving sub-types between packages and keeps the dependency graph simple: `config` has no new internal imports, `build` continues importing `config`, and `cmdtypes` drops its `config` import.

**Alternative considered**: Define in `cmdtypes`. Rejected because it would require moving `KubernetesConfig`/`LogConfig` to `cmdtypes` (or creating a cycle), and `cmdtypes` would gain a CUE SDK dependency it doesn't need.

### Decision 2: Loader mutates `*GlobalConfig` instead of returning a new struct

**Choice**: `config.Load(cfg *GlobalConfig, opts LoaderOptions) error` populates fields on the caller's struct.

**Rationale**: The root command creates `GlobalConfig` as a stack variable and passes `&cfg` to all subcommands. Having the loader write directly into it avoids an intermediate struct or a field-by-field copy. The loader sets: `Kubernetes`, `Log`, `Registry`, `ConfigPath`, `Providers`, `CueContext`. The caller sets `Flags` before or after.

**Alternative considered**: Return a `*GlobalConfig`. Rejected because root.go needs a stable address shared across subcommands — copying would add noise.

### Decision 3: Resolver accepts `*GlobalConfig` in options struct

**Choice**: `ResolveKubernetesOptions.Config *GlobalConfig` replaces `Config *Config` + `ProviderNames []string`.

**Rationale**: The resolver internally reads `.Kubernetes.*` for config-tier values and iterates `.Providers` for provider names. This eliminates the `cmdutil.ResolveKubernetes` wrapper whose sole job was this unpacking. All six mod commands call `config.ResolveKubernetes` directly.

### Decision 4: Remove `ResolveBase` and `ResolveAll`

**Choice**: Delete both functions and their option/result types.

**Rationale**: `ResolveBase` has one production caller (root.go) that re-resolves the registry after `LoadOPMConfig` already resolved it — redundant. After the loader populates `cfg.Registry` and `cfg.ConfigPath`, `ResolveBase` is unnecessary. `ResolveAll` has zero production callers. The underlying helpers (`ResolveRegistry`, `ResolveConfigPath`, `resolveStringField`, `resolveProvider`) remain — they're used by the loader and `ResolveKubernetes`.

### Decision 5: Drop `Registry` from `RenderReleaseOpts`

**Choice**: `RenderReleaseOpts` carries `Config *config.GlobalConfig` instead of separate `OPMConfig` + `Registry` fields.

**Rationale**: The pipeline already receives the full config via `NewPipeline(cfg)` and reads `.Registry` from it. Passing `Registry` separately was redundant — it was always `cfg.OPMConfig.Registry`, the same value the pipeline reads.

### Decision 6: `build.NewPipeline` accepts `*config.GlobalConfig`

**Choice**: The pipeline constructor takes `*config.GlobalConfig` instead of `*config.OPMConfig`.

**Rationale**: The pipeline reads exactly three fields: `.CueContext`, `.Registry`, `.Providers`. These are direct fields on the merged `GlobalConfig`. The pipeline ignores `Kubernetes`, `Log`, `Flags` — the same pattern as today where it ignores `OPMConfig.Config` and `RegistrySource`. No narrower interface type is introduced (YAGNI).

## Risks / Trade-offs

- **`build` package sees CLI-level fields**: `NewPipeline` receives `*GlobalConfig` which includes `Flags.Verbose` etc. that build never reads. This is unchanged from today (build already ignores `OPMConfig.RegistrySource`, `OPMConfig.Config.Log`, etc.). Acceptable given Principle VII — adding an interface to narrow access is not justified.
- **`config vet` re-loads into a throwaway `GlobalConfig`**: Config vet validates by calling `config.Load(&temp, ...)` with a temporary. This is functionally identical to today's `config.LoadOPMConfig(...)` call — minimal risk.
- **Test churn**: Resolver tests and render tests construct option/config structs inline. All of them need updating. Mitigated by the fact that the new structs are simpler (fewer fields, no nesting).
