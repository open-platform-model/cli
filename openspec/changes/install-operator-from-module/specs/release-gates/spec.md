## MODIFIED Requirements

### Requirement: Release PRs refuse local, unpublished or mismatched pins
A release-pin check (G1) SHALL fail, naming every offending pin and its file, when any of the following holds in the tree under test:
- `go.mod` carries a `replace` directive;
- a `github.com/open-platform-model/*` requirement in `go.mod` is a Go pseudo-version, or names a version for which the module's repository has no tag;
- a template `cue.mod/module.cue` under `templates/` pins a dependency at a `-0.dev.` version;
- a `cue.mod/local-module.cue` file is tracked anywhere in the repository;
- the operator module pin in `internal/operator/pin.go` names a module version the registry does not serve, records a content digest that differs from the one the registry serves for that module or for any module the pin lists, or records a `PinnedOperatorVersion` other than the operator version the pinned module's render deploys, or the pin cannot be read.

The check SHALL report all failures in one run, not only the first, and SHALL exit zero when none holds. It SHALL run as a step inside the existing `lint` job of both the pull-request workflow and the push workflow, and only when the branch name (`head_ref`, or `ref_name` when `head_ref` is empty) starts with `release-please--`; on every other branch the step SHALL be skipped while the job still runs. `task deps:release-check` SHALL run the same check locally on any branch. Source: workspace RELEASING.md, section "Gates".

#### Scenario: Clean release PR passes
- **WHEN** a release-please PR's tree has no replace, no tracked `local-module.cue`, library pinned at an existing tag, no dev template pin, and an operator module pin whose version, digests and operator version match what the registry serves
- **THEN** the release-pin step exits zero and the `lint` job's result is decided by golangci-lint alone

#### Scenario: Pseudo-version library pin fails the release PR
- **WHEN** a release-please PR's `go.mod` requires `github.com/open-platform-model/library` at a pseudo-version
- **THEN** the release-pin step exits non-zero naming the module, the version and `go.mod`

#### Scenario: Several violations are reported together
- **WHEN** a release-please PR carries a `replace` directive and a template pinned at a `-0.dev.` core version
- **THEN** the release-pin step exits non-zero and its output names both violations

#### Scenario: Embedded operator out of step with its pin
- **WHEN** `PinnedOperatorVersion` names one operator release and the pinned operator module's render deploys another
- **THEN** the release-pin step exits non-zero naming both versions and `task operator:pin VERSION=<module version>` as the fix

#### Scenario: Missing or duplicated operator image line
- **WHEN** the operator module pin records a digest for a module that the registry serves with another digest, or names a module version the registry does not serve
- **THEN** the release-pin step exits non-zero naming `internal/operator/pin.go`, the module and its version

#### Scenario: Tag lookup failure is not a pass
- **WHEN** the tag lookup for an OPM Go pin, or the registry lookup for the operator module pin, fails for a reason other than a missing tag or version
- **THEN** the release-pin step exits non-zero and reports a lookup failure for that pin

#### Scenario: Ordinary PR is not gated
- **WHEN** a pull request from any branch not starting with `release-please--` carries a pseudo-version library pin
- **THEN** the release-pin step is skipped and the `lint` job does not fail on account of the pin

#### Scenario: Release branch push is gated
- **WHEN** the release App pushes to `release-please--branches--main--components--opm` and the push workflow runs
- **THEN** the push workflow's `lint` job runs the release-pin step against that tree

### Requirement: A moved embedded operator on a release PR needs e2e evidence
An operator evidence check (G4) SHALL run on every pull request against `main`, on open, synchronize, reopen, label and unlabel events, as a job that is never skipped at job level. On a pull request whose branch does not start with `release-please--` it SHALL pass and say the check does not apply. On a release-please pull request it SHALL read the last released cli version from the base branch's `.release-please-manifest.json`, take the tag `v<version>`, and compare the pinned operator module version (`PinnedModuleVersion` in `internal/operator/pin.go`) at that tag with the one in the pull request; a tag that predates the module pin has no module version, which counts as moved. When they are equal it SHALL pass. When they differ it SHALL pass only if the pull request carries the label `e2e-verified`, and otherwise SHALL fail with a message naming both module versions, the tag, `task test:e2e`, the label, and that the check is interim until the cluster-backed e2e job (`add-embedded-operator-e2e-job`) runs in CI. When the tag does not exist, or the pull request's pin cannot be read, the check SHALL fail naming the tag or the file and SHALL NOT pass as unchanged. Source: workspace RELEASING.md, section "Gates".

#### Scenario: Operator unchanged since the last release
- **WHEN** a release-please PR's `PinnedModuleVersion` equals the one at the last cli tag
- **THEN** the evidence check passes without looking at labels

#### Scenario: Operator moved without evidence
- **WHEN** a release-please PR's `PinnedModuleVersion` differs from the one at the last cli tag, or the last tag has no module pin, and the PR has no `e2e-verified` label
- **THEN** the evidence check fails naming both versions and telling the reviewer to run `task test:e2e` and add `e2e-verified`

#### Scenario: Adding the label re-runs the check
- **WHEN** a human adds `e2e-verified` to that release-please PR
- **THEN** the evidence check runs again on the label event and passes

#### Scenario: Ordinary PR passes
- **WHEN** a pull request from a feature branch changes `PinnedModuleVersion`
- **THEN** the evidence check passes and states that it applies to release-please PRs only

#### Scenario: Missing last tag fails closed
- **WHEN** the version in the base branch's `.release-please-manifest.json` has no matching tag
- **THEN** the evidence check fails naming the missing tag
