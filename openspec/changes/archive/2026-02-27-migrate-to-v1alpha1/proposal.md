## Why

The OPM catalog has been redesigned as v1alpha1 (`opmodel.dev@v1`), introducing structural changes to core types, new metadata fields, renamed traits/resources, and a unified monorepo module structure. The CLI currently targets v0 schemas exclusively. To remain functional against the new catalog, every part of the CLI that references v0 schemas — Go types, CUE loader paths, builder, templates, test fixtures, and examples — must be updated. This is a **clean break**: v0 support is being removed entirely.

## What Changes

- **BREAKING**: All CUE import paths change from `@v0` to `@v1` (e.g., `opmodel.dev/core@v0` → `opmodel.dev/core@v1`)
- **BREAKING**: Module metadata structure changes — `metadata.apiVersion` removed, replaced by top-level `apiVersion: "opmodel.dev/core/v1alpha1"` and `metadata.cueModulePath`
- **BREAKING**: `metadata.name` is now kebab-case (RFC 1123 DNS label); PascalCase is auto-derived as `#definitionName` by CUE
- **BREAKING**: Trait `#Replicas` renamed to `#Scaling` (spec field: `scaling` with `count` subfield instead of flat `replicas`)
- **BREAKING**: Resource `#PersistentVolume` renamed to `#Volumes` (spec field: `volumes` map instead of `storage`)
- **BREAKING**: Container image is now a structured object `{repository, tag, digest}` instead of a plain string
- **BREAKING**: Container ports and env vars are now struct-keyed maps with explicit `name` fields
- **BREAKING**: Components using `#Container` must declare `"core.opmodel.dev/workload-type"` label
- Go `ModuleMetadata` type gains `CueModulePath` field; FQN computation changes
- Builder loads `opmodel.dev/core@v1` instead of `@v0` for `#ModuleRelease` schema
- Config defaults updated from `@v0` to `@v1` package paths
- All 3 init templates (simple, standard, advanced) rewritten for v1alpha1 schema
- All 9 example modules rewritten for v1alpha1 schema
- All test fixtures and Go test assertions updated

## Capabilities

### New Capabilities

- `v1alpha1-types`: Go type updates to `internal/core/` for v1alpha1 metadata fields (CueModulePath, workload-type label constant) and updated FQN computation in the loader
- `v1alpha1-templates`: Rewrite of all `opm mod init` templates (simple, standard, advanced) to generate v1alpha1-compatible CUE modules with structured images, renamed traits, and new metadata layout
- `v1alpha1-builder`: Update builder and config to load from `opmodel.dev/core@v1` and use v1alpha1 package paths
- `v1alpha1-fixtures`: Update all test fixtures, Go tests, and examples to target v1alpha1 schema

### Modified Capabilities

- `module-metadata-extraction`: Loader must extract `cueModulePath` instead of `apiVersion`/`fqn` from metadata, and compute FQN from `cueModulePath + "#" + definitionName`
- `release-building`: Builder changes the core schema import from `@v0` to `@v1`
- `config`: Default config template module path changes from `@v0` to `@v1`

## Impact

- **Packages**: `internal/core/module`, `internal/core/labels`, `internal/loader`, `internal/builder`, `internal/config`, `internal/templates` (all template files)
- **Tests**: Every `*_test.go` file that asserts on FQN strings, apiVersions, or metadata fields. All CUE test fixtures in `internal/loader/testdata/`, `internal/pipeline/testdata/`, `tests/fixtures/valid/`
- **Examples**: All 9 modules in `examples/` need full rewrites
- **Dependencies**: Requires `opmodel.dev@v1` to be resolvable in the CUE registry
- **SemVer**: This is a **MAJOR** change — all existing user modules on v0 will break and need migration
- **Not in scope**: `experiments/` directory (left on v0), `debugValues` pattern (separate change), new types like Blueprint/Bundle/Policy (added incrementally later)
