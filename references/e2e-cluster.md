# E2E on kind: cluster tasks and the required check

Covers the local `kind` cluster tasks, what `task cluster:operator` prepares, and the CI check
`E2E (kind, embedded operator)` with the rule for when it runs. The registry mapping is the workspace
`references/registry.md`.

## Contents

- Cluster tasks
- Operator and Platform readiness
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

## Operator and Platform readiness

`task cluster:operator` is the complete path to a reconciling operator on `kind-opm-dev`:

1. It installs the pinned operator module with `--skip-platform`.
2. It applies `hack/kind-platform.yaml` as the cluster Platform.
3. It applies the dev-only applier grant `hack/kind-operator-rbac.yaml`.

`opm operator install` alone does neither step 2 nor step 3. The operator-owned e2e tests then fail at their
applier precondition (`operator applier ... may not patch services`) and name `task cluster:operator`
as the remedy.

It finishes with `task cluster:operator:wait-ready`. That passes only when `Platform/cluster` is
`Ready=True` for its current generation (`status.observedGeneration` equal to `metadata.generation`).
`status.operatorVersion` alone is not proof, because the operator stamps it whatever the outcome. On
timeout (`PLATFORM_READY_TIMEOUT`, default 120 seconds) it prints both generations and the `Ready`
and `Stalled` conditions.

## The CI check

`.github/workflows/e2e-cluster.yml` (check `E2E (kind, embedded operator)`) is the only CI job that
runs the e2e suite against a real cluster and the pinned operator module. The check keeps its old name
because the `main` ruleset requires it.

It runs on every pull request, but does the work only when `.github/scripts/e2e-cluster-applies.sh`
says it applies to an open pull request:

- a change under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`;
- a change to the job's own inputs: the workflow and that script, `Taskfile.yml`, `hack/fixtures.sh`,
  `hack/kind-{config,platform,operator-rbac}.yaml`, `hack/opm-config.cue`, a Go file directly under
  `tests/e2e/`, `tests/e2e/testdata/operator-owned/`;
- the cascade branch `deps/cascade` or the label `deps-cascade`;
- a release-please pull request;
- `workflow_dispatch`.

Otherwise it passes in seconds and says why. When it applies, it prepares the cluster with `task
cluster:create` and `task cluster:operator` (a seeded `opm-registry` on kind's network through
`KIND_CUE_REGISTRY`) and sets `OPM_E2E_REQUIRE_CLUSTER=1`. That turns the suite's "no cluster" skips
into failures. It runs on every release PR.

## Reproduce it locally

Prepare a `kind-opm-dev` you own with `task cluster:create` and `task cluster:operator`, then run
`OPM_E2E_REQUIRE_CLUSTER=1 task test:e2e`. Never use a shared cluster: the lifecycle test
tears the operator down.

## Registry for the kind flow

No local registry is required for the kind flow. `task cluster:operator` installs the pinned operator
module from GHCR and relies on the operator's built-in `--registry` default, which routes both domains
to GHCR. It passes the mapping as the operator instance's `registry` value (a values file, recorded on
the instance). It never patches the Deployment. The shipped CLI default
(`internal/config/templates.go`) matches.

It requires the `opm-registry` container only when `KIND_CUE_REGISTRY` is set, which is the opt-in
path for iterating on a locally published module. Use the in-cluster spelling (see the comment in
`Taskfile.yml`).

## Pitfalls

- Running `opm operator install` and the operator-owned e2e tests: run `task cluster:operator`.
- Reading `status.operatorVersion` as ready: wait for `observedGeneration` to match.
- Running the suite on a shared cluster: the lifecycle test removes the operator.
- A stale `OPM_CONFIG` in your shell: run the suite with `env -u OPM_CONFIG`. The e2e helper passes the
  full parent environment (`tests/e2e/mod_build_test.go:44`), and `OPM_CONFIG` wins over the seeded
  home (`internal/config/resolver.go:105-123`).
- Skipped e2e tests read as a pass: set `OPM_E2E_REQUIRE_CLUSTER=1`.
- Renaming the CI check: the `main` ruleset requires the current name.
- Setting `KIND_CUE_REGISTRY` by habit: it adds a local registry dependency.

## Check before you finish

- [ ] A change to a path in the applies list also passes locally with `OPM_E2E_REQUIRE_CLUSTER=1`.
- [ ] `task cluster:operator:wait-ready` passed before the suite ran.
- [ ] The workflow name and check name stay `E2E (kind, embedded operator)`.
- [ ] You ran the suite on a cluster you own.
