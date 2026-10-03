## MODIFIED Requirements

### Requirement: Old test pins are current or frozen with a reason

Each `opmodel.dev/core@v2` or `opmodel.dev/catalogs/*` pin in a test input that tests resolve from a registry SHALL be moved to the release cascade's target, unless `.cascade-frozen` lists it with a reason.

- **The target.** A catalog pin's target is the newest published release in its major. A core pin's target is the core version that the input's catalog pins, or, for an input that pins no catalog, the core that `templates/minimal`'s catalog pins. The source is workspace RELEASING.md, section "The cascade", "Consistent set".
- **No backwards moves.** A core pin already newer than that target stays.
- **How a pin moves.** A `cue.mod` file moves by `cue mod get` then `cue mod tidy`, which `task deps:cascade` runs for the inputs it lists. A Go literal moves by editing the literal. A `cue.mod` file SHALL NOT be bumped by a hand edit.
- **Test inputs** include a test module's `cue.mod/module.cue` and a module file a test writes from a Go literal.
- **`.cascade-frozen` entries.** Each entry names the repo-relative file or directory, the module paths frozen there, and a one-sentence reason saying why the pin must stay old. The file uses the format in workspace RELEASING.md, section "Cascade files".

A pin is frozen only when the test depends on that exact old version. Examples:

- a platform that must be refused as too old;
- a platform that must show catalog version skew;
- a floor that the test re-pins up to.

A pin that is old only because a tree was copied from an older tree, or because an upstream release moved on, is not frozen. Between an upstream release and the release cascade that follows it, a pin is behind but not in breach.

A literal that a test only compares as expected output and never resolves from a registry (a golden) MAY also be listed, with a reason saying so, so that it is not mistaken for a stale pin.

#### Scenario: Every old core or catalog pin in a test module is accounted for

- **WHEN** the tracked `cue.mod/module.cue` files under `tests/` and `internal/` are read, and their `opmodel.dev/core@v2` and `opmodel.dev/catalogs/*` pins are compared with the cascade target (the newest published catalog, and the core that catalog pins)
- **THEN** each one that is older than its target is at or under a path listed in `.cascade-frozen`, with that module path among its pins

#### Scenario: A deliberately old pin carries its reason

- **WHEN** `.cascade-frozen` is read
- **THEN** every entry has a non-empty `path` that exists in the repo, a non-empty `pins` list, and a non-empty `reason`
- **AND** it lists `tests/e2e/instance_build_test.go` with `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4`, whose tests need an older core to be refused, the collision floor, and an older catalog to show version skew
- **AND** it lists `internal/cmd/platform/check_test.go` with `opmodel.dev/core@v2`, whose tests need older cores to be refused and the collision floor
- **AND** it lists `internal/instinit/render_test.go` with `opmodel.dev/core@v2`, whose golden asserts rendered text that carries the old core version

#### Scenario: Bumped test trees still pass

- **WHEN** the module vet, module eval, render, instance-init, skip-unprovided and duplicate-identity tests run against the bumped trees, with the canonical registry mapping, which maps both `testing.opmodel.dev` and `opmodel.dev` to `ghcr.io/open-platform-model`
- **THEN** they pass without skipping for an unresolvable core
