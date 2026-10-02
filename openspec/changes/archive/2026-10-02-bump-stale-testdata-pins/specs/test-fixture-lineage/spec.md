## ADDED Requirements

### Requirement: Old test pins are current or frozen with a reason

Each `opmodel.dev/core@v2` or `opmodel.dev/catalogs/*` pin in a test input that tests resolve from a registry SHALL be moved to the newest published release (a `cue.mod` file by `cue mod get` then `cue mod tidy`, a Go literal by editing the literal) unless `.cascade-frozen` lists it with a reason. Test inputs include a test module's `cue.mod/module.cue` and a module file a test writes from a Go literal. A `cue.mod` file SHALL NOT be bumped by a hand edit. Each `.cascade-frozen` entry names the repo-relative file or directory, the module paths frozen there, and a one-sentence reason saying why the pin must stay old. The file uses the format in workspace RELEASING.md, section "Cascade files".

A pin is frozen only when the test depends on that exact old version. Examples are a platform that must be refused as too old, a platform that must show catalog version skew, or a floor that the test re-pins up to. A pin that is old only because a tree was copied from an older tree, or because an upstream release moved on, is not frozen. Between an upstream release and the release cascade that follows it, a pin is behind but not in breach. A literal that a test only compares as expected output and never resolves from a registry (a golden) MAY also be listed, with a reason saying so, so that it is not mistaken for a stale pin.

#### Scenario: Every old core or catalog pin in a test module is accounted for

- **WHEN** the tracked `cue.mod/module.cue` files under `tests/` and `internal/` are read and their `opmodel.dev/core@v2` and `opmodel.dev/catalogs/*` pins compared with the newest published release of each module
- **THEN** each one that is older is at or under a path listed in `.cascade-frozen` with that module path among its pins

#### Scenario: A deliberately old pin carries its reason

- **WHEN** `.cascade-frozen` is read
- **THEN** every entry has a non-empty `path` that exists in the repo, a non-empty `pins` list, and a non-empty `reason`
- **AND** it lists `tests/e2e/instance_build_test.go` with `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4`, whose tests need an older core to be refused, the collision floor, and an older catalog to show version skew
- **AND** it lists `internal/cmd/platform/check_test.go` with `opmodel.dev/core@v2`, whose tests need older cores to be refused and the collision floor
- **AND** it lists `internal/instinit/render_test.go` with `opmodel.dev/core@v2`, whose golden asserts rendered text that carries the old core version

#### Scenario: Bumped test trees still pass

- **WHEN** the module vet, module eval, render, instance-init, skip-unprovided and duplicate-identity tests run against the bumped trees, with the canonical registry mapping, which maps both `testing.opmodel.dev` and `opmodel.dev` to `ghcr.io/open-platform-model`
- **THEN** they pass without skipping for an unresolvable core

## MODIFIED Requirements

### Requirement: Maintained fixtures track the current schema line

Fixtures and examples consumed by tests or presented as current documentation SHALL import only the current published schema line (`opmodel.dev/core@v2` and the versioned `opmodel.dev/catalogs/opm@v4` packages). Artifacts kept for a retired line SHALL live under an explicitly marked legacy location with a note naming the line they document, and SHALL NOT be consumed by any test.

A maintained module fixture's identity package SHALL declare `Version` as a plain string literal with no default arm and no local `#VersionType`, and its `metadata.version` SHALL reference that literal directly. A fixture SHALL load through the kernel's module loader with no interpolation or other workaround in `metadata`.

A fixture SHALL NOT depend on a catalog mechanism the catalog has withdrawn. When a catalog member or contract type a fixture exercises is removed from the published catalog, the fixture is removed with it rather than kept unvetted; a replacement fixture is added when the replacement mechanism exists.

#### Scenario: No retired-schema imports outside legacy

- **WHEN** the repo is grepped for retired-line imports (`opmodel.dev/core@v1`, `core/v1alpha1`, `modulerelease@v1`, `opm/v1alpha1`) outside the marked legacy location
- **THEN** there SHALL be no matches in fixtures, examples, or test inputs

#### Scenario: Vet fixtures exercise current schema

- **WHEN** the module vet tests run
- **THEN** their fixtures SHALL be `core@v2`-line modules exercising the same behaviors (valid module, debug values) as before the port

#### Scenario: No fixture exercises a withdrawn catalog mechanism

- **WHEN** `tests/fixtures/` is grepped for the legacy secret vocabulary (`res.#Secret`, `$secretName`, `$dataKey`, `#AutoSecrets`, `opm-secrets`)
- **THEN** there SHALL be no matches

#### Scenario: Fixture identity is a literal

- **WHEN** `tests/fixtures/modules/*/identity/identity.cue` is read
- **THEN** each declares `Version: "<semver>"` with no `|`, no `*` and no `#VersionType`
- **AND** the matching `module.cue` declares `version: id.Version`
- **AND** `opm module build` on the fixture passes the loader shape gate
