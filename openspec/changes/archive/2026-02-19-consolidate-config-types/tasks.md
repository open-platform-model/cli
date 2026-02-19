## 1. Define new types in config package

- [x] 1.1 Add `GlobalFlags` struct to `internal/config/config.go` with fields: `Config string`, `Registry string`, `Verbose bool`, `Timestamps bool`
- [x] 1.2 Add `GlobalConfig` struct to `internal/config/config.go` with fields: `Kubernetes`, `Log`, `Registry`, `ConfigPath`, `Providers`, `CueContext`, `Flags GlobalFlags`
- [x] 1.3 Delete `Config` struct and `DefaultConfig()` from `internal/config/config.go`
- [x] 1.4 Delete `OPMConfig` struct from `internal/config/config.go`

## 2. Update loader

- [x] 2.1 Change `LoadOPMConfig(opts LoaderOptions) (*OPMConfig, error)` to `Load(cfg *GlobalConfig, opts LoaderOptions) error` in `internal/config/loader.go`
- [x] 2.2 Set `cfg.ConfigPath` from the resolved config path inside the loader
- [x] 2.3 Update internal `loadFullConfig` to populate `GlobalConfig` fields directly instead of returning `*Config`
- [x] 2.4 Inline default values (previously in `DefaultConfig()`) into the loader
- [x] 2.5 Remove `RegistrySource` — do not store it on GlobalConfig

## 3. Update resolver

- [x] 3.1 Change `ResolveKubernetesOptions.Config` from `*Config` to `*GlobalConfig`, remove `ProviderNames []string` field
- [x] 3.2 Update `ResolveKubernetes` to read `opts.Config.Kubernetes.*` and extract provider names from `opts.Config.Providers` internally
- [x] 3.3 Delete `ResolveBase`, `ResolveBaseOptions`, `ResolvedBaseConfig`
- [x] 3.4 Delete `ResolveAll`, `ResolveAllOptions`, `ResolvedConfig`

## 4. Update cmdtypes

- [x] 4.1 Replace `GlobalConfig` in `internal/cmdtypes/cmdtypes.go` — remove `OPMConfig`, `ConfigPath`, `Registry`, `RegistryFlag`, `Verbose` fields; replace with single `*config.GlobalConfig` field or re-export (whichever avoids cycles). Since GlobalConfig is now in `config`, `cmdtypes` drops its `config` import entirely — remove the old `GlobalConfig` struct and have `cmd` packages use `*config.GlobalConfig` directly.

## 5. Update cmdutil

- [x] 5.1 Delete `ResolveKubernetes` wrapper from `internal/cmdutil/render.go`
- [x] 5.2 Update `RenderReleaseOpts`: replace `OPMConfig *config.OPMConfig` and `Registry string` with `Config *config.GlobalConfig`
- [x] 5.3 Update `RenderRelease` to use `opts.Config` for pipeline construction and registry
- [x] 5.4 Update `NewK8sClient` parameter types if needed (currently takes `*config.ResolvedKubernetesConfig` + string — unchanged)

## 6. Update root command

- [x] 6.1 Change `var cfg cmdtypes.GlobalConfig` to `var cfg config.GlobalConfig` in `internal/cmd/root.go`
- [x] 6.2 Set `cfg.Flags` from cobra flag values before calling loader
- [x] 6.3 Replace `config.LoadOPMConfig(opts)` + `cfg.OPMConfig = loadedConfig` with `config.Load(&cfg, opts)`
- [x] 6.4 Remove `ResolveBase` call and direct assignment of `cfg.ConfigPath`/`cfg.Registry` — these are now set by the loader
- [x] 6.5 Update timestamps resolution to read `cfg.Log.Timestamps` instead of `loadedConfig.Config.Log.Timestamps`
- [x] 6.6 Update all subcommand constructors to pass `*config.GlobalConfig` instead of `*cmdtypes.GlobalConfig`

## 7. Update mod commands

- [x] 7.1 Update `mod apply` — use `config.ResolveKubernetes(config.ResolveKubernetesOptions{Config: cfg, ...})`, `cfg.Log.Kubernetes.APIWarnings`, `RenderReleaseOpts{Config: cfg, ...}`
- [x] 7.2 Update `mod build` — same pattern as apply
- [x] 7.3 Update `mod diff` — same pattern as apply
- [x] 7.4 Update `mod vet` — same pattern as apply
- [x] 7.5 Update `mod delete` — use `config.ResolveKubernetes`, `cfg.Log.Kubernetes.APIWarnings`
- [x] 7.6 Update `mod status` — use `config.ResolveKubernetes`, `cfg.Log.Kubernetes.APIWarnings`

## 8. Update config commands

- [x] 8.1 Update `config vet` — use `cfg.Flags.Config`, `cfg.Flags.Registry`; call `config.Load(&temp, ...)` for re-validation

## 9. Update build package

- [x] 9.1 Change `NewPipeline(*config.OPMConfig)` to `NewPipeline(*config.GlobalConfig)` in `internal/build/pipeline.go`
- [x] 9.2 Update `pipeline` struct field from `*config.OPMConfig` to `*config.GlobalConfig`
- [x] 9.3 Update `NewProviderLoader` to accept `*config.GlobalConfig` in `internal/build/transform/provider.go`

## 10. Update tests

- [x] 10.1 Update `internal/config/resolver_test.go` — use `GlobalConfig` in resolver option structs, remove `ResolveBase`/`ResolveAll` tests
- [x] 10.2 Update `internal/config/config_test.go` — remove `TestDefaultConfig`, update for new types
- [x] 10.3 Update `internal/cmdutil/render_test.go` — use new `RenderReleaseOpts` shape
- [x] 10.4 Update `internal/build/pipeline_test.go` — construct `*config.GlobalConfig` instead of `*config.OPMConfig`
- [x] 10.5 Update any other test files that construct `OPMConfig` or `Config` inline

## 11. Validation

- [x] 11.1 Run `task check` (fmt + vet + test) — all must pass
- [x] 11.2 Run `task build` — binary must compile
- [x] 11.3 Verify no import cycles: `go vet ./...` passes
