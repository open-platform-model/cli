## Context

The OPM CLI targets the v0 CUE catalog (`opmodel.dev@v0`), which was structured as separate modules (`opmodel.dev/core@v0`, `opmodel.dev/resources@v0`, `opmodel.dev/traits@v0`, etc.). The v1alpha1 catalog consolidates everything into a single monorepo module (`opmodel.dev@v1`) with subpackages (`core`, `resources/workload`, `traits/workload`, etc.).

The CLI has ~409 references to `@v0` across ~80 files spanning Go source, templates, test fixtures, and examples. The migration touches every layer: Go types, CUE loader paths, the builder's core schema import, init templates, test fixtures, and example modules.

Key structural changes in v1alpha1:
- `apiVersion` and `kind` are top-level fields on `#Module` (not inside `metadata`)
- `metadata.apiVersion` is replaced by `metadata.modulePath` (plain registry path, no version)
- `metadata.name` is kebab-case; PascalCase is auto-derived as `#definitionName`
- Module FQN format: `modulePath/name:semver` (`#ModuleFQNType`) — e.g., `"opmodel.dev/modules/my-app:1.0.0"`
- Primitive FQN format: `modulePath/name@majorVersion` (`#FQNType`) — e.g., `"opmodel.dev/resources/workload/container@v1"`
- `metadata.fqn` is a computed field — extractable directly from CUE evaluation
- Container images are structured objects `{repository, tag, digest}`
- Ports/env are struct-keyed maps with explicit `name` fields
- `#Replicas` is renamed to `#Scaling` (with `count` subfield)
- `#PersistentVolume` is renamed to `#Volumes`
- Components require a `"core.opmodel.dev/workload-type"` label

## Goals / Non-Goals

**Goals:**
- Update all Go types to model v1alpha1 metadata fields
- Update the loader to extract v1alpha1 metadata (`modulePath`, `fqn`, `version`)
- Update the builder to load `opmodel.dev/core@v1` for `#ModuleRelease`
- Rewrite all 3 init templates (simple, standard, advanced) for v1alpha1 schema
- Update all test fixtures and Go test assertions
- Rewrite all 9 example modules for v1alpha1
- Remove all `@v0` references from production code

**Non-Goals:**
- Adding Go type support for new v1alpha1 types (Blueprint, Bundle, Policy, PolicyRule) — done incrementally later
- Changing the `values.cue` loading pattern to `debugValues` — separate change
- Updating the `experiments/` directory — left on v0
- Backward compatibility or dual v0/v1alpha1 support — clean break
- Updating integration tests that require a live cluster with v1 registry

## Decisions

### D1: User module version stays `@v0`

User modules created by `opm mod init` keep `module: "{{.ModulePath}}@v0"` in their `cue.mod/module.cue`. The `@v0` here is the user's own module version (starting at major version 0), not the catalog version. Catalog imports change to `@v1`.

**Alternatives considered:** Using `@v1` to match catalog — rejected because the user's module version is independent of the catalog version they import.

### D2: FQN extracted directly from CUE evaluation

v1alpha1 `#Module` has `metadata.fqn` as a computed field:
- Module FQN: `"\(modulePath)/\(name):\(version)"` — e.g., `"opmodel.dev/modules/my-app:1.0.0"`
- Primitive FQN: `"\(modulePath)/\(name)@\(version)"` — e.g., `"opmodel.dev/resources/workload/container@v1"`

The loader will extract `metadata.fqn` directly from the evaluated CUE value. No Go-side FQN computation is needed — CUE computes it from `modulePath`, `name`, and `version`.

`#definitionName` (PascalCase of name) is still available via `cue.Hid` but is NOT used in the FQN. It may be extracted for display purposes if needed.

**Alternatives considered:** Computing FQN in Go — rejected because CUE already computes it, and extracting a concrete string is simpler and less error-prone than duplicating the logic.

### D3: Lightweight test fixtures kept inline

Test fixtures in `internal/loader/testdata/`, `internal/pipeline/testdata/`, and `tests/fixtures/` will continue to define inline `#resources`/`#traits` maps without importing the real catalog. They will be updated to use v1alpha1-compatible FQN keys (e.g., `"opmodel.dev/resources/workload/container@v1"`) and metadata structure, but won't import `opmodel.dev/core@v1`.

**Alternatives considered:** Rewriting fixtures to import real catalog — rejected because it adds registry dependency to unit tests and increases test setup complexity.

### D4: Templates use full structured image format

All init templates will generate the full `{repository, tag, digest}` image structure in both `#config` schema and `values.cue`. The `#config` image field will use `schemas.#Image` type from the v1alpha1 catalog.

### D5: Template `TemplateData.ModuleNamePascal` field retained but usage changes

The `ModuleNamePascal` field in `TemplateData` is kept for backward compatibility with any external consumers, but templates will use `{{.ModuleName}}` (kebab-case) for `metadata.name` since v1alpha1 auto-derives PascalCase via `#definitionName`.

## Risks / Trade-offs

- **Registry availability** — The v1 packages must be resolvable via `OPM_REGISTRY`/`CUE_REGISTRY`. If `opmodel.dev@v1` is not yet published, examples and integration tests will fail.
  → Mitigation: Verify registry publishing before merging. Unit tests use inline fixtures and don't need the registry.

- **FQN format change** — Module FQN (`path/name:semver`) and primitive FQN (`path/name@version`) use different separators. Go code must handle both formats correctly.
  → Mitigation: The loader reads FQN from CUE; it doesn't need to know the format. The builder and inventory code that compares FQNs will be updated to expect the new formats.

- **Blast radius** — Touching 80+ files in one change increases merge conflict risk and review difficulty.
  → Mitigation: Mechanical changes (import path `@v0` → `@v1`) are straightforward to verify. The structural changes (metadata layout, trait renames) are concentrated in templates and fixtures.

- **Example modules may not validate** — Examples rewritten to v1alpha1 can't be verified without the v1 registry.
  → Mitigation: Mark examples as needing `cue mod tidy` with proper registry setup. Verify structure manually against v1alpha1 catalog examples.
