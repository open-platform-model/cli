## Why

The CLI config implementation exists in `cli/internal/config/` but lacks formal specification. Documenting the config system establishes the contract for how configuration loading, resolution, and validation works - enabling consistent behavior and future enhancements.

## What Changes

- Document the existing config implementation as a formal spec
- Establish the two-phase loading architecture (bootstrap → full load)
- Codify the precedence rules for configuration resolution (flag > env > config > default)
- Define the CUE-native config approach and its rationale
- Specify the file structure (`~/.opm/config.cue`, `cue.mod/module.cue`)

## Capabilities

### New Capabilities

- `config`: Core CLI configuration system - loading, resolution, paths, templates, and the two-phase bootstrap process
- `config-commands`: The `config init` and `config vet` commands and their behavior

### Modified Capabilities

(none - this is documentation of existing implementation)

## Impact

- **Code**: `cli/internal/config/` (config.go, loader.go, resolver.go, paths.go, templates.go)
- **Commands**: `cli/internal/cmd/` (config.go, config_init.go, config_vet.go)
- **Dependencies**: CUE SDK (`cuelang.org/go/cue`)
- **User files**: `~/.opm/config.cue`, `~/.opm/cue.mod/module.cue`
- **Environment**: OPM_REGISTRY, OPM_CONFIG, OPM_KUBECONFIG, OPM_CONTEXT, OPM_NAMESPACE

This is a PATCH-level change (documentation only, no behavioral changes).
