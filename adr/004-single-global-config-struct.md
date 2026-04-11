# ADR-004: Single GlobalConfig Struct

## Status

Accepted

## Context

The CLI resolves configuration from multiple sources: config file, environment variables, CLI flags, and loaded CUE providers. Previously, this was modeled as three nested layers (`Config` -> `OPMConfig` -> `cmdtypes.GlobalConfig`), each wrapping the next. The nesting created verbose access paths (e.g., `cfg.OPMConfig.Config.Kubernetes.Namespace`) and made it unclear which layer owned which data. The config struct is threaded through every command constructor (see ADR-003), so its ergonomics affect the entire codebase.

## Decision

Replace the three-layer nesting with a single flat `GlobalConfig` struct in `internal/config`. The struct contains: `Kubernetes` (`KubernetesConfig`), `Log` (`LogConfig`), `Registry` (resolved registry URL after flag > env > config precedence), `ConfigPath` (resolved config file path), `Providers` (`map[string]cue.Value` for loaded CUE provider definitions), `CueContext` (shared CUE evaluation context), and `Flags` (`GlobalFlags` substruct holding raw CLI flag values).

The `GlobalFlags` substruct (`Config`, `Registry`, `Verbose`, `Timestamps`) distinguishes raw flag values from resolved runtime state. Remove the intermediate `Config` and `OPMConfig` types entirely. Do not track `RegistrySource` on the config struct — the resolution source is not needed after resolution completes. Ensure no import cycles exist in the dependency graph between config and consuming packages.

## Consequences

**Positive:** Simpler access paths throughout the codebase (e.g., `cfg.Kubernetes.Namespace` instead of `cfg.OPMConfig.Config.Kubernetes.Namespace`).

**Positive:** Clear separation between raw flag values (`cfg.Flags.Registry`) and resolved state (`cfg.Registry`).

**Positive:** No import cycles — config package remains leaf-level.

**Negative:** Breaking change for any code importing the removed `Config` or `OPMConfig` types.

**Trade-off:** Flat struct is less extensible if config categories multiply, but the current set of fields is stable.
