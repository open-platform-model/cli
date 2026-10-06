## Context

The cli names the controller in four kinds of places: Go packages and identifiers, string contracts with the controller (instance coordinates, the module path and its `controller` sub-package, `spec.owner`, `status.controllerVersion`), task/CI/cascade wiring that other repos mirror, and prose and specs. The controller side moves in its own change; this change is merged in the same cutover, after the controller's first release under the new name.

## Goals / Non-Goals

**Goals:** every live surface names the controller; no alias, no compatibility read of `status.operatorVersion` or `spec.owner: operator`.

**Non-Goals:** CHANGELOG.md and `openspec/changes/archive/**` stay as written; the required check name `E2E (kind, embedded operator)` stays (ruleset contract); `.tasks/cascade/wiring-check.sh` is a byte copy of `.github`'s and changes only through a cascade pin PR; the human role "platform operator" and upstream operators are not renamed.

## Research & Decisions

### Rename mechanics
**Context**: about 100 Go files and 70 other files name the product.
**Options considered**:
1. Exact-pattern sed only (`opm-operator`, `opm_operator`, `opm operator`) - leaves identifiers and prose behind.
2. Whole-word product rename per file class, with protected phrases - Go code: every `operator`/`Operator` token is the product in this repo; prose: protected phrases (`platform operator`, `External Secrets Operator`, the required check name) and a generic-docs list where only the exact patterns change.
**Decision**: Option 2, then a per-line review for stutters ("the controller's controller Deployment") and articles ("an controller").
**Rationale**: the compiler and the residue grep catch what a review misses.

### The pin
**Decision**: `PinnedModuleVersion` is `0.1.0`, which is also the first `opm_controller` release. `PinnedControllerVersion` cannot be produced before that module is published; it is written by `task controller:pin VERSION=0.1.0` at cutover, never by hand.

### `docs-pins-check.sh` fallback
**Decision**: replace the long-dead `manifest.go` fallback with `internal/operator/pin.go` and accept both constant names, so the rename PR's own `--moved-from <base>` run can read its base. The file is not mirrored in `.github`, so its later removal costs no pin round. `cascade.sh` gets no fallback: it builds `hack/controller-pin` from the merge base, which has it once this change merges.

### Fixture
**Decision**: the podinfo fixture's comments name the controller repo, so its tree changes and its version moves to `0.1.13` (fixture rule: a changed tree needs a version GHCR does not hold); the two consumers re-pin. CI seeds it; GHCR holds it after `publish-fixtures.yml` runs on merge.

## Risks / Trade-offs

- Merging before `opm_controller` 0.1.0 is public breaks `opm controller install` and the e2e job → the PR merges only after that release and its module are published.
- `cmdutil`'s registry-backed test and the examples resolve podinfo `0.1.13` from GHCR → they fail locally until the fixture is published; CI uses a seeded registry.
