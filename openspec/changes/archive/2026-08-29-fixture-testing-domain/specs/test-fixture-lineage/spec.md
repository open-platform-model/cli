# test-fixture-lineage — Delta

## ADDED Requirements

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
