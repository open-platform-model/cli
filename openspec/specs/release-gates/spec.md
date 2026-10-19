# release-gates Specification

## Purpose
Gates that run on the cli's release-please pull requests, so a release never ships a local, unpublished or mismatched upstream pin, and a moved opm-operator module pin is released only with evidence that the end-to-end suite ran against it.

## Requirements

### Requirement: The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement

A release-pin check (G1) SHALL fail, naming every offending pin and its file, when any of the following holds in the tree under test:

- `go.mod` carries a `replace` directive;
- a `github.com/open-platform-model/*` requirement in `go.mod` is a Go pseudo-version, or names a version for which the module's repository has no tag.

**Source**: workspace RELEASING.md, section "Gates", enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Pseudo-version library pin fails the release PR
- **WHEN** a release-please PR's `go.mod` requires `github.com/open-platform-model/library` at a pseudo-version
- **THEN** the release-pin step exits non-zero naming the module, the version and `go.mod`

#### Scenario: Tag lookup failure is not a pass
- **WHEN** the tag lookup for an OPM Go pin, or the registry lookup for the operator module pin, fails for a reason other than a missing tag or version
- **THEN** the release-pin step exits non-zero and reports a lookup failure for that pin

### Requirement: The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`

A release-pin check (G1) SHALL fail, naming every offending pin and its file, when any of the following holds in the tree under test:

- a template `cue.mod/module.cue` under `templates/` pins a dependency at a `-0.dev.` version;
- a `cue.mod/local-module.cue` file is tracked anywhere in the repository.

**Source**: workspace RELEASING.md, section "Gates", enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: A template pin and a replace directive are reported together
- **WHEN** a release-please PR carries a `replace` directive and a template pinned at a `-0.dev.` core version
- **THEN** the release-pin step exits non-zero and its output names both violations

### Requirement: The release-pin check fails on an unserved or mismatched operator module pin

A release-pin check (G1) SHALL fail, naming every offending pin and its file, when any of the following holds in the tree under test:

- the operator module pin in `internal/operator/pin.go` names a module version the registry does not serve, or records a `PinnedOperatorVersion` other than the operator version the pinned module's source states it deploys, or the pin cannot be read.

**Source**: workspace RELEASING.md, section "Gates", enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Recorded operator version out of step with the pinned module
- **WHEN** `PinnedOperatorVersion` names one operator release and the pinned operator module states it deploys another
- **THEN** the release-pin step exits non-zero naming both versions and `task operator:pin VERSION=<module version>` as the fix

#### Scenario: Pinned module version not served
- **WHEN** the operator module pin names a module version the registry does not serve
- **THEN** the release-pin step exits non-zero naming `internal/operator/pin.go`, the module and its version

### Requirement: The release-pin check reports all failures in one run and exits zero when none holds

The check SHALL report all failures in one run, not only the first, and SHALL exit zero when none holds.
(Requirements: The release-pin check fails on a `replace` directive and on a pseudo-version or untagged OPM Go requirement; The release-pin check fails on a template `-0.dev.` pin and on a tracked `local-module.cue`; The release-pin check fails on an unserved or mismatched operator module pin)

**Source**: workspace RELEASING.md, section "Gates", enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Clean release PR passes
- **WHEN** a release-please PR's tree has no replace, no tracked `local-module.cue`, library pinned at an existing tag, no dev template pin, and an operator module pin whose module version the registry serves and whose operator version that module states
- **THEN** the release-pin step exits zero and the `lint` job's result is decided by golangci-lint alone

#### Scenario: Several violations are reported together
- **WHEN** a release-please PR carries a `replace` directive and a template pinned at a `-0.dev.` core version
- **THEN** the release-pin step exits non-zero and its output names both violations

### Requirement: The release-pin check runs as a step of the `lint` job on release-please branches only

A release-pin check (G1) SHALL run as a step inside the existing `lint` job of both the pull-request workflow and the push workflow, and only when the branch name (`head_ref`, or `ref_name` when `head_ref` is empty) starts with `release-please--`; on every other branch the step SHALL be skipped while the job still runs.

**Source**: workspace RELEASING.md, section "Gates", enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: Ordinary PR is not gated
- **WHEN** a pull request from any branch not starting with `release-please--` carries a pseudo-version library pin
- **THEN** the release-pin step is skipped and the `lint` job does not fail on account of the pin

#### Scenario: Release branch push is gated
- **WHEN** the release App pushes to `release-please--branches--main--components--opm` and the push workflow runs
- **THEN** the push workflow's `lint` job runs the release-pin step against that tree

### Requirement: `task deps:release-check` runs the release-pin check locally on any branch

`task deps:release-check` SHALL run the same check locally on any branch.
(Requirement: The release-pin check runs as a step of the `lint` job on release-please branches only)

**Source**: workspace RELEASING.md, section "Gates", enhancement 0021 ("The operator's install artifact is a registry module on its own version train").

#### Scenario: The task runs the check on any branch

- **WHEN** `task deps:release-check` runs on any branch
- **THEN** it SHALL run the same check locally

### Requirement: The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit

`pr.yml`'s `lint` job SHALL run the step "Docs bundles for moved pins" on every pull request. The step SHALL fetch the pull request's base commit and run `.github/scripts/docs-pins-check.sh --warn --moved-from <base sha>`. The step runs with the job's read-only token.

**Source**: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

#### Scenario: The step runs on every pull request

- **WHEN** a pull request is opened
- **THEN** the `lint` job SHALL run the step "Docs bundles for moved pins"

### Requirement: The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's

When the `github.com/open-platform-model/library` version in `go.mod` and `PinnedOperatorVersion` in `internal/operator/pin.go` (in `internal/operator/manifest.go` at a base from before the operator module pin) both equal the base's, the step SHALL check nothing and say so; a move of `PinnedModuleVersion` alone changes no docs pin.
(Requirement: The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit)

**Source**: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

#### Scenario: An ordinary pull request checks nothing

- **WHEN** a pull request changes neither the library version nor `PinnedOperatorVersion`
- **THEN** the step builds nothing, prints that both are unchanged, and exits zero

### Requirement: The docs bundle step runs the dump program on the tree and looks up each printed pin anonymously

Otherwise the step SHALL run `go run ./hack/docskit-dump pins` on the tree and look up each printed pin anonymously at `ghcr.io/open-platform-model/docs/<project>:<pin>`.
(Requirement: The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's)

**Source**: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

#### Scenario: The dump program runs when a pin moved

- **WHEN** the cascade pull request moves library to a version whose docs bundle is not published
- **THEN** the `lint` job shows a warning naming `library`, the version and the bundle reference, and the job's result is decided by its other steps

### Requirement: The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete

Otherwise the step SHALL print a GitHub warning annotation, also written to the job summary, for each bundle that is missing (401, 403 or 404), each lookup that fails, a program that does not build, a base commit that cannot be fetched, and a library version or `PinnedOperatorVersion` it cannot read on either side (an empty read never counts as unchanged).
(Requirement: The docs bundle step checks nothing when the library version and `PinnedOperatorVersion` equal the base's)

**Source**: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

#### Scenario: A cascade pull request moves library to a version without a bundle

- **WHEN** the cascade pull request moves library to a version whose docs bundle is not published
- **THEN** the `lint` job shows a warning naming `library`, the version and the bundle reference, and the job's result is decided by its other steps

#### Scenario: An unreadable pin is a warning, not a pass

- **WHEN** `go.mod` or `internal/operator/pin.go` changes shape so the step reads no library version or no `PinnedOperatorVersion`
- **THEN** the step prints a warning that the bundles were not checked, instead of saying both are unchanged

### Requirement: The docs bundle step bounds each GHCR request and has a timeout and `continue-on-error: true`

Each GHCR request SHALL be bounded (`--connect-timeout 10 --max-time 30`) and the step SHALL have `timeout-minutes: 10` and `continue-on-error: true`, because the `lint` job is a required check.
(Requirement: The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit)

**Source**: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

#### Scenario: Each GHCR request is bounded

- **WHEN** the step makes a GHCR request
- **THEN** each GHCR request SHALL be bounded (`--connect-timeout 10 --max-time 30`)

### Requirement: The docs bundle step exits zero in every case that prints a warning and so never fails the job

The step SHALL exit zero in every one of those cases, so it never fails the job: the blocking check stays G1 on the release pull request.
(Requirement: The docs bundle step prints a GitHub warning annotation for each missing bundle and each lookup or read it cannot complete)

**Source**: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

#### Scenario: A warning never fails the job

- **WHEN** the cascade pull request moves library to a version whose docs bundle is not published
- **THEN** the `lint` job shows a warning naming `library`, the version and the bundle reference, and the job's result is decided by its other steps

### Requirement: G1 runs the docs pins script without `--warn` and counts each printed line as a violation

G1 (`release-pin-check.sh`) SHALL run the same script without `--warn` and report each line it prints as a G1 violation; a non-zero exit with no line SHALL itself be a violation.
(Requirement: The `lint` job of `pr.yml` runs the step "Docs bundles for moved pins" on every pull request and fetches the base commit)

**Source**: security pass 2026-10-04, finding CAS-R2 (the check left the cascade task).

#### Scenario: G1 still fails a release pull request

- **WHEN** a release pull request's tree pins a library version whose docs bundle is not published
- **THEN** the release-pin step exits non-zero naming the bundle, as before
