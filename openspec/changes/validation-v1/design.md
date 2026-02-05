# Design: CLI Validation Commands

This document contains implementation details, pseudo-code, validation pipeline diagrams, and CLI reference for `opm mod vet` and `opm config vet`.

## Module Validation Pipeline

```text
┌─────────────────────────────────────────────────────────────────┐
│                    OPM Module Validation Pipeline               │
├─────────────────────────────────────────────────────────────────┤
│  Phase 1: Load & Build                                          │
│           ├─ Parse all .cue files via cue/load                  │
│           ├─ Resolve imports using configured registry          │
│           │   (--registry > OPM_REGISTRY > config.registry)     │
│           └─ Build unified value with cuecontext                │
│                                                                 │
│           Catches: syntax errors, missing imports, conflicts    │
│           Report: Errors with file:line:col                     │
│           Exit on failure: code 2                               │
├─────────────────────────────────────────────────────────────────┤
│  Phase 2: Schema Validation                                     │
│           ├─ Call value.Validate()                              │
│           ├─ Unify with core.#Module schema                     │
│           └─ Extract entity counts for summary                  │
│                                                                 │
│           Catches: schema violations, constraint errors         │
│           Report: Schema violations with entity context         │
│           Exit on failure: code 2                               │
├─────────────────────────────────────────────────────────────────┤
│  Phase 3: Concrete Validation (if --concrete flag)              │
│           ├─ Call value.Validate(cue.Concrete(true))            │
│           └─ Identify open/incomplete fields                    │
│                                                                 │
│           Catches: non-concrete fields, incomplete values       │
│           Report: List of non-concrete fields                   │
│           Exit on failure: code 2                               │
├─────────────────────────────────────────────────────────────────┤
│  Output: Success Summary                                        │
│           ├─ Module [name] validated                            │
│           ├─ [N] components validated                           │
│           └─ [N] scopes validated (if present)                  │
│                                                                 │
│           Exit: code 0                                          │
└─────────────────────────────────────────────────────────────────┘
```

## Config Validation Pipeline

```text
┌─────────────────────────────────────────────────────────────────┐
│                 OPM Config Validation Pipeline                  │
├─────────────────────────────────────────────────────────────────┤
│  Phase 1: Load & Build                                          │
│           ├─ Parse config.cue via cue/load                      │
│           ├─ Resolve imports using configured registry          │
│           │   (--registry > OPM_REGISTRY > config.registry)     │
│           └─ Build unified value with cuecontext                │
│                                                                 │
│           Catches: syntax errors, missing imports               │
│           Report: Errors with file:line:col                     │
│           Exit on failure: code 2                               │
├─────────────────────────────────────────────────────────────────┤
│  Phase 2: Schema Validation                                     │
│           ├─ Call value.Validate()                              │
│           ├─ Unify with core.#Config schema                     │
│           └─ Validate field constraints (registry format, etc.) │
│                                                                 │
│           Catches: schema violations, invalid field values      │
│           Report: Schema violations with field path             │
│           Exit on failure: code 2                               │
├─────────────────────────────────────────────────────────────────┤
│  Output: Success Summary                                        │
│           ├─ Config [path] validated                            │
│           └─ Registry: [url]                                    │
│                                                                 │
│           Exit: code 0                                          │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Pseudo-code

### Module Validation

```go
package validator

import (
    "cuelang.org/go/cue"
    "cuelang.org/go/cue/cuecontext"
    "cuelang.org/go/cue/load"
)

type ModuleValidator struct {
    ctx         *cue.Context
    moduleRoot  string
    pkgName     string
    debug       bool
    concrete    bool
    valuesFiles []string
}

func NewModuleValidator(moduleRoot, pkgName string) *ModuleValidator {
    return &ModuleValidator{
        ctx:        cuecontext.New(),
        moduleRoot: moduleRoot,
        pkgName:    pkgName,
    }
}

func (v *ModuleValidator) Validate() (*ValidationResult, error) {
    var errors []ValidationError
    
    // Phase 1: Load & Build
    cfg := &load.Config{
        ModuleRoot: v.moduleRoot,
        Package:    v.pkgName,
        Dir:        v.moduleRoot,
    }
    
    instances := load.Instances([]string{}, cfg)
    if len(instances) == 0 {
        return nil, exitWithCode(2, "no CUE instances found")
    }
    
    inst := instances[0]
    if inst.Err != nil {
        errors = append(errors, formatCUEError(inst.Err))
        return nil, &ValidationErrors{Errors: errors}
    }
    
    value := v.ctx.BuildInstance(inst)
    if value.Err() != nil {
        errors = append(errors, formatCUEError(value.Err()))
        return nil, &ValidationErrors{Errors: errors}
    }
    
    // Phase 2: Schema Validation
    if err := value.Validate(); err != nil {
        errors = append(errors, formatCUEError(err))
    }
    
    // Validate against core.#Module schema
    moduleSchema := value.LookupPath(cue.ParsePath("#Module"))
    if moduleSchema.Err() != nil {
        errors = append(errors, ValidationError{
            Message: "module does not conform to #Module schema",
        })
    }
    
    // Phase 3: Concrete Validation (optional)
    if v.concrete {
        if err := value.Validate(cue.Concrete(true), cue.Final()); err != nil {
            errors = append(errors, formatConcreteError(err))
        }
    }
    
    // Fail-on-end: return all errors
    if len(errors) > 0 {
        return nil, &ValidationErrors{Errors: errors}
    }
    
    // Extract entity counts for summary
    return &ValidationResult{
        ModuleName:     extractModuleName(value),
        ComponentCount: countComponents(value),
        ScopeCount:     countScopes(value),
    }, nil
}
```

### Config Validation

```go
type ConfigValidator struct {
    ctx        *cue.Context
    configPath string
    registry   string
}

func NewConfigValidator(configPath string) *ConfigValidator {
    return &ConfigValidator{
        ctx:        cuecontext.New(),
        configPath: configPath,
    }
}

func (v *ConfigValidator) Validate() (*ConfigResult, error) {
    var errors []ValidationError
    
    // Phase 1: Load & Build
    cfg := &load.Config{
        ModuleRoot: v.configPath,
        Package:    "config",
        Dir:        v.configPath,
    }
    
    instances := load.Instances([]string{}, cfg)
    if len(instances) == 0 {
        return nil, exitWithCode(2, "no CUE instances found in config directory")
    }
    
    inst := instances[0]
    if inst.Err != nil {
        errors = append(errors, formatCUEError(inst.Err))
        return nil, &ValidationErrors{Errors: errors}
    }
    
    value := v.ctx.BuildInstance(inst)
    if value.Err() != nil {
        errors = append(errors, formatCUEError(value.Err()))
        return nil, &ValidationErrors{Errors: errors}
    }
    
    // Phase 2: Schema Validation
    configValue := value.LookupPath(cue.ParsePath("config"))
    if !configValue.Exists() {
        errors = append(errors, ValidationError{
            Message: "config field not found",
        })
        return nil, &ValidationErrors{Errors: errors}
    }
    
    if err := configValue.Validate(); err != nil {
        errors = append(errors, formatCUEError(err))
    }
    
    // Validate against core.#Config schema
    // (implicit via CUE unification if config imports and uses core.#Config)
    
    if len(errors) > 0 {
        return nil, &ValidationErrors{Errors: errors}
    }
    
    // Extract registry for summary
    registry, _ := configValue.LookupPath(cue.ParsePath("registry")).String()
    
    return &ConfigResult{
        ConfigPath: v.configPath,
        Registry:   registry,
    }, nil
}
```

## CLI Reference

### Command: `mod vet`

#### Syntax

```bash
opm mod vet [path] [flags]
```

#### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `path` | No | Path to module directory (default: current directory `.`) |

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--package` | `-p` | string | `"main"` | CUE package to validate |
| `--debug` | | bool | `false` | Use `debug_values.cue` instead of `values.cue` |
| `--concrete` | | bool | `false` | Require all values to be concrete (no open fields) |
| `--values` | `-f` | []string | `[]` | Additional values files to unify (CUE, YAML, JSON) |
| `--registry` | | string | `""` | Override registry for CUE module resolution |

#### Examples

```bash
# Validate current directory module
opm mod vet

# Validate specific module path
opm mod vet ./modules/my-app

# Validate with debug values
opm mod vet --debug

# Require concrete values
opm mod vet --concrete

# Validate with custom values file
opm mod vet --values staging.cue

# Validate specific package
opm mod vet -p components

# Combine flags
opm mod vet --debug --concrete --values overrides.cue
```

### Command: `config vet`

#### Syntax

```bash
opm config vet [path] [flags]
```

#### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `path` | No | Path to config directory (default: `~/.opm/`) |

#### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--registry` | | string | `""` | Override registry for CUE module resolution |

#### Examples

```bash
# Validate default config at ~/.opm/
opm config vet

# Validate config at custom path
opm config vet /path/to/config

# Validate with registry override
opm config vet --registry localhost:5001
```

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Validation succeeded |
| `1` | General error (usage, config issues) |
| `2` | Validation failed (syntax, schema, concrete errors) |

### Example Output (Success)

```
✓ Module my-app validated
✓ 5 components validated
✓ 2 scopes validated
```

```
✓ Config validated: ~/.opm/config.cue
✓ Registry: localhost:5001
```

### Example Output (Failure)

```
Error: validation failed

  module.cue:24:5
  metadata.name: conflicting values "foo" and "bar"

  Suggested fix: Ensure metadata.name has a single concrete value
```

```
Error: config validation failed

  config.cue:15:5
  config.kubernetes.namespace: invalid value "INVALID" (must match RFC-1123)

  Suggested fix: Use lowercase letters, numbers, and hyphens only
```

## Error Formatting

Errors are formatted with Charm lipgloss styling:

1. **File location** (file:line:col) when available
2. **Clear description** of the error
3. **Suggested fix** for common errors
4. **Color-coded severity** (Info, Warning, Error)

All errors are aggregated and displayed before exiting (fail-on-end pattern), so users can fix multiple issues in a single iteration.

## Schema Dependencies

Both validators depend on schemas from `opmodel.dev/core@v0`:

| Validator | Schema | Location |
|-----------|--------|----------|
| Module | `core.#Module` | `catalog/v0/core/module.cue` |
| Config | `core.#Config` | `catalog/v0/core/config.cue` |
