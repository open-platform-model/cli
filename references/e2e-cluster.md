# E2E on kind: cluster tasks and the required check

Covers the local `kind` cluster tasks, what `task cluster:controller` prepares, and the CI check
`E2E (kind, embedded operator)` with the rule for when it runs. The registry mapping is the workspace
`references/registry.md`.

## Contents

- Cluster tasks
- Controller and Platform readiness
- The CI check
- Reproduce it locally
- Registry for the kind flow
- Pitfalls
- Check before you finish

## Cluster tasks

- `task cluster:create` creates the local `kind` cluster `opm-dev`.
- `task cluster:status` checks that it runs.
- `task cluster:delete` removes it. `task cluster:recreate` rebuilds it from scratch.
- `task test:integration` checks for the context `kind-opm-dev` before it runs, and needs a live
  cluster.

## Controller and Platform readiness

`task cluster:controller` is the complete path to a reconciling controller on `kind-opm-dev`:

1. It installs the pinned controller module with `--skip-platform`.
2. It applies the kind Platform manifest in `hack/` as the cluster Platform.
3. It applies the dev-only applier grant the RBAC manifest in `hack/`.

`opm controller install` alone does neither step 2 nor step 3. The controller-owned e2e tests then fail at their
applier precondition (`controller applier ... may not patch services`) and name `task cluster:controller`
as the remedy.

It finishes with `task cluster:controller:wait-ready`. That passes only when `Platform/cluster` is
`Ready=True` for its current generation (`status.observedGeneration` equal to `metadata.generation`).
`status.controllerVersion` alone is not proof, because the controller stamps it whatever the outcome. On
timeout (`PLATFORM_READY_TIMEOUT`, default 120 seconds) it prints both generations and the `Ready`
and `Stalled` conditions.

## The CI check

`.github/workflows/e2e-cluster.yml` (check `E2E (kind, embedded operator)`) is the only CI job that
runs the e2e suite against a real cluster and the pinned controller module. The check keeps its old name
because the `main` ruleset requires it.

It runs on every pull request, but does the work only when a script in `.github/scripts/`
says it applies to an open pull request:

- a change under `internal/controller/`, `internal/cmd/controller/`, `templates/` or `hack/platform/`;
- a change to the job's own inputs: the workflow and that script, `Taskfile.yml`, the fixtures script
  and the kind and config files in `hack/`, a Go file directly under
  `tests/e2e/`, `tests/e2e/testdata/controller-owned/`;
- the cascade branch `deps/cascade` or the label `deps-cascade`;
- a release-please pull request;
- `workflow_dispatch`.

Otherwise it passes in seconds and says why. When it applies, it prepares the cluster with `task
cluster:create` and `task cluster:controller` (a seeded `opm-registry` on kind's network through
`KIND_CUE_REGISTRY`) and sets `OPM_E2E_REQUIRE_CLUSTER=1`. That turns the suite's "no cluster" skips
into failures. It runs on every release PR.

## Reproduce it locally

Prepare a `kind-opm-dev` you own with `task cluster:create` and `task cluster:controller`, then run
`OPM_E2E_REQUIRE_CLUSTER=1 task test:e2e`. Never use a shared cluster: the lifecycle test
tears the controller down.

## Registry for the kind flow

No local registry is required for the kind flow. `task cluster:controller` installs the pinned controller
module from GHCR and relies on the controller's built-in `--registry` default, which routes both domains
to GHCR. It passes the mapping as the controller instance's `registry` value (a values file, recorded on
the instance). It never patches the Deployment. The shipped CLI default
(in `internal/config/`) matches.

It requires the `opm-registry` container only when `KIND_CUE_REGISTRY` is set, which is the opt-in
path for iterating on a locally published module. Use the in-cluster spelling (see the comment in
`Taskfile.yml`).

## Pitfalls

- Running `opm controller install` and the controller-owned e2e tests: run `task cluster:controller`.
- Reading `status.controllerVersion` as ready: wait for `observedGeneration` to match.
- Running the suite on a shared cluster: the lifecycle test removes the controller.
- A stale `OPM_CONFIG` in your shell: run the suite with `env -u OPM_CONFIG`. The e2e helper passes the
  full parent environment (the e2e command helper in `tests/e2e/`), and `OPM_CONFIG` wins over the seeded
  home (the config path resolver in `internal/config/`).
- Skipped e2e tests read as a pass: set `OPM_E2E_REQUIRE_CLUSTER=1`.
- Renaming the CI check: the `main` ruleset requires the current name.
- Setting `KIND_CUE_REGISTRY` by habit: it adds a local registry dependency.

## Check before you finish

- [ ] A change to a path in the applies list also passes locally with `OPM_E2E_REQUIRE_CLUSTER=1`.
- [ ] `task cluster:controller:wait-ready` passed before the suite ran.
- [ ] The workflow name and check name stay `E2E (kind, embedded operator)`.
- [ ] You ran the suite on a cluster you own.
