# Delta for CLI Config Registries

## MODIFIED Requirements

### Requirement: Configuration precedence chain

The precedence table is MODIFIED to support registries map:

| Value | Flag | Env Var | Config Field | Default |
|-------|------|---------|--------------|---------|
| Registry | `--registry` | `OPM_REGISTRY` | `config.registries` | (none) |

Note: `OPM_REGISTRY` accepts CUE_REGISTRY format (comma-separated prefix routing). Single URL values remain valid as a default registry for backward compatibility.

---

## ADDED Requirements

### Requirement: Multi-Registry Routing

The CLI MUST support multi-registry routing via `config.registries` prefix-to-registry map.

#### Scenario: Prefix-based routing

- GIVEN config with `company.internal` prefix mapped to `harbor.internal`
- WHEN the CLI resolves a module path starting with `company.internal/`
- THEN the module is resolved from `harbor.internal`

#### Scenario: Default registry

- GIVEN config with empty key `""` as default registry
- WHEN the CLI resolves a module path with no matching prefix
- THEN the default registry is used

#### Scenario: Longest prefix wins

- GIVEN config with `company.internal` and `company.internal/critical` prefixes
- WHEN the CLI resolves `company.internal/critical/module`
- THEN the more specific `company.internal/critical` prefix is used

#### Scenario: Environment override with CUE_REGISTRY format

- GIVEN `OPM_REGISTRY=company.internal=harbor.internal,registry.opmodel.dev`
- WHEN the CLI loads configuration
- THEN environment variable takes precedence over config file
- THEN routing follows the prefix rules

#### Scenario: Single URL backward compatibility

- GIVEN `OPM_REGISTRY=localhost:5000` (single URL without prefix)
- WHEN the CLI loads configuration
- THEN it is treated as the default registry (empty prefix)
- THEN existing workflows continue to work

#### Scenario: Insecure registry support

- GIVEN config with `insecure: true` for a registry entry
- WHEN the CLI converts to CUE_REGISTRY format
- THEN the `+insecure` suffix is appended to that registry

---

## REMOVED Requirements

### Requirement: Single registry field

The `config.registry` single string field is REMOVED and replaced by `config.registries` map.

---

## Breaking Changes

- **BC-001**: The `config.registry` field (single string) is REMOVED and replaced by `config.registries` (prefix-to-registry map). Users must migrate their config.cue files.
- **BC-002**: `OPM_REGISTRY` now accepts `CUE_REGISTRY` format (comma-separated prefix routing) instead of a single URL. Single URL values remain valid for backward compatibility.

---

## Key Entities

- **RegistryEntry**: Configuration for a single registry (URL + insecure flag).
- **RegistryMap**: Map from module prefixes to RegistryEntry. Empty string key represents the default registry.

---

## Success Criteria

- **SC-001**: Existing single-registry users can migrate by changing `registry: "url"` to `registries: { "": { url: "url" } }`.
- **SC-002**: `ToCUERegistry()` output is 100% compatible with CUE CLI's `CUE_REGISTRY` format.
- **SC-003**: Single URL in `OPM_REGISTRY` env var continues to work (backward compatibility).
