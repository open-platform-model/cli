## Why

The build pipeline's namespace resolution currently collapses `--namespace`, `OPM_NAMESPACE`, and `config.kubernetes.namespace` into a single value before the module is loaded, meaning `module.metadata.defaultNamespace` can never be correctly inserted into the precedence chain. The module author's declared default namespace is either ignored or silently overridden by the config file's `"default"` fallback.

## What Changes

- Remove the `ExtractMetadata` CUE fallback for `metadata.name` in `pipeline.Render()` — `name!` is a mandatory CUE field and is always present after Phase 2; the intermediate fallback is unnecessary
- Introduce a 4-step namespace resolution order for commands that use the build pipeline (`mod build`, `mod apply`, `mod diff`, `mod vet`):
  1. `--namespace` flag
  2. `OPM_NAMESPACE` environment variable
  3. `module.metadata.defaultNamespace` (from the loaded module, if set)
  4. `config.kubernetes.namespace` from `.opm/config.cue`
- The pipeline receives the flag value and the config default as separate inputs; `module.metadata.defaultNamespace` is inserted as step 3 after the module is loaded in the PREPARATION phase
- Commands that do not use the build pipeline (`mod delete`, `mod status`) are unaffected — they continue using the existing 3-step resolution (`--namespace` > `OPM_NAMESPACE` > `config.kubernetes.namespace`)
- `GlobalConfig.Kubernetes.Namespace` is not mutated — namespace is resolved transiently inside the pipeline for build-pipeline commands

## Capabilities

### New Capabilities

- `namespace-resolution-precedence`: Correct 4-step namespace precedence for build-pipeline commands, inserting `module.metadata.defaultNamespace` between env/flag resolution and config file fallback

### Modified Capabilities

- `render-pipeline`: The PREPARATION phase now participates in namespace resolution by surfacing `module.metadata.defaultNamespace` as step 3 in the precedence chain

## Impact

- `internal/build/pipeline.go` — `resolveNamespace()` receives flag value, module default namespace, and config default as separate arguments; applies 4-step logic
- `internal/build/module/loader.go` — `ExtractMetadata()` CUE fallback removed; `module.MetadataPreview` type removed if no longer needed; `module.Load()` surfaces `defaultNamespace` from AST inspection
- `internal/cmdutil/render.go` — passes flag/env-resolved namespace and config-default namespace separately to `RenderOptions` so the pipeline can apply step 3
- `internal/build/types.go` — `RenderOptions` may need a new field to carry the config-default namespace separately from the flag/env-resolved value
- Related change: `core-module-receiver-methods` — `module.Load()` introduced there must surface `defaultNamespace` from the module; this change depends on that loader being in place
- SemVer: **PATCH** — no change to CLI flags or public interface; behavior change only when `module.metadata.defaultNamespace` is set and neither `--namespace` nor `OPM_NAMESPACE` is provided
