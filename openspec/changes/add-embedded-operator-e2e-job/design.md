# Design: add-embedded-operator-e2e-job

## Context

See proposal.md, Why. Current state, read 2026-10-01 at `origin/main` `f3569b24`:

- **PR CI has two kinds of e2e coverage, neither cluster-backed.** `pr.yml`'s `integration` job
  (`.github/workflows/pr.yml:181-212`) installs kind v0.31.0 by checksum (`:189-195`), creates
  `opm-dev` with the pinned node image (`:197`), installs only the CRDs (`:201`) and runs six
  `tests/integration` programs. The `e2e` job (`:214-246`) seeds a `registry:2` service from the tree
  and runs `go test ./tests/e2e/... -v -timeout 25m` with no cluster.
- **Four cluster-backed tests skip on a runner without a cluster.** `requireKindCluster`
  (`tests/e2e/operator_test.go:30-48`) skips when `~/.kube/config` is missing or `kind-opm-dev` does
  not answer. Its callers: `TestE2E_Operator_InstallUninstallLifecycle` (`operator_test.go:210`),
  `TestE2E_InstanceBuild_ReadsTheNamedContextsClusterPlatform` (`instance_build_test.go:766`), and,
  through `runOperatorOwnedOPM` (`instance_operator_owned_test.go:45-57`),
  `TestE2E_ThinEditor_ValuesRoundTrip` (`:266`) and `TestE2E_Delete_OperatorOwnedDelegates` (`:307`).
- **The operator-owned tests already refuse to skip on a half-prepared cluster.**
  `requireReconcilingOperator` (`instance_operator_owned_test.go:91-98`) fails when no operator is
  running, and `requireOperatorApplierGrant` (`:126-153`) fails on an explicit denial but skips when
  the check itself cannot run (`:148-150`).
- **The lifecycle test branches on a tool.** `imagePullable` (`operator_test.go:68-83`) returns false
  when `crane` is not on `PATH`, and the test then *requires* `operator install` to fail its 15-second
  rollout wait (`:238-245`). On a runner that can pull the image within 15 seconds, that branch fails
  for a reason unrelated to the product. CI must have `crane`.
- **The lifecycle test rebuilds the dev operator by calling `task cluster:operator`**
  (`restoreDevOperator`, `operator_test.go:166-179`), so the runner needs `task`, and whatever
  environment the job gives `cluster:operator` must still be in place when the test calls it.
- **`task cluster:operator` already is the cluster preparation** (`Taskfile.yml:214-300`): it builds
  the CLI (`deps: [build]`), installs the embedded operator with `./bin/opm operator install --config
  hack/opm-config.cue` (`:254`, no `--version`, so the embed), optionally points the operator at a
  local registry through `KIND_CUE_REGISTRY` (`:33`, `:258-280`), waits for the rollout (`:281`),
  applies `hack/kind-platform.yaml` (`:282`) and `hack/kind-operator-rbac.yaml` (`:286`), then waits for
  `status.operatorVersion` (`:287-300`).
- **`status.operatorVersion` is not a health signal.** The archived change
  `2026-09-19-e2e-local-loop-self-sufficient` (design.md, Decision 3) measured `task
  cluster:operator` printing "operator v1.0.0-alpha.14 is reconciling" while `Platform/cluster` sat at
  `Ready=False`/`Stalled=True`, reason `MaterializeFailed`. That is the alpha.14 stall (cli issue
  214), the outage proposal.md describes. It deferred waiting for `Ready` until the operator pin
  moved; `PinnedOperatorVersion` was `v1.0.0-beta.2` when this was written
  (`internal/operator/manifest.go:19`), and workspace RELEASING.md, section "Rollout and changes"
  (Phase 1), catches the embed up to the newest published operator (`v1.0.0-beta.3` exists) before
  or alongside this change. The spike therefore records whatever `PinnedOperatorVersion` is embedded
  when it runs. The operator owns `Ready`, `Reconciling` and `Stalled` on the Platform (opm-operator
  `internal/controller/platform_controller.go:458,490-501`).
- **`Ready=True` alone can describe an older spec.** `task cluster:operator` runs `opm operator
  install` without `--skip-platform` (`Taskfile.yml:254`), which creates `Platform/cluster` subscribed
  to the newest published catalog (`internal/cmd/operator/install.go:52-57,150-153,187-190`). The
  task's `kubectl apply -f hack/kind-platform.yaml` (`:282`) then changes the spec to the pinned
  catalogs, which bumps `metadata.generation`. The operator sets `status.observedGeneration` on both
  its Ready and its Stalled paths (opm-operator `v1.0.0-beta.2`,
  `platform_controller.go:212,302,456`), so a wait that reads only `Ready` can see generation 1's
  `Ready=True` while generation 2 is stalling. Readiness means `Ready=True` *and*
  `status.observedGeneration == metadata.generation`.
- **The operator-owned fixture is a published module.** `tests/e2e/testdata/operator-owned/cue.mod/module.cue`
  pins `testing.opmodel.dev/modules/cli/podinfo@v0` `v0.1.11`. The operator resolves it inside the
  cluster. With the built-in registry default it reads GHCR, which holds a bumped fixture only after
  `publish-fixtures.yml` runs on merge.
- **Release PRs run CI.** release-please runs as the `opm-release-please` App (`release.yml:66-79`),
  so its pull requests fire `pull_request` workflows like any other. The cli's release branch is
  `release-please--branches--main--components--opm` (PRs 250, 257, 259, 261, 267); the rule keys on
  the `release-please--` prefix.
- **The developer's workspace registry is in use.** The workspace `opm-registry` container
  (`.tasks/config.yml:14`, `registry:2` on `0.0.0.0:5000`, bind-mounted to `.registry-data`) runs on
  the developer's Docker host and opm-kind-demo's Flux reads it. Local spike steps therefore use their
  own container and port (`opm-spike-registry` on `5001`) and never start, seed or remove
  `opm-registry`. `Taskfile.yml:11` already takes a `REGISTRY_CONTAINER` override. kind also switches
  the current kube-context to a cluster it creates, so local steps record and restore it.
- **Local timing.** The archived change's baseline (2026-09-19) ran the full suite in 616 s, of which
  about 562 s was three operator-owned tests waiting out 3-minute deadlines against the stalled
  alpha.14 operator. A healthy run is therefore expected to take a few minutes; it has not been
  measured on the current pin. The local `kind-opm-dev` cluster was up on 2026-10-01 but carried
  someone's unrelated workload (`default/backup-demo-web`) and no `Platform/cluster`; the suite's
  lifecycle test deletes every ModuleInstance and the operator, so it was not run there. The tests
  hardcode `kind-opm-dev`, and kind cluster names are unique per Docker host, so the suite cannot be
  timed on a second cluster beside it. The planned spike on a throwaway cluster
  (`CLUSTER_NAME=opm-spike`, with its own registry container) was dropped at implementation time:
  the implementing session may not create or apply to any cluster (Migration Plan, "No local
  cluster"). The preparation (the `Ready` wait) and the full suite are therefore both first proven
  by this change's own PR run, which applies because the workflow file is one of its inputs.

## Goals / Non-Goals

**Goals**

- A break of the alpha.14-stall class (the embedded operator cannot serve the current core or catalogs) fails a cli
  pull request that touches the operator, a cascade PR, or the release PR, and says so in the
  preparation step rather than three minutes into a test.
- CI and the local loop prepare the cluster with the same task, so neither can drift.
- The check can be made required without blocking unrelated pull requests.

**Non-Goals**

- Running the `tests/integration` programs against a full operator.
- Testing an operator version other than the embedded one (`--version`).
- Multi-version or upgrade testing (old embedded operator to new).
- Changing any repository setting. The ruleset entry that makes the check required is the owner's.

## Decisions

### 1. A separate workflow file, not a seventh job in `pr.yml`

The `deps-cascade` condition needs the `labeled` activity type. Adding it to `pr.yml` would re-run all
six of its jobs on every label change, and with `cancel-in-progress` it would cancel a running set to
do so. A separate `e2e-cluster.yml` takes `labeled` for itself alone.

**Alternatives considered**

1. *Seventh job in `pr.yml`.* One file, but the label trigger then costs every other job a re-run, and
   the `pr-workflow` capability's "six independent jobs" contract changes for a job that behaves
   differently from the other six.
2. *Give the existing `e2e` job a cluster.* Would make every PR pay for kind and the operator. That is
   the cost the owner chose not to pay; it is recorded here as the simpler option if the applicability
   logic ever proves fragile.
3. *Separate workflow.* Chosen.

### 2. Decide inside the job from pull-request state; always report

The workflow triggers on `pull_request` to `main` with the types `opened`, `synchronize`,
`reopened`, `labeled` and `unlabeled`, and on `workflow_dispatch`. It has no `paths:` filter, and
the job has no `if:`. The job declares `permissions: { contents: read, pull-requests: read }`
(job-level permissions replace the workflow-level set, so `contents: read`, which checkout needs, is
restated). Its steps start:

1. An ungated `actions/checkout` (the SHA `pr.yml` pins, shallow), so the decision script exists.
2. `decide`: runs `.github/scripts/e2e-cluster-applies.sh` with `GH_REPO: ${{ github.repository }}`
   in its `env:` (so `gh` needs no git remote) and every other expression passed through `env:`.

The script computes `applies` from inputs the step fetches and passes in (event name, head ref, a
labels file, a changed-files file), so the rule can be exercised locally with hand-written inputs:

```
applies := event == workflow_dispatch
        || head_ref starts with "release-please--"
        || head_ref == "deps/cascade"
        || "deps-cascade" in labels(PR, read live)
        || any(changed_files(PR, all pages) matches APPLY_PATHS)
```

with `APPLY_PATHS` as listed in the `e2e-cluster-workflow` spec. `deps/cascade` is the one rolling
cascade branch (workspace RELEASING.md, section "The cascade", "One rolling PR per repo"), so a
cascade PR applies even after someone removes its label. Labels and files come from the API
(`gh pr view <n> --json labels`, `gh api --paginate repos/{repo}/pulls/<n>/files`). A renamed
file contributes both its `filename` and its `previous_filename`, so moving a file out of an apply
path still applies. Every step after
`decide` carries `if: steps.decide.outputs.applies == 'true'`.

Why every condition reads live state: a required check is satisfied by the newest run for the head
commit. If an unrelated label event could produce a run that skips, that run would overwrite a
failing run with a pass. Making the decision a pure function of the pull request's head commit,
branch and labels means that the same head commit, branch and labels give the same decision, so a
re-run caused by an unrelated label can only repeat the earlier verdict. Removing `deps-cascade`
from a PR that applied only through the label does change the decision; that is why `unlabeled` is
a trigger (the change is reported at once, not at the next unrelated event) and why the cascade
branch name is a condition of its own. The cost is that an unrelated label on an applying PR re-runs
the whole job; labels are rare enough that this is accepted. The concurrency group is the workflow
and ref with `cancel-in-progress`, so the restart replaces, never duplicates.

Why not a job-level `if:`: a job skipped by its condition reports as skipped, which satisfies a
required check, and a workflow that path filters out never reports at all, which leaves a required
check pending. The step-level form is the usual always-report pattern and keeps one stable check name.

Example output when the job does not apply:

```
e2e-cluster: not applicable
  head branch: feat/mod-list-wide
  labels: enhancement
  changed files: 7, none under internal/operator/, templates/, hack/platform/ or the job's own inputs
  nothing to do; passing
```

**Alternatives considered**

1. *`paths:` filter on the workflow.* Simple, but cannot be required.
2. *A separate `decide` job with the e2e job `needs:` it.* Two check names, and the e2e job's
   skipped state is again a pass. No gain over a step.
3. *Read labels from the event payload.* The `opened` payload is taken when the pull request is
   created; a bot that adds the label in a second call would be missed until the `labeled` run. Live
   reads close that window.

### 3. Prepare with `task cluster:create` and `task cluster:operator`

The job installs kind exactly as `pr.yml:189-195` does (same version, same checksum), installs `task`
with `go-task/setup-task` at the pin `pr.yml:90-92` uses, then runs `task cluster:create` (node image
from `Taskfile.yml:10`, identical to `pr.yml:197`) and `task cluster:operator`. Nothing in the
workflow restates the operator install, the Platform or the RBAC grant.

This is also what makes the lifecycle test's `restoreDevOperator` work in CI: it calls the same task,
with the same job environment.

**Alternatives considered**

1. *Inline `opm operator install` plus `kubectl apply` steps in the workflow.* Faster to read in the
   YAML, but a second copy of the preparation that the restore cleanup would not use, so CI and the
   restore could prepare different clusters.
2. *Reuse the tasks.* Chosen.

### 4. A registry container on kind's network, not a `services:` registry

The operator-owned tests make the operator resolve the fixture the testdata pins. On a fixture bump,
GHCR does not hold that version until the merge publishes it, so the operator must read the tree's
seed. A `services:` container is on the runner's job network, which kind's node cannot reach by name.
The job instead runs `docker run -d --name opm-registry -p 5000:5000 registry:2` as a step, seeds it
with `hack/fixtures.sh seed`, and sets at job level

```
CUE_REGISTRY / OPM_REGISTRY = testing.opmodel.dev=localhost:5000+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works
KIND_CUE_REGISTRY           = testing.opmodel.dev=opm-registry:5000+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works
```

`cluster:operator` then joins `opm-registry` to the `kind` network and adds `--registry` to the
operator (`Taskfile.yml:258-280`); this is the opt-in path the `kind-cluster-tasks` spec already
defines, so nothing in the task changes for it. The container name matches the task's
`REGISTRY_CONTAINER` default (`Taskfile.yml:11`). That name and port are free on a fresh runner;
on a developer machine they belong to the workspace registry (Context), so local reproductions of
this setup use `REGISTRY_CONTAINER=opm-spike-registry` on port `5001` instead.

Verified 2026-10-01: Task reads an exported environment variable into a var declared as
`'{{.KIND_CUE_REGISTRY | default ""}}'` (Task 3.52.0, scratch Taskfile: unset prints empty, exported
prints the value). So a job-level `KIND_CUE_REGISTRY` reaches both the job's own
`task cluster:operator` and the one `restoreDevOperator` runs from inside the suite.

**Alternatives considered**

1. *Let the operator read GHCR (default).* Works until a PR bumps the fixture, then fails for a
   reason the PR cannot fix. Cascade and release PRs bump fixtures routinely.
2. *Connect the kind node to the services network.* Possible, but depends on the runner's
   generated network name and is not what the local loop does.
3. *A named registry container on the kind network.* Chosen; it is the local opt-in path.

### 5. `OPM_E2E_REQUIRE_CLUSTER=1` turns skips into failures

A cluster job whose tests all skip would pass green and prove nothing, the same failure mode as
today's `e2e` job. The switch changes only the skip branches in `requireKindCluster` and the
"could not check" branch of `requireOperatorApplierGrant`:

```go
// skipOrFailf skips, or fails when the run requires the cluster. A failure
// carries the variable as a prefix, so the log says why a skip became a failure.
func skipOrFailf(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("OPM_E2E_REQUIRE_CLUSTER") == "1" {
		t.Fatalf("OPM_E2E_REQUIRE_CLUSTER=1: "+format, args...)
	}
	t.Skipf(format, args...)
}
```

The messages change so they read correctly as either outcome: neither says "skipping", and both
`requireKindCluster` branches name the cause, the `kind-opm-dev` context and `task cluster:create`
(today the missing-kubeconfig branch at `operator_test.go:36-38` names neither):

```
no kubeconfig at "/home/runner/.kube/config", so context "kind-opm-dev" cannot be reached; run `task cluster:create`
kind cluster "kind-opm-dev" not reachable; run `task cluster:create`: exit status 1
```

Under `OPM_E2E_REQUIRE_CLUSTER=1` the same lines appear behind the `OPM_E2E_REQUIRE_CLUSTER=1: `
prefix as failures. The applier-grant message likewise drops its trailing "skipping operator e2e".
Unset, every precondition skips exactly where it skips now, so developer machines and `pr.yml`'s
`e2e` job are unaffected; only the skip text changes.

**Alternatives considered**

1. *Parse `go test -json` in the workflow and fail on any skip.* Catches every skip including
   registry-gated ones (`OPM_SKIP_REGISTRY_TESTS`), which the job does not mean to police, and it
   puts test knowledge in YAML.
2. *A build tag.* Needs a second compile of the suite and hides the switch from a developer who
   wants the same strictness locally.
3. *An environment switch.* Chosen. It is opt-in and a developer can set it too.

### 6. `cluster:operator` waits for `Ready=True` for the Platform's current generation

The final wait in `cluster:operator` (`Taskfile.yml:287-300`) becomes a wait on the Platform's
`Ready` condition for the Platform's current generation: it passes only when
`status.observedGeneration` equals `metadata.generation` and `Ready` is `True` (Context: the install
creates generation 1, the task's apply makes generation 2, and a `Ready=True` left from generation 1
proves nothing about the pinned catalogs). On timeout it prints both generations and the `Ready` and
`Stalled` conditions, and points at the operator log. Without this, an alpha.14-class stall reaches
the suite and costs three 3-minute timeouts before
anything fails, and the failure names a test, not the Platform.

The wait stays a shell loop rather than `kubectl wait --for=condition=Ready`, because the Platform
may not exist yet when the wait starts and `kubectl wait` fails at once on a missing object. The
loop lives in its own task, `cluster:operator:wait-ready`, which `cluster:operator` calls last: a
developer can re-run just the wait, and the loop can be exercised without a cluster by putting a
stub `kubectl` first on `PATH` (tasks.md 3.3). Its wait is `PLATFORM_READY_TIMEOUT` seconds, default
120; the knob exists so that check can run in seconds and so the first PR run's measurement can
raise the default in one place. The
timeout stays 120 seconds unless the first PR run measures a longer first reconcile on the runner.

**Alternatives considered**

1. *Wait for `Ready` only in the workflow.* Fixes CI, leaves the local loop reporting a stalled
   operator as reconciling, and splits the preparation again (Decision 3).
2. *Keep `status.operatorVersion`.* The measured false positive above.
3. *Change the task.* Chosen. This is the follow-up the archived change named and gated on the
   operator bump that has since happened.

### 7. Recommend the check be required; G4 retires behind it

The job is built to be required (Decision 2). Whether it is required is the owner's ruleset choice;
workspace RELEASING.md, section "Owner settings", does not list this check yet, and the workspace item
is asked to add it. G4 retires only when all hold: this change has merged, the check has passed on
at least one cli release PR, and it is required (owner decision 2026-10-02, RELEASING.md, "Gates").
Retiring it is its own later cli change, working name `retire-g4-operator-embed-evidence`
(proposal.md, Depends on / gates). Until then the check runs advisory, and G4 stays.

## Research & Decisions

### Can the decision be made without the API?

**Context**: A `git diff` against the merge base would avoid `pull-requests: read` and API calls.
**Explored**: `pr.yml`'s `fixtures` job already diffs against `origin/<base>` with `fetch-depth: 0`
(`pr.yml:149-163`).
**Options considered**:
1. `git diff --name-only origin/main...HEAD`: no API, but needs the full history and still cannot
   see labels.
2. API for files and labels: one permission, two calls, works on fork pull requests with the
   read-only token.
**Decision**: API for both.
**Rationale**: Labels need the API anyway. One mechanism is simpler than two.

### Which tests does the job make real?

**Context**: Only four tests use the cluster; the rest of the suite already runs in `pr.yml`'s `e2e` job.
**Explored**: The callers of `requireKindCluster` and `runOperatorOwnedOPM` listed under Context.
**Options considered**:
1. Run only the cluster-backed tests with `-run`: faster, but a new cluster test outside the regex
   silently drops out.
2. Run the whole suite: a few minutes of duplicated non-cluster tests on applying PRs.
**Decision**: Whole suite.
**Rationale**: The duplication is cheap and only paid when the job applies; a regex that drifts is
the kind of silent gap this change exists to close.

## Risks / Trade-offs

- [GHCR pull rate or outage fails the job] → Same exposure as every existing job; the job's log
  names the failing pull, and a re-run is the remedy.
- [Flaky cluster timing makes a required check flap] → The job starts advisory. The owner makes it
  required only after it has been green on real PRs, and G4 stays until then.
- [The lifecycle test is destructive and slow] → It already restores through `task cluster:operator`,
  which works in CI because the job provides `task` and the environment (Decision 3, Decision 4).
- [An unrelated label re-runs a 15-minute job] → Accepted for correctness (Decision 2).
- [Runner minutes] → Paid only on applying PRs; non-applying runs take about 15 seconds.
- [`cluster:operator` now fails where it used to pass] → Intended. A developer whose Platform is not
  Ready gets the stall reason at preparation time instead of three test timeouts.

## Migration Plan

Local gates never touch a cluster the implementer does not own. `task test` runs `test:unit`, then
`test:integration` (which needs `kind-opm-dev`, `Taskfile.yml:98-100`), then `test:e2e`, whose
lifecycle test deletes every ModuleInstance, the CRDs and the operator namespace. The shared
`kind-opm-dev` carries someone else's workload (Context), so each section's commit task replaces
`task test` with its local half: `task fmt`, `task lint`, `task test:unit`, `go vet ./...`, and
`go test ./tests/e2e/... -timeout 25m` with `OPM_E2E_REQUIRE_CLUSTER` unset and `HOME` pointed at an
empty directory (so the cluster tests skip), with `GOMODCACHE`, `GOCACHE` and `CUE_CACHE_DIR`
exported to their real values so the override does not force a cold rebuild, plus
`task openspec:check`. The cluster half of `task test` is this PR's own e2e-cluster run.

**Accepted.** The release-cascade supervisor accepted this explicit local gate in place of `task
test` on 2026-10-02, when it scheduled this change's implementation; it departs
from `openspec/config.yaml`'s "task test" gate on purpose and only for this change.

**No local cluster.** The implementing session was not allowed to create, delete or apply to any
cluster, so the planned section-1 spike (the embedded operator reaching `Ready` on a throwaway
cluster) and the throwaway-cluster check of the new `Ready` wait (including the simulated stall)
did not run locally. Both move to the PR's own run: tasks.md section 5 records what that run shows,
and the stall-simulation check is done there or by a developer on a cluster they own. If the first
run shows the embedded operator stalling, that is an alpha.14-class stall and needs a `fix(deps)`
change before this one merges; the job has then done its job.

Before merge, the PR's own run is the first real execution: the workflow file is one of its
apply paths. Record its job duration and the time `cluster:operator` took to see `Ready=True` under
Context in this file, in a commit on the PR branch, and raise the `Ready` timeout if the runner
needed more than 120 seconds. After merge, watch the check on the next operator-touching PR and the next release PR. Rollback is
deleting `e2e-cluster.yml`; the test switch is inert when unset, and the `Ready` wait can be reverted
on its own if it misfires on a healthy operator.

## Open Questions

- How long the job takes on ubuntu-latest, and whether 120 seconds covers the first Platform
  reconcile there. Both numbers come from the first PR run (Migration Plan, tasks.md section 5).
  Neither changes the approach.
