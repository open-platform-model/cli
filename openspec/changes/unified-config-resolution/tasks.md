## 1. Extend config/resolver with ResolvedConfig and ResolveAll

- [ ] 1.1 Add `ResolvedField` struct to `config/resolver.go` with `Value string`, `Source Source`, `Shadowed map[Source]string`
- [ ] 1.2 Add `SourceConfigAuto Source = "config-auto"` constant for provider auto-resolution
- [ ] 1.3 Add `ResolvedConfig` struct to `config/resolver.go` with fields: ConfigPath, Registry, Kubeconfig, Context, Namespace, Provider (all `ResolvedField`), plus Output `string`
- [ ] 1.4 Add `ResolveAllOptions` struct with flag values for all fields, config values from `*Config`, and provider names from `map[string]cue.Value`
- [ ] 1.5 Implement `ResolveAll(opts ResolveAllOptions) (*ResolvedConfig, error)` — resolve each field using Flag > Env > Config > Default precedence, reusing existing `ResolveConfigPath` and `ResolveRegistry` internally
- [ ] 1.6 Implement provider auto-resolution within `ResolveAll`: if `--provider` flag is empty and exactly one provider exists, auto-select it with source `config-auto`
- [ ] 1.7 Write tests for `ResolveAll`: flag overrides env, env overrides config, config overrides default, default used when nothing set, provider auto-resolution with 0/1/N providers, flag overrides auto-resolution

## 2. Store resolved state in cmd/root.go

- [ ] 2.1 Add package-level `opmConfig *config.OPMConfig` and `resolvedConfig *config.ResolvedConfig` variables
- [ ] 2.2 Add `GetOPMConfig() *config.OPMConfig` and `GetResolvedConfig() *config.ResolvedConfig` accessor functions
- [ ] 2.3 Update `initializeGlobals` to call `config.ResolveAll()` after `LoadOPMConfig`, passing flag values, config values from `opmConfig.Config`, and provider names from `opmConfig.Providers`
- [ ] 2.4 Store results in the new package-level variables
- [ ] 2.5 Update existing `Get*()` functions (`GetKubeconfig`, `GetContext`, `GetNamespace`, `GetRegistry`, `GetProvider`) to delegate to `resolvedConfig` fields
- [ ] 2.6 Remove the standalone `resolvedRegistry` variable (replaced by `resolvedConfig.Registry`)

## 3. Update "initializing CLI" log

- [ ] 3.1 Replace raw flag values in the "initializing CLI" debug log with resolved values from `resolvedConfig` (kubeconfig, context, namespace, config path, registry, provider)
- [ ] 3.2 Remove `registry_flag` from the log output (redundant with resolved registry)

## 4. Remove redundant debug logs from config/loader.go

- [ ] 4.1 Remove `output.Debug("bootstrap: extracted registry from config", ...)` at loader.go:61
- [ ] 4.2 Remove `output.Debug("setting CUE_REGISTRY for config load", ...)` at loader.go:171
- [ ] 4.3 Remove `output.Debug("extracted provider from config", ...)` at loader.go:254
- [ ] 4.4 Remove `output.Debug("extracted providers from config", ...)` at loader.go:261

## 5. Remove redundant debug logs from build packages

- [ ] 5.1 Remove `output.Debug("release built successfully", ...)` at build/release_builder.go:191 (Build method)
- [ ] 5.2 Remove `output.Debug("release built successfully", ...)` at build/release_builder.go:253 (BuildFromValue method)
- [ ] 5.3 Remove `output.Debug("loading provider", ...)` at build/provider.go:62
- [ ] 5.4 Update `output.Debug("extracted transformer", ...)` at build/provider.go:141 to use FQN as `name` field and remove the separate `fqn` field

## 6. Remove duplicate LoadOPMConfig calls from subcommands

- [ ] 6.1 Update `cmd/mod_apply.go` to use `GetOPMConfig()` instead of calling `config.LoadOPMConfig()`, and use `GetResolvedConfig()` for resolved field values
- [ ] 6.2 Update `cmd/mod_build.go` to use `GetOPMConfig()` instead of calling `config.LoadOPMConfig()`, and use `GetResolvedConfig()` for resolved field values
- [ ] 6.3 Update `cmd/mod_diff.go` to use `GetOPMConfig()` instead of calling `config.LoadOPMConfig()`, and use `GetResolvedConfig()` for resolved field values
- [ ] 6.4 Update `cmd/config_vet.go` to use `GetOPMConfig()` instead of re-loading config
- [ ] 6.5 Update `resolveFlag()` helper in `cmd/mod_apply.go` to accept `config.ResolvedField` as the fallback instead of a raw string
- [ ] 6.6 Update all `resolveFlag` call sites in apply, delete, diff, status to use `GetResolvedConfig()` fields

## 7. Update "rendering module" log to show resolved provider

- [ ] 7.1 In `cmd/mod_apply.go`, ensure the `provider` variable used in `output.Debug("rendering module", ...)` comes from the resolved config (not empty flag)
- [ ] 7.2 In `cmd/mod_diff.go`, same fix for the "rendering module for diff" log if applicable

## 8. Validation

- [ ] 8.1 Run `task fmt` and fix any formatting issues
- [ ] 8.2 Run `task test` and fix any failures
- [ ] 8.3 Run `task check` (fmt + vet + test) and confirm all pass
- [ ] 8.4 Manual smoke test: run `opm mod apply . --name jellyfin --namespace default --verbose` and verify the debug output shows resolved values, no redundant lines, and provider is populated
