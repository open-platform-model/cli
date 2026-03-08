## REMOVED Requirements

### Requirement: Builder.Build() function
**Reason**: The separate build phase is eliminated. Loading IS building — `pkg/loader/LoadModuleReleaseFromValue()` handles value unification, gate validation, finalization, and metadata extraction in one pass. CUE evaluation naturally handles what the builder did imperatively.
**Migration**: Replace `builder.Build(ctx, mod, opts, valuesFiles)` calls with `loader.LoadReleasePackage()` + `loader.LoadModuleReleaseFromValue()`.

### Requirement: Values file resolution in builder
**Reason**: Values file resolution (--values flags, fallback to values.cue) moves to `internal/cmdutil/` as a CLI-layer concern. The loader accepts explicit file paths.
**Migration**: `cmdutil` resolves values file paths from flags and passes them to `loader.LoadReleasePackage()`.

### Requirement: Auto-secrets injection in builder
**Reason**: Go-side auto-secrets injection (`autosecrets.go`) is eliminated. The CUE layer handles secret discovery, grouping, and component injection via `#AutoSecrets` and `#OpmSecretsComponent` in `v1alpha1/core/`.
**Migration**: No Go-side migration needed — the CUE definitions handle this declaratively when `#ModuleRelease` is evaluated.

### Requirement: Values validation against #config in builder
**Reason**: Values validation moves into the gate system in `pkg/loader/validate.go`. The Module Gate validates consumer values against `#module.#config` during loading.
**Migration**: The loader's Module Gate replaces `builder.ValidateValues()`. Structured `FieldError` output is preserved via `ConfigError.FieldErrors()`.
