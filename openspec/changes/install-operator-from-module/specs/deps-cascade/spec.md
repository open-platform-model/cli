## MODIFIED Requirements

### Requirement: The cascade task moves the cli's upstream pins to published versions

`task -x deps:cascade` SHALL move the following pins in the working tree only. It SHALL NOT commit, branch or push.

- **library.** `github.com/open-platform-model/library` in `go.mod` SHALL move to the newest version the Go proxy serves in its major, through `go get <module>@<exact version>` followed by `go mod tidy`.
- **opm-operator module.** The operator module pin in `internal/operator/pin.go` (module version, content digests and `PinnedOperatorVersion`) SHALL move only through `task operator:pin VERSION=<v>`. The target is the newest published release of `opmodel.dev/modules/opm_operator` in the pinned major whose deployed operator `MAJOR.MINOR` is not above the cli's own, the same rule install applies to a target module version.
- **Catalog and core as a consistent set.** This covers the `cue.mod/module.cue` files of the three templates, `hack/platform`, `examples`, `tests/fixtures/modules/podinfo`, `tests/e2e/testdata/operator-owned`, `internal/instinit/testdata/initvalues`, `internal/workflow/render/testdata/skip-unprovided`, `tests/e2e/testdata/duplicate-identities`, `tests/integration/module-apply/testdata`, `tests/fixtures/valid/simple-module` and `tests/fixtures/valid/module-with-debug-values`.
  - `opmodel.dev/catalogs/opm@v4` SHALL move to the newest published catalog in its major, resolved against `templates/minimal`'s catalog.
  - `opmodel.dev/core@v2` SHALL move to the core version that the file's resulting catalog pins. A file without a catalog SHALL use `templates/minimal`'s resulting catalog.
- **Kind Platform.** The bare `version:` under `opmodel.dev/catalogs/opm@v4:` in `hack/kind-platform.yaml` SHALL be raised to `hack/platform`'s catalog after the move when it is lower. A version above it SHALL stay, with a warning, since no pin moves backwards.

How every move behaves:

- No pin SHALL move backwards. No pin SHALL move past an in-date `.cascade-hold` `max`. When a hold on core is below the core a newer catalog pins, the catalog SHALL stay too, with a warning.
- No pin SHALL move to a version that is tagged but not yet published.
- No pin listed for a file in `.cascade-frozen` SHALL change in that file. If `cue mod tidy` raises such a pin, the task SHALL fail, naming the file and the key.
- `cue mod get` SHALL name only the moved `opmodel.dev` keys, at exact versions. It and `cue mod tidy` SHALL run only in a module where a pin moved.
- Third-party pins, including `cue.dev/x/k8s.io@v0`, SHALL NOT be named. A third-party pin that `tidy` or `go mod tidy` raised SHALL be reported as a warning.
- The task SHALL use its own registry mapping, `testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`, for both `CUE_REGISTRY` and `OPM_REGISTRY`, whatever the environment sets.
- The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: Older pins move to the published set

- **WHEN** library, the operator module, the catalog and core are older than the newest published versions, and `task -x deps:cascade` runs
- **THEN** `go.mod`, `go.sum`, `internal/operator/pin.go`, every listed `cue.mod/module.cue` and `hack/kind-platform.yaml` carry the resolved versions
- **AND** core in each file equals the core that file's catalog pins
- **AND** the task exits 0

#### Scenario: Core already ahead of the catalog's pin stays

- **WHEN** a file pins a core newer than the core its catalog pins
- **THEN** that file's core is unchanged, and a warning names both versions

#### Scenario: A kind catalog ahead of hack/platform stays

- **WHEN** `hack/kind-platform.yaml` pins a catalog newer than `hack/platform`'s, and nothing else moves
- **THEN** the file is unchanged, the task exits 3, and a warning names both versions

#### Scenario: A hold on core holds the catalog

- **WHEN** `.cascade-hold` holds `opmodel.dev/core@v2` in date at the tree's core, and the newest catalog pins a newer core
- **THEN** no catalog or core pin changes, the task exits 3, and a warning says the catalog is held too

#### Scenario: An unpublished newest tag leaves the pin

- **WHEN** the newest catalog tag is not yet served by GHCR, and every older tag is not newer than the current pin
- **THEN** no catalog pin changes

#### Scenario: A frozen file is left byte-unchanged

- **WHEN** `.cascade-frozen` lists a moved test tree's `cue.mod/module.cue` for both `opmodel.dev` keys, and the other pins are older
- **THEN** that file is byte-unchanged, the other files move, and the task exits 0

#### Scenario: The third-party k8s pin is never named

- **WHEN** the catalog moves in `hack/platform`
- **THEN** `cue mod get` names only the `opmodel.dev` keys, and `cue.dev/x/k8s.io@v0` changes only if `cue mod tidy` raised it, in which case a warning names it

