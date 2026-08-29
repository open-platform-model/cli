# Proposal — fixture-testing-domain

## Why

`tests/fixtures/modules/podinfo/` declared `opmodel.dev/modules/test/podinfo@v0` — a test fixture on
the production namespace, published nowhere. CUE routes by longest prefix on the module path, so the
only way to reach it was to bend `opmodel.dev` itself. That is exactly what the repo did, in three
places: `KIND_CUE_REGISTRY` pointed all of `opmodel.dev` at `opm-registry:5000`,
`hack/opm-config.cue` at `localhost:5000`, and `examples/Taskfile.yml` carried a
`opmodel.dev/modules/test=localhost:5000+insecure` prefix entry. The collateral was that core and the
catalogs had to be mirrored into a laptop registry too, and `task cluster:operator` hard-required a
running registry container before it would do anything.

Two facts make the fix cheap. `gateNamespace` (`internal/publish/gates.go:258-262`) deliberately
exempts `testing.opmodel.dev` from the owned-domain shape rules, so the new coordinate is publishable
by `opm module publish` while the old one is *refused* (pinned at `gates_test.go:208`). And both the
shipped CLI default (`internal/config/templates.go:9`) and the pinned operator's built-in `--registry`
default (`opm-operator` `v1.0.0-alpha.11`, `cmd/main.go`) already route `testing.opmodel.dev` to
GHCR — only the developer-facing mappings still said localhost.

This is enhancement 0011's `registry-cleanup` slice (D17 item 3) for the CLI's own fixture. The
operator's fleet is a separate move on its own schedule.

## What Changes

- **The fixture moves to `testing.opmodel.dev/modules/cli/podinfo@v0`** and becomes a real publishable
  artifact: a new `identity/identity.cue` package is the single source of path and version, and
  `metadata.{name,modulePath,version}` derive from it (the shape `templates/minimal` already uses).
  Version stays `0.1.4`, so no consumer pin changes value.
- **New `publish-fixtures.yml` workflow + `.github/scripts/publish-fixtures.sh`** publish every
  fixture tree carrying an `identity/` package to GHCR through `opm module publish` — the same
  pipeline and the same gates the official templates go through. Triggers: push to `main` touching
  the fixtures, plus `workflow_dispatch` for the bootstrap/repair path. Idempotency is a caller-side
  GHCR probe, because publish itself never skips an already-published tag (D15).
- **Both publish scripts export `OPM_REGISTRY`, not just `CUE_REGISTRY`.** `opm` resolves
  `--registry` > `OPM_REGISTRY` > `~/.opm/config.cue` and never consults `CUE_REGISTRY`, so the
  existing `publish-templates.sh` was silently running against whatever the caller's personal config
  said. It works on a CI runner that has no config and misleads everywhere else — a pre-existing
  defect fixed alongside the new script.
- **`pr.yml` gains a `fixture-gates` dry-run job** mirroring `template-gates`, so a fixture that would
  fail the publish fails the PR first. Its `OPM_REGISTRY`/`CUE_REGISTRY` env gains the
  `testing.opmodel.dev` mapping the tests now need.
- **Every `opmodel.dev` override is deleted from the repo.** `KIND_CUE_REGISTRY` defaults to empty;
  `hack/opm-config.cue` and `examples/Taskfile.yml` map both domains to GHCR.
- **`cluster:operator` no longer requires a local registry.** The pinned operator ships no `--registry`
  arg and its built-in default already routes both domains to GHCR, so the registry precondition, the
  `docker network connect`, and the `--registry` patch all move behind an explicit `KIND_CUE_REGISTRY`
  opt-in for iterating against a locally published module.
- **Consumers re-pointed**: `examples/`, `tests/e2e/testdata/handoff/`, `render-parity`,
  `ssa-ownership`, and two unit tests.
- **Deliberately NOT changed**: the operator-owned `opmodel.dev/modules/test/{hello,hello_web,redis}`
  fixtures and the `opmodel.dev/releases/test/*` Flux artifacts (opm-operator's move, its own
  deviation note stays accurate); `gates_test.go:208`, which uses an `opmodel.dev/modules/test/...`
  path as an expected *refusal* and still illustrates the rule correctly;
  `instance-inventory`'s digest scenario, whose coordinate is an arbitrary illustration of an
  algorithm this change does not touch.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `test-fixture-lineage`: fixtures live on the testing domain, carry an identity package, and are
  published to GHCR rather than existing only in a local registry.
- `ci-workflow`: a fixture publish workflow and its PR-side dry-run gate.
- `kind-cluster-tasks`: `cluster:operator` runs against GHCR with no local registry; the
  local-registry path is an explicit opt-in.

## Impact

A fresh clone can run the examples, the e2e suite, and the kind dev cluster with no registry
container and no sibling checkout. The workspace Registry Policy (root `CLAUDE.md`) is amended in the
same pass: `testing.opmodel.dev` is a GHCR-backed domain whose local routing is an opt-in override,
not its definition.
