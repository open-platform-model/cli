## Context

The OPM CLI requires configuration for registry resolution, Kubernetes connectivity, and caching. Per the constitution, configuration uses CUE-native format (`~/.opm/config.cue`) rather than YAML/TOML to maintain type safety (Principle I) and use a single language for all configuration.

**Current State**: Fully implemented in `cli/internal/config/` with:

- Two-phase loading (bootstrap registry extraction → full CUE evaluation)
- Precedence-based resolution (flag > env > config > default)
- Commands: `config init`, `config vet`

**Stakeholders**: All CLI users; Platform Operators configuring registries and Kubernetes contexts.

## Goals / Non-Goals

**Goals:**

- Document the two-phase loading architecture and its rationale
- Specify precedence rules for all configuration values
- Define the CUE schema contract for `config.cue`
- Establish error handling patterns for config failures

**Non-Goals:**

- Changing the existing implementation behavior
- Adding new configuration fields
- Supporting non-CUE config formats (YAML, TOML, JSON)

## Decisions

### D1: Two-Phase Loading Architecture

**Decision**: Config loading uses two phases:

1. **Bootstrap**: Extract `registry` field via regex without full CUE parsing
2. **Full Load**: Set `CUE_REGISTRY` and evaluate config with import resolution

**Rationale**: The config file may import provider definitions from the registry (e.g., `import prov "opmodel.dev/providers@v0"`). To resolve these imports, CUE needs `CUE_REGISTRY` set. But the registry URL is defined *in* the config. Bootstrap extraction breaks this circular dependency.

**Alternatives Considered**:

- Require registry via env var only → Rejected: poor UX, forces external config
- Separate registry-only file → Rejected: adds complexity, splits config

### D2: CUE-Native Configuration

**Decision**: Use `~/.opm/config.cue` with CUE schema validation, not viper/YAML.

**Rationale**:

- Aligns with Principle I (Type Safety) - config validated at load time
- Single language for modules and CLI config
- CUE constraints prevent invalid values (e.g., `namespace: *"default" | string`)

**Trade-off**: Users must learn CUE syntax. Mitigated by `config init` generating valid config with comments.

### D3: Precedence Chain

**Decision**: Resolution order is Flag > Environment > Config > Default.

**Rationale**: Standard 12-factor app pattern. Flags for one-off overrides, env vars for CI/deployment, config file for persistent settings.

| Value | Flag | Env Var | Config Field |
|-------|------|---------|--------------|
| Registry | `--registry` | `OPM_REGISTRY` | `config.registry` |
| Config Path | `--config` | `OPM_CONFIG` | (n/a) |
| Kubeconfig | `--kubeconfig` | `OPM_KUBECONFIG` | `kubernetes.kubeconfig` |
| Context | `--context` | `OPM_CONTEXT` | `kubernetes.context` |
| Namespace | `--namespace` | `OPM_NAMESPACE` | `kubernetes.namespace` |

### D4: Fail-Fast on Missing Registry with Providers

**Decision**: If config references providers but no registry is resolvable, fail immediately with actionable error.

**Rationale**: Provider imports require registry. Letting CUE fail with cryptic import errors is poor UX.

## Risks / Trade-offs

**[Risk] Bootstrap regex extraction is fragile**
→ Mitigation: Simple pattern (`registry: "..."`) covers 99% of cases. Edge cases (computed registry) can use env var.

**[Risk] CUE learning curve for users**
→ Mitigation: `config init` generates valid config with extensive comments. `config vet` validates before use.

**[Trade-off] No hot-reload of config**
→ Accepted: Config is read once at CLI startup. Simplifies implementation. Users restart CLI for changes.

**[Trade-off] Provider definitions loaded from registry**
→ Accepted: Enables versioned provider schemas. Requires network at startup if providers configured.
