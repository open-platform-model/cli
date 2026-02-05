# Design: Multi-Registry Routing

## Research & Decisions

### Registry Routing Strategy

**Context**: Users need to consume modules from multiple registries (public, internal, air-gapped) with prefix-based routing.

**Explored**: CUE's [standard registry configuration](https://cuelang.org/docs/reference/command/cue-help-registryconfig/) pattern.

**Options considered**:

1. **Single registry URL** (current implementation)
   - Pros: Simple
   - Cons: Cannot support multi-registry environments

2. **Prefix-based routing map** (CUE_REGISTRY compatible)
   - Pros: Supports enterprise multi-registry, compatible with CUE tooling
   - Cons: Breaking change to existing config

**Decision**: Prefix-based routing map

**Rationale**: Enables enterprise use cases. CUE_REGISTRY compatibility ensures `opm` and `cue` behavior is identical. Breaking change is acceptable for MAJOR version bump.

---

## Technical Approach

### Configuration Format

Prefix-based routing via `config.registries` map, translated to `CUE_REGISTRY` format:

```cue
config: {
    registries: {
        "company.internal": {
            url: "harbor.internal/modules"
            insecure: true
        }
        "": {  // default registry
            url: "registry.opmodel.dev"
        }
    }
}
```

### CUE_REGISTRY Format

The `ToCUERegistry()` method converts the map to CUE_REGISTRY format:

```text
[modulePrefix=]hostname[:port][/repoPrefix][+insecure],...
```

Example output: `company.internal=harbor.internal/modules+insecure,registry.opmodel.dev`

Prefixes are sorted by length (longest first) for deterministic output and correct matching.

### Backward Compatibility

Single URL values in `OPM_REGISTRY` remain valid:

- `OPM_REGISTRY=localhost:5000` → treated as default registry (empty prefix)
- `OPM_REGISTRY=company.internal=harbor.internal,registry.opmodel.dev` → full CUE_REGISTRY format

## Data Model

### Registry Types (`internal/config/registry.go`)

```go
// RegistryEntry defines configuration for a single registry.
// Maps to CUE_REGISTRY format: hostname[:port][/repoPrefix][+insecure]
type RegistryEntry struct {
    URL      string // hostname[:port][/repoPrefix]
    Insecure bool   // Adds +insecure suffix
}

// RegistryMap maps module prefixes to registry configurations.
// Empty string key ("") represents the default registry.
type RegistryMap map[string]RegistryEntry

// ToCUERegistry converts to CUE_REGISTRY format string.
// Sorts prefixes by length (longest first) for deterministic output.
// Default registry (empty key) is always appended last.
//
// Example output: "company.internal/critical=harbor-critical.internal/modules+insecure,company.internal=harbor.internal/modules+insecure,registry.opmodel.dev"
func (m RegistryMap) ToCUERegistry() string

// ParseCUERegistry parses a CUE_REGISTRY format string into RegistryMap.
// Supports: [modulePrefix=]hostname[:port][/repoPrefix][+insecure]
// Single URL without prefix is treated as default registry.
func ParseCUERegistry(s string) (RegistryMap, error)
```

## Error Handling

| Error | When | Message |
|-------|------|---------|
| Invalid CUE_REGISTRY format | Parsing env var | "invalid CUE_REGISTRY format: {detail}" |
| Duplicate prefix | Parsing with same prefix twice | "duplicate registry prefix: {prefix}" |
| Empty URL | Registry entry with no URL | "registry URL cannot be empty for prefix: {prefix}" |

## File Changes

- `cli/internal/config/registry.go` - New file with RegistryEntry, RegistryMap, conversions
- `cli/internal/config/config.go` - Update Config struct: `Registry string` → `Registries RegistryMap`
- `cli/internal/config/loader.go` - Update bootstrap and loading logic
- `cli/internal/config/resolver.go` - Update ResolveRegistry to handle RegistryMap
- `cli/internal/config/templates.go` - Update default config template
