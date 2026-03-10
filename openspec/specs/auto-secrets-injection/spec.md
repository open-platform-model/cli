# Auto-Secrets Injection

## Purpose

**Superseded.** The Go-side auto-secrets injection has been eliminated. The CUE layer in `v1alpha1/` handles this declaratively.

## Removed Requirements

### Requirement: Go-side auto-secrets injection
**Reason**: The Go-side `builder/autosecrets.go` (which loaded `#Secrets` schema, built an `opm-secrets` component, and injected it into the components map) is eliminated. The CUE layer in `v1alpha1/` handles this declaratively: `#AutoSecrets` discovers `schemas.#Secret` fields in resolved config, `#GroupSecrets` groups them by secret name, and `#ModuleRelease` automatically injects an `opm-secrets` component via `helpers.#OpmSecretsComponent` when secrets exist.

**Migration**: No Go-side migration needed. Module authors continue using `schemas.#Secret` in their `#config` definitions. The CUE evaluation of `#ModuleRelease` handles discovery, grouping, and component injection automatically.
