## Context

See proposal.md. A fixture is a published artifact on `testing.opmodel.dev/modules/cli/podinfo@v0`; PR CI (`hack/fixtures.sh check` + `seed`) refuses a changed fixture whose declared version GHCR already holds, seeds the new version into a job-local registry, and runs render parity against it. Two consumers pin the version by literal in `cue.mod`.

## Goals / Non-Goals

**Goals:** the fixture carries the CLI-emitted identity form; consumers follow.
**Non-Goals:** any behavior change in rendered output (byte-identical; the version string is the only rendered difference).

## Decisions

**D1. Version advance via `opm module version set 0.1.6 ./tests/fixtures/modules/podinfo` after the form edit.** Proves the literal writer on the fixture and is the flow `deps:pins:fixtures` uses. Alternative (edit the literal by hand): same bytes; the tool run is the check.

**D2. Pins by hand or by the root task.** `task deps:pins:fixtures` also bumps CUE deps of the fixture; nothing is stale there today, so it is a no-op on deps and does the pins. Either path yields the same two-line diff.

**D3. The shared coordinate reader is strict, and the literal form is tested.** `fixtures.go` `Load` drops its `Default()` fallback: with literals there is nothing to default, and a defaulted disjunction would otherwise load here yet fail the kernel gate. `TestIdentityIsLiteral` guards the scenario across every fixture. Both files are copies shared with `opm-operator`; the identical bytes go into the operator's `fixtures-identity-literal` change, where the test is red until that change's fixtures are rewritten.

## Research & Decisions

### Does the fixture still pass render parity with the literal form?
**Context**: parity compares rendered output against a golden set.
**Explored**: the same edit on a fleet module rendered byte-identically apart from the version label (2026-08-28).
**Options considered**: (1) plain literal; (2) keep interpolation. **Decision**: (1). **Rationale**: the fixture's comment justifying interpolation ("the registry loader's shape gate requires a concrete metadata.version, and id.Version is a defaulted disjunction") describes the defect this set of changes removes.

## Risks / Trade-offs

- [`hack/fixtures.sh check` refuses because `0.1.6` exists on GHCR] → verify with the script before opening the PR; pick the next free patch if so.
