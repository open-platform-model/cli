## Context

Configuration resolution is currently split across three layers:

1. **`config/resolver.go`** — Resolves `registry` and `configPath` using Flag > Env > Config > Default precedence. Has proper `Source` tracking and shadowed-value recording.
2. **`cmd/root.go` `initializeGlobals()`** — Calls `config.LoadOPMConfig()`, stores `resolvedRegistry` as a package-level var, logs raw cobra flag values.
3. **Subcommand `RunE` functions** — Each subcommand (apply, build, diff) calls `config.LoadOPMConfig()` again. Uses `resolveFlag(localFlag, GetGlobalFlag())` for kubeconfig/context/namespace, but `GetGlobalFlag()` returns raw cobra defaults (`""`), so config.cue values are never reached.

The `config/resolver.go` pattern is well-designed — `ResolveRegistry` and `ResolveConfigPath` both return result structs with value, source, and shadowed info. The problem is that this pattern was only applied to registry and config path, not to the other fields. And the `OPMConfig` loaded in root.go is discarded (local variable, no way to pass to subcommands).

Commands that don't need the build pipeline (`delete`, `status`) never call `LoadOPMConfig` — they only need kubeconfig, context, and namespace. These commands currently get those values from raw flags only.

## Goals / Non-Goals

**Goals:**

- Resolve ALL config values (kubeconfig, context, namespace, registry, provider, output format) once in `PersistentPreRunE` using Flag > Env > Config > Default precedence
- Store the resolved config and `OPMConfig` so subcommands can access them without re-loading
- Auto-resolve provider to the single configured provider when exactly one exists
- Emit a single "initializing CLI" debug log showing final resolved values
- Remove redundant debug log lines from `config/loader.go` and `build/` packages

**Non-Goals:**

- Changing the two-phase config loading mechanism (bootstrap registry extraction + full CUE load) — that design is sound
- Adding new flags or config fields
- Changing the `config/resolver.go` API — we'll extend the pattern, not replace it
- Restructuring the `build/` pipeline's internal logging beyond removing duplicates

## Decisions

### Decision 1: Extend the resolver pattern to all config fields

**Choice**: Add a `ResolveAll()` function in `config/resolver.go` that resolves all fields at once, returning a `ResolvedConfig` struct.

**Why**: The existing `ResolveRegistry` and `ResolveConfigPath` functions follow a clean pattern (options in, result with source tracking out). Extending this to kubeconfig, context, namespace, and provider keeps the resolution logic in one place and testable in isolation.

**Alternative considered**: Resolve each field individually with separate functions (`ResolveKubeconfig`, `ResolveNamespace`, etc.). Rejected because: (a) all fields follow the same Flag > Env > Config > Default pattern, (b) a single function with a single options struct is simpler, and (c) the env var names and config paths are known at compile time.

**Shape of `ResolvedConfig`**:

```go
type ResolvedConfig struct {
    ConfigPath  ResolvedField  // from ResolveConfigPath (already exists)
    Registry    ResolvedField  // from ResolveRegistry (already exists)
    Kubeconfig  ResolvedField  // NEW: flag > OPM_KUBECONFIG > config > ~/.kube/config
    Context     ResolvedField  // NEW: flag > OPM_CONTEXT > config > ""
    Namespace   ResolvedField  // NEW: flag > OPM_NAMESPACE > config > "default"
    Provider    ResolvedField  // NEW: flag > config auto-resolve > ""
    Output      string         // from flag only (no env/config source)
}

type ResolvedField struct {
    Value    string
    Source   Source
    Shadowed map[Source]string
}
```

### Decision 2: Store OPMConfig and ResolvedConfig as package-level state in root.go

**Choice**: Add two package-level variables in `cmd/root.go`:

```go
var (
    opmConfig      *config.OPMConfig      // full CUE config (providers, CueContext)
    resolvedConfig *config.ResolvedConfig  // all resolved values with sources
)
```

Expose via `GetOPMConfig()` and `GetResolvedConfig()` accessor functions.

**Why**: This matches the existing pattern (`resolvedRegistry` is already a package-level var with a `GetRegistry()` accessor). Subcommands already call `Get*()` functions — they'll just switch to `GetResolvedConfig().Namespace.Value` instead of `GetNamespace()`. The old `Get*()` helpers can be updated to delegate to `resolvedConfig` for backward compatibility during transition.

**Alternative considered**: Pass config through cobra's `cmd.Context()`. Rejected because: it would require changing every subcommand's signature and doesn't match the existing codebase conventions. Could be revisited later but is out of scope for this fix.

### Decision 3: Provider auto-resolution when single provider configured

**Choice**: During `ResolveAll()`, if no `--provider` flag is set AND `OPMConfig.Providers` has exactly one entry, auto-select that provider. Source is recorded as `"config-auto"`.

**Why**: When config.cue defines `providers: { kubernetes: ... }` and nothing else, requiring `--provider kubernetes` is pointless friction. This follows Principle VII (simplicity).

**Rules**:

- 0 providers configured, no flag → provider stays `""` (commands that need it will fail with a clear error)
- 1 provider configured, no flag → auto-select, source = `"config-auto"`
- N>1 providers configured, no flag → provider stays `""` (explicit choice required)
- Flag set → flag wins regardless of provider count

### Decision 4: Subcommands stop calling LoadOPMConfig

**Choice**: Remove `config.LoadOPMConfig()` calls from `mod_apply.go`, `mod_build.go`, `mod_diff.go`, and `config_vet.go`. They use `GetOPMConfig()` instead.

**Why**: Eliminates the double-load and the duplicate debug logging it causes. The `OPMConfig` is identical between the two calls (same flags, same env, same config file), so loading twice is pure waste.

**Special case — `config vet`**: This command's purpose is to validate the config. It should still work, but it validates the already-loaded config rather than loading a second time.

### Decision 5: Subcommand local flags override resolved values

**Choice**: Subcommands that have local flags (`--kubeconfig`, `--namespace`, `--context` on apply/delete/status/diff) override the globally-resolved values. The `resolveFlag()` helper becomes:

```go
func resolveFlag(localFlag string, resolved config.ResolvedField) string {
    if localFlag != "" {
        return localFlag
    }
    return resolved.Value
}
```

**Why**: Local flags should have highest precedence — a user running `opm mod apply -n production` expects production regardless of what config.cue says. The globally-resolved value already incorporates Flag > Env > Config > Default for the global `--namespace`, so the local flag simply adds one more layer on top.

### Decision 6: Logging cleanup — what stays, what goes

**Stays** (in `config/loader.go`):

- `"resolved config path"` — useful, shows where config was loaded from
- `"resolved registry"` — useful, shows final registry and source

**Removed** (from `config/loader.go`):

- `"bootstrap: extracted registry from config"` — internal detail, redundant with "resolved registry"
- `"setting CUE_REGISTRY for config load"` — internal detail
- `"extracted provider from config"` (per-provider) — redundant with provider loader output
- `"extracted providers from config"` (count) — redundant with provider loader output

**Removed** (from `build/`):

- `"release built successfully"` in `release_builder.go` (lines 191, 253) — duplicate of pipeline-level log
- `"loading provider"` in `provider.go` (line 62) — redundant with `"loaded provider"` that follows

**Stays** (in `build/`):

- `"release built"` in `pipeline.go` — single source of truth for release build summary
- `"loaded provider"` in `provider.go` — summary with name, version, transformer count
- `"extracted transformer"` in `provider.go` — useful for debugging transformer matching

**Modified** (in `cmd/root.go`):

- `"initializing CLI"` — changes from raw flag values to resolved values from `ResolvedConfig`

**Modified** (in `build/provider.go`):

- `"extracted transformer"` — merge `name` and `fqn` fields: use FQN as `name`, drop `fqn` field

## Risks / Trade-offs

**[Risk] Subcommands that don't need full config may fail on CUE errors** → Mitigation: `initializeGlobals` already handles this — if `LoadOPMConfig` fails, it logs the error and continues. `opmConfig` will be nil. Subcommands that need it (apply, build, diff) check for nil and fail with a clear error. Commands that don't need it (delete, status, version) are unaffected.

**[Risk] Provider auto-resolution could surprise users who later add a second provider** → Mitigation: When auto-resolving, the debug log shows `source=config-auto`. When a second provider is added and no `--provider` flag is passed, the build pipeline will fail with the existing "provider required" error. This is the correct behavior — the user needs to make an explicit choice.

**[Risk] Removing debug log lines reduces observability during config loading issues** → Mitigation: The kept lines ("resolved config path", "resolved registry", "initializing CLI" with full values) provide all the information needed for debugging. The removed lines were implementation details that added noise without diagnostic value. If needed, they can be added back at TRACE level in the future.

**[Trade-off] Package-level state vs. dependency injection**: Storing `opmConfig` as a package-level var in root.go is not ideal for testing, but it matches the existing pattern and avoids a larger refactor. A future change could move to cobra context propagation if needed.
