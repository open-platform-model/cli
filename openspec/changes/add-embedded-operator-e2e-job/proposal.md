## Why

No cluster-backed e2e test has ever run in cli CI. The PR `integration` job boots kind but installs
only the CRDs (`.github/workflows/pr.yml:201`, `opm operator install --crds-only`), and the PR `e2e`
job has no cluster at all (`pr.yml:214-246`), so `requireKindCluster` skips every cluster test there
(`tests/e2e/operator_test.go:36-45`). The embedded operator (`internal/operator/dist/install.yaml`,
pinned by `PinnedOperatorVersion` at `internal/operator/manifest.go:19`) is therefore never installed,
never seeded with a Platform and never asked to reconcile an instance in CI.

That gap hid a 27-day outage. Core alpha.7 broke the embedded operator alpha.14 without any pin
moving, the cli released alpha.18 to alpha.21 against it, and the fix (`7cd50cb`, cli issue 214)
shipped only in alpha.22. The local loop that would have caught it was the only gate, and it was not
run. The release cascade (workspace RELEASING.md, section "Gates") covers this with G4, a manual
`e2e-verified` label on cli release PRs that change `PinnedOperatorVersion`, explicitly as an interim
measure until this job exists (workspace RELEASING.md, section "Rollout and changes").

## What Changes

- **New workflow `.github/workflows/e2e-cluster.yml`, one job "E2E (kind, embedded operator)".** It
  boots kind with the node image and kind version `pr.yml` already pins, runs a registry container
  seeded from the tree that both the host and the in-cluster operator can reach, prepares the cluster
  with the same `task cluster:operator` the local loop uses (embedded operator, cluster Platform from
  `hack/kind-platform.yaml`, dev applier grant), and runs the whole `tests/e2e` suite with the cluster
  required.
- **When it does real work.** On a pull request to `main` that changes `internal/operator/**`,
  `templates/**`, `hack/platform/**`, or the job's own inputs (the workflow file and its decision script, `hack/kind-*.yaml`,
  `hack/opm-config.cue`, the cluster-backed e2e test files and their testdata); on a pull request
  carrying the `deps-cascade` label; on every release-please PR (head branch starting
  `release-please--`); and on `workflow_dispatch`.
- **It always reports.** The workflow triggers on every pull request and decides inside the job;
  when none of the conditions hold, the job passes in seconds and says why. A path-filtered workflow
  would leave a required check pending forever, so this is what lets the owner make it required.
- **A cluster-required run cannot pass by skipping.** A new opt-in, `OPM_E2E_REQUIRE_CLUSTER=1`, turns
  the suite's "no cluster, skip" and "could not check the applier grant, skip" paths into failures.
  The new job sets it; local runs and the existing PR `e2e` job do not, so they behave as today.
- **`task cluster:operator` waits for the Platform to be Ready, not merely stamped.** Today it waits
  for `status.operatorVersion` (`Taskfile.yml:287-300`), which the operator stamps on every reconcile
  regardless of outcome, so a stalled operator (the H3 failure exactly) passes the task and the job
  only fails three minutes per test later, far from the cause. The archived change
  `2026-09-19-e2e-local-loop-self-sufficient` deferred this until the operator pin moved past
  alpha.14; it is now `v1.0.0-beta.2`.

- **Not in this change.**
  - *Retiring G4.* The `e2e-verified` step belongs to `prepare-release-cascade`. Removing it is a
    one-commit follow-up, gated as listed below.
  - *A scheduled (nightly) run.* H3 needed no PR to break: a core publish did it. This job catches
    that class at the next release PR, which would have blocked cli alpha.18. A nightly run would
    catch it within a day; it is a one-line trigger the owner can add later, left out to keep to
    the trigger set the owner chose (workspace RELEASING.md, section "Rollout and changes").
  - *The integration programs.* `pr.yml`'s `integration` job keeps its CRDs-only cluster.
  - *Repository settings.* Making the check required is a ruleset edit the owner makes.

## Depends on / gates

- **Depends on:** nothing. The change is self-contained and can merge before or after any other
  release-cascade change. The `deps-cascade` label it keys on is created by `prepare-release-cascade`;
  until that label exists, the label condition simply never matches.
- **Gates G4 retirement.** G4 (the `e2e-verified` label rule added by cli `prepare-release-cascade`,
  workspace RELEASING.md, section "Gates") retires when all of these hold: this change has merged,
  the "E2E (kind, embedded operator)" check has passed on at least one cli release-please PR, and the
  owner has added that check to the cli ruleset's required checks (workspace RELEASING.md, section
  "Owner settings"). The retirement is a `ci(release)` commit that deletes the G4 step and the
  `e2e-verified` label entry, landing in whichever of `prepare-release-cascade` or
  `join-release-cascade` (cli) is open at that point, or on its own.
- **Interacts with:** `bump-stale-testdata-pins` and the cascade's fixture bumps, which change
  `tests/e2e/testdata/**`. The job seeds the tree's fixtures into its own registry and routes the
  operator to it, so a PR that bumps the operator-owned fixture is tested against the bumped version
  before GHCR holds it.

## Capabilities

### New Capabilities

- `e2e-cluster-workflow`: the CI workflow that runs the cluster-backed e2e suite against the embedded
  operator: when it does real work, that it always reports, how it prepares the cluster, and that a
  cluster-required run fails rather than skips.

### Modified Capabilities

- `e2e-cluster-preconditions`: one added requirement: with `OPM_E2E_REQUIRE_CLUSTER=1` the cluster
  preconditions fail instead of skipping.
- `kind-cluster-tasks`: `Operator install task needs no local registry` changes its readiness signal
  from `status.operatorVersion` to the Platform's `Ready=True` condition, and fails naming the
  Platform's stall reason otherwise.

## Impact

- **Affected files:** `.github/workflows/e2e-cluster.yml` and `.github/scripts/e2e-cluster-applies.sh` (new), `tests/e2e/operator_test.go`
  (`requireKindCluster`), `tests/e2e/instance_operator_owned_test.go`
  (`requireOperatorApplierGrant`), `Taskfile.yml` (`cluster:operator`), plus this change's design.md
  for the spike record.
- **Affected commands:** none. No `opm` behaviour changes.
- **CI cost:** one extra ubuntu-latest job, roughly 10 to 15 minutes when it applies (estimate;
  section 1 measures it), and about 15 seconds when it does not.
- **SemVer:** none. Tests, CI and dev tooling only. Section commits are `chore(openspec)`,
  `test(e2e)`, `chore(taskfile)` and `ci(e2e)`, none of which releases.
- **Complexity (Principle VII):** one workflow file and one environment switch. The job reuses the
  local loop's tasks rather than restating them, so CI and a developer's `task cluster:operator`
  cannot drift apart.
