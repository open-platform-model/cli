# test-fixture-lineage

## Purpose

Keeps the repo's fixtures and examples on the current published schema line and its tests free of sibling-checkout dependencies. (Test-infrastructure capability; precedent: `validation-gates`, `kind-cluster-tasks`, `ci-workflow`.)

## Requirements

### Requirement: Maintained fixtures track the current schema line

Fixtures and examples consumed by tests or presented as current documentation SHALL import only the current published schema line (`opmodel.dev/core@v2` and the versioned `opmodel.dev/catalogs/opm@v2` packages). Artifacts kept for a retired line SHALL live under an explicitly marked legacy location with a note naming the line they document, and SHALL NOT be consumed by any test.

A maintained module fixture's identity package SHALL declare `Version` as a plain string literal with no default arm and no local `#VersionType`, and its `metadata.version` SHALL reference that literal directly. A fixture SHALL load through the kernel's module loader with no interpolation or other workaround in `metadata`.

#### Scenario: No retired-schema imports outside legacy

- **WHEN** the repo is grepped for retired-line imports (`opmodel.dev/core@v1`, `core/v1alpha1`, `modulerelease@v1`, `opm/v1alpha1`) outside the marked legacy location
- **THEN** there SHALL be no matches in fixtures, examples, or test inputs

#### Scenario: Vet fixtures exercise current schema

- **WHEN** the module vet tests run
- **THEN** their fixtures SHALL be `core@v2`-line modules exercising the same behaviors (valid module, secrets discovery, debug values) as before the port

#### Scenario: Fixture identity is a literal

- **WHEN** `tests/fixtures/modules/*/identity/identity.cue` is read
- **THEN** each declares `Version: "<semver>"` with no `|`, no `*` and no `#VersionType`
- **AND** the matching `module.cue` declares `version: id.Version`
- **AND** `opm module build` on the fixture passes the loader shape gate

### Requirement: Tests depend only on repo-local fixtures

No test or integration program SHALL read fixtures from a sibling repository checkout. Vendored copies SHALL carry a provenance header naming their origin.

#### Scenario: render-parity is self-contained

- **WHEN** the render-parity program runs in a standalone clone of this repo (no sibling checkouts)
- **THEN** it SHALL locate its module fixture under this repo's `tests/fixtures/` and proceed to the registry-gated comparison

### Requirement: Fixtures live on the testing domain and are published

A test fixture module SHALL declare a module path under `testing.opmodel.dev/modules/cli/`, and SHALL NOT declare one under `opmodel.dev/`. Each fixture tree SHALL carry an `identity/` package as the single source of its path and version, with `metadata.name`, `metadata.modulePath`, and `metadata.version` derived from it. Fixture trees SHALL be published to GHCR by repository CI through `opm module publish`, so every consumer — examples, e2e testdata, the kind dev cluster, a fresh clone — resolves them from a public registry with no local registry involved.

The rule is mechanical, not stylistic: CUE resolves modules by longest-prefix match on the module path, so a fixture declared under `opmodel.dev/` forces that entire prefix — core and the catalogs included — onto whatever registry serves the fixture.

#### Scenario: No fixture occupies the production namespace

- **WHEN** `tests/fixtures/modules/*/cue.mod/module.cue` is inspected
- **THEN** every declared `module:` path SHALL begin with `testing.opmodel.dev/modules/cli/`

#### Scenario: A fixture passes the publish gates

- **WHEN** `opm module publish --dry-run` runs over a fixture tree
- **THEN** the plan SHALL resolve with no refusals, deriving the registry repository and tag from the fixture's own identity package

#### Scenario: Consumers resolve fixtures without a local registry

- **WHEN** the examples, the e2e testdata, or an integration program resolves its fixture module with only the default GHCR registry mapping configured and no local registry running
- **THEN** resolution SHALL succeed

#### Scenario: A version bump is an identity edit

- **WHEN** a fixture's version changes
- **THEN** the edit SHALL be to that fixture's `identity/identity.cue`, and every consumer naming the version SHALL be re-pinned to match
