## Why

Module authors today can render a module to manifests with `opm module build` (using the synthetic-release flow), but to actually deploy it they must author a `release.cue` file first and run `opm release apply`. For the inner-loop case — "iterate on a module, deploy to a test cluster, repeat" — this extra authoring step is unnecessary friction. The synthetic-release pipeline already produces a fully-formed `*ModuleRelease` with a deterministic UUID (derived in CUE from module FQN + name + namespace), so the same release apply pipeline works without code changes.

This change adds `opm module apply` as a sibling to `opm module build`, completing the inner-loop deploy story.

## What Changes

- **New subcommand**: `opm module apply [path]` (alias `opm mod apply`) — apply a module package directly to a cluster via the synthetic `#ModuleRelease` flow.
- **Behavior**: Semantically identical to `opm release apply` after the render stage — writes an inventory Secret, prunes stale resources, supports re-apply / upgrade against the same synthetic release identity.
- **Flag surface** (matches `opm release apply` + the `--name` flag from `opm module build`):
  - `-f, --values` — values files (repeat)
  - `--provider` — provider override
  - `--name` — synthetic release name override (default: `<module>-debug`)
  - `-n, --namespace` — target namespace
  - `--kubeconfig`, `--context` — Kubernetes targeting
  - `--dry-run` — server-side dry-run
  - `--create-namespace` — auto-create target namespace
  - `--no-prune` — skip stale-resource pruning
  - `--force` — allow a 0-resource render to prune all previously tracked resources
- **No safety gate on cluster context**: this is the module author's responsibility — same posture as `opm release apply`.
- **Out of scope**: `--wait`/`--timeout` (separate proposal if added later), output-formatting flags (`--output`/`--split`/`--out-dir` — those belong on `module build`, not `apply`).

This is a **MINOR** SemVer bump — a new command with sensible defaults, no breaking changes.

## Capabilities

### New Capabilities

- `mod-apply`: The `opm module apply` (alias `opm mod apply`) subcommand — flag surface, default behavior, and command-level error contract. Renders a module package via the synthetic-release flow and applies the result to a Kubernetes cluster with the same semantics (inventory, prune, dry-run, force) as `opm release apply`.

### Modified Capabilities

- `cmd-structure`: Register `apply` as a subcommand under the `module` command group, alongside the existing `init`, `vet`, `build` subcommands. Mirrors the existing pattern for `opm module build`.

## Impact

- **Affected code**:
  - `internal/cmd/module/apply.go` (new) — cobra command, flag wiring, runner that calls `render.FromModule` then `workflowapply.Execute`.
  - `internal/cmd/module/mod.go` — register the new subcommand on `NewModuleCmd`.
  - `internal/cmd/module/apply_test.go` (new) — unit tests for flag wiring + error paths.
  - `tests/integration/` or `tests/e2e/` — at least one integration test exercising the full module-apply → kind cluster → inventory check loop.
- **Unaffected**:
  - `render.FromModule`, `loader.SynthesizeModuleReleaseFromPackage`, `workflowapply.Execute` — all already produce/consume the right types.
  - `pkg/inventory`, `internal/inventory`, `internal/kubernetes/apply` — behave identically; synthetic release UUID is already deterministic via CUE.
- **Documentation**:
  - Help text for `opm module apply` must call out: synth-release naming (`<module>-debug`), that `--name`/`--namespace` participate in release identity (different values = different releases), and the graduation path to `opm release apply` (delete the synthetic release first to avoid orphan inventory).
- **No new dependencies, no schema changes, no provider-side changes.**
