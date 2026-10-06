## Why

The Kubernetes controller OPM ships is renamed from `opm-operator` to `opm-controller` (0032:D1, decided 2026-10-05): it is a generic delivery engine in the family of Flux's controllers and Argo CD, not an "operator" carrying one application's day-2 knowledge. The controller repo, image, namespace, CUE module (`opmodel.dev/modules/opm_controller@v0`, sub-package `controller`) and CRD fields move in the controller's own change; the cli must follow in the same cutover, with no alias, no warning and no trace of the old name.

## What Changes

- **BREAKING** `opm operator install|uninstall` becomes `opm controller install|uninstall`. No alias.
- **BREAKING** The CLI installs `opmodel.dev/modules/opm_controller` as the CLI-owned ModuleInstance `opm-controller` in `opm-controller-system`; the controller Deployment is `opm-controller-manager`; the module's version is read from its `controller` sub-package. `PinnedModuleVersion` stays `0.1.0` (the first `opm_controller` release); `PinnedOperatorVersion` becomes `PinnedControllerVersion`.
- **BREAKING** `spec.owner: operator` becomes `spec.owner: controller` (`inventory.OwnerController`; an absent owner displays as `controller`), and the version gate reads `Platform.status.controllerVersion` only.
- Packages: `internal/operator` → `internal/controller` (with `controllertest`), `internal/cmd/operator` → `internal/cmd/controller` (`controllercmd`), `hack/operator-pin` → `hack/controller-pin` (`task controller:pin`), every Go identifier, help text, hint and error string.
- Tasks, CI and cascade: `task cluster:controller*`, `hack/kind-controller-rbac.yaml` (`opm-controller-dev-applier`), `.tasks/cascade/*`, `docs-kit.cue` pins `opm-controller`, `hack/docskit-dump`, `release-pin-check.sh`, `e2e-cluster-applies.sh`, `pr.yml`, `e2e-cluster.yml`. The required check name `E2E (kind, embedded operator)` is kept exactly. `docs-pins-check.sh` keeps one fallback that reads the old pin file at a base revision from before the rename.
- Tests: e2e constants and fixtures (`tests/e2e/testdata/controller-owned`), the podinfo fixture comment (fixture version `0.1.13`), the fixture namespace `testing.opmodel.dev/*/controller/*`.
- Docs: README, QUICKSTART, AGENTS, `docs/site`, RFC 0007 (renamed file) and the other dated design material.

Release class: MAJOR after GA (a renamed command, a changed `spec.owner` value); during beta it ships as the next `-beta.N`. Commit `refactor!:`.

## Capabilities

### New Capabilities

- `controller-lifecycle`: the `operator-lifecycle` requirements under the controller's names.

### Modified Capabilities

- `operator-lifecycle`: retired (every requirement REMOVED, `retire_capabilities: true`), restated as `controller-lifecycle`.
- Body or title wording only, the product word and names: `apply-preflight-gates`, `apply-pruning`, `artifact-publishing`, `cmd-structure`, `config-types`, `deploy`, `deps-cascade`, `docs-bundle`, `e2e-cluster-preconditions`, `e2e-cluster-workflow`, `health-evaluation`, `instance-inventory`, `inventory-ownership`, `kernel-render`, `kind-cluster-tasks`, `mod-apply`, `platform-check`, `platform-pull`, `platform-resolution`, `pr-workflow`, `release-cascade-wiring`, `release-gates`, `release-workflow`, `resource-conversion`, `test-fixture-lineage`. A requirement whose title or scenario names change is REMOVED and ADDED under a new title; Purpose lines are edited in the main specs at archive.

## Impact

- Needs, before merge: the controller's first release under the new name (`v1.0.0-beta.9`, image `ghcr.io/open-platform-model/opm-controller`) and the module `opm_controller` `0.1.0`, so `task controller:pin VERSION=0.1.0` can write the pin; the renamed fixtures published.
- `inventory-ownership` loses the scenario for an operator applied with kubectl from a release manifest, which nobody runs.
- Delivers the cli side of 0032:D1 ("The operator becomes the controller"), declared in `enhancement.yaml`.
