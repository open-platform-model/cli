# Proposal: Multi-Registry Routing

## Intent

Implement prefix-based multi-registry routing via `config.registries` map, enabling enterprise multi-registry environments. This replaces the single `config.registry` field with a flexible routing system compatible with CUE's `CUE_REGISTRY` semantics.

## Version Impact

**MAJOR** - Breaking changes to existing configuration:

- **BC-001**: `config.registry` (single string) is removed and replaced by `config.registries` (prefix-to-registry map)
- **BC-002**: `OPM_REGISTRY` now accepts `CUE_REGISTRY` format (comma-separated prefix routing) instead of a single URL

## Scope

**In scope:**

- `config.registries` prefix-to-registry map in `~/.opm/config.cue`
- `RegistryMap` type with `ToCUERegistry()` / `ParseCUERegistry()` conversions
- Precedence: flag > `OPM_REGISTRY` > `config.registries`
- Insecure registry support (`+insecure` suffix)
- Backward compatibility: single URL in `OPM_REGISTRY` still works as default registry

**Out of scope:**

- Distribution commands (`opm mod publish`, `opm mod get`, etc.) - see distribution-v1
- Registry authentication (handled by `~/.docker/config.json`)

## Affected Packages

| Package | Changes |
|---------|---------|
| `internal/config/` | `registry.go` - RegistryMap type and CUE_REGISTRY conversions |
| `internal/config/` | `config.go` - Update Config struct |
| `internal/config/` | `loader.go` - Update loading logic |

## Approach

1. Define `RegistryEntry` and `RegistryMap` types in new `registry.go` file
2. Implement bidirectional conversion to/from `CUE_REGISTRY` format
3. Update config loading to use new `registries` field
4. Maintain backward compatibility: single URL in `OPM_REGISTRY` treated as default registry

## Clarifications

- **CUE Compatibility**: The `ToCUERegistry()` output is passed directly to `CUE_REGISTRY` env var, ensuring `opm` and `cue` CLI behavior is identical.
- **Prefix Matching**: Longest prefix wins (e.g., `company.internal/critical` matches before `company.internal`).
- **Default Registry**: Empty string key `""` in the map represents the default registry for unmatched prefixes.
