## ADDED Requirements

### Requirement: Old test pins are current or frozen with a reason

Every test input that the tests resolve from a registry SHALL pin `opmodel.dev/core@v2` and each `opmodel.dev/catalogs/*` module it depends on at the newest published release, or SHALL be listed in the repo-root `.cascade-frozen`. Test inputs include a test module's `cue.mod/module.cue` and a module file a test writes from a Go literal. Each `.cascade-frozen` entry names the repo-relative file or directory, the module paths frozen there, and a one-sentence reason saying why the pin must stay old. The file uses the format in workspace RELEASING.md, section "Cascade files". An old pin with no entry is stale and SHALL be bumped with `cue mod get` followed by `cue mod tidy`, never by a hand edit.

A pin is frozen only when the test depends on that exact old version. Examples are a platform that must be refused as too old, or a floor that the test re-pins up to. A pin that is old only because a tree was copied from an older tree is not frozen.

#### Scenario: Every old core pin in a test module is accounted for

- **WHEN** the tracked `cue.mod/module.cue` files under `tests/` and `internal/` are read and their `opmodel.dev/core@v2` pins compared with the newest published core release
- **THEN** each one that is older is at or under a path listed in `.cascade-frozen` with `opmodel.dev/core@v2` among its pins

#### Scenario: A deliberately old pin carries its reason

- **WHEN** `.cascade-frozen` is read
- **THEN** every entry has a non-empty `path` that exists in the repo, a non-empty `pins` list, and a non-empty `reason`
- **AND** it lists `tests/e2e/instance_build_test.go` and `internal/cmd/platform/check_test.go` with `opmodel.dev/core@v2`, the two files whose tests need an older core to be refused or need the collision floor

#### Scenario: Bumped test trees still pass

- **WHEN** the module vet, module eval, render, instance-init and duplicate-identity tests run against the bumped trees, with the CI registry mapping (`opmodel.dev=ghcr.io/open-platform-model`)
- **THEN** they pass without skipping for an unresolvable core
