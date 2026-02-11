## Why

The CLI's configuration resolution is fragmented across multiple call sites. `config.LoadOPMConfig()` is called twice per command — once in `PersistentPreRunE` (root.go) and again in each subcommand (apply, build, diff, config vet). The "initializing CLI" debug log displays raw cobra flag values (`kubeconfig=""`, `context=""`, `namespace=""`) instead of the final resolved values, making verbose output misleading and unhelpful for debugging.

More critically, the precedence chain defined in the config spec (Flag > Env > Config > Default) is only fully implemented for `registry`. For `kubeconfig`, `context`, and `namespace`, the config.cue values are never consulted — the `resolveFlag()` helper chains local flags to global flags, but global flags default to `""`, so the config file defaults are never reached. This means setting `kubernetes: { namespace: "staging" }` in config.cue has no effect.

Additionally, there is no auto-resolution for the provider field when exactly one provider is configured, resulting in `provider=""` in debug output even though the config file unambiguously defines a single provider.

## What Changes

- **Centralize all config resolution into `PersistentPreRunE`**: Load config once, resolve all values (kubeconfig, context, namespace, registry, provider) using the full precedence chain, and store the result in a shared `ResolvedConfig` struct accessible to all subcommands.
- **Eliminate duplicate `LoadOPMConfig()` calls**: Subcommands (`mod apply`, `mod build`, `mod diff`, `config vet`) will use the pre-resolved config instead of re-loading.
- **Implement full precedence for kubeconfig, context, namespace**: These fields will resolve through Flag > Env > Config > Default, matching the existing config spec's requirements.
- **Auto-resolve provider when single provider configured**: When config.cue defines exactly one provider, it becomes the default provider without requiring `--provider` flag.
- **Clean up verbose debug logging**: Remove redundant log lines produced during the two-phase config load (bootstrap, CUE_REGISTRY, per-provider extraction). Replace with a single "initializing CLI" log showing final resolved values. Remove duplicate "release built" messages from the build pipeline.

## Capabilities

### New Capabilities

- `provider-auto-resolution`: When exactly one provider is defined in config.cue, auto-select it as the default provider. This eliminates the need for `--provider` when only one provider exists. When multiple providers exist, `--provider` remains required (no change).

### Modified Capabilities

- `config`: The "Configuration precedence chain" requirement is already specified but not fully implemented for kubeconfig, context, and namespace. This change completes the implementation. The "Resolution tracking for debugging" requirement will be refined — the single "initializing CLI" log line will show final resolved values with their sources, replacing the current scattered debug output.

## Impact

- **Files modified**: `cmd/root.go`, `config/loader.go`, `config/config.go`, `cmd/mod_apply.go`, `cmd/mod_build.go`, `cmd/mod_diff.go`, `cmd/config_vet.go`, `build/release_builder.go`, `build/pipeline.go`, `build/provider.go`
- **SemVer**: PATCH — fixes broken resolution and noisy logging, no new flags or commands, no breaking changes
- **Risk**: Low-medium. The resolution logic changes could affect existing behavior if anyone relies on the current (broken) config.cue value passthrough. All existing tests must pass. The provider auto-resolution adds a new default behavior, but only activates when a single provider is configured (safe default).
- **Testing**: Existing config resolver tests cover the precedence chain. New tests needed for: full resolution with config values, provider auto-resolution, and single "initializing CLI" log output verification.
