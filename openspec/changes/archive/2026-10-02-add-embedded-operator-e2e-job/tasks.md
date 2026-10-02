Local gate (design.md, Migration Plan), named in every commit task below: `task fmt`, `task lint`,
`task test:unit`, `go vet ./...`, `task openspec:check`, and the e2e suite with no cluster:
`HOME=$(mktemp -d) GOMODCACHE=$(go env GOMODCACHE) GOCACHE=$(go env GOCACHE) CUE_CACHE_DIR=<real cache> go test ./tests/e2e/... -timeout 25m`
with `OPM_E2E_REQUIRE_CLUSTER` unset, so the cluster tests skip. This replaces `task test`, whose
`test:integration` needs `kind-opm-dev` and whose `test:e2e` would tear down that shared cluster's
operator. The cluster half of `task test` is this PR's own e2e-cluster run, and never runs on a
cluster the implementer does not own.

The local gate above was accepted by the release-cascade supervisor on 2026-10-02 in place of `task
test` (design.md, Migration Plan, "Accepted").

No step in sections 1 to 4 creates, deletes or applies to a cluster: the implementing session may
not (design.md, Migration Plan, "No local cluster"). The cluster checks the plan first put in a
local spike run on the PR's own e2e-cluster job instead, and section 6 moves them to the pull request. Any later
local cluster step never touches the developer's workspace `opm-registry` container (port 5000) or
`kind-opm-dev`: it uses `opm-spike-registry` on port 5001 and `CLUSTER_NAME=opm-spike`, records the
current kube-context first and restores it afterwards, because `kind create` switches it.

## 1. Reconcile with workspace RELEASING.md

- [x] 1.1 Cite workspace RELEASING.md sections instead of bare decision numbers; state the G4 retirement conditions as the owner decision of 2026-10-02 (RELEASING.md, "Gates"): this change merged, its check passed on at least one cli release PR, and the check required; retirement is the later change `retire-g4-operator-embed-evidence`
- [x] 1.2 Record in design.md that the explicit local gate is accepted in place of `task test`, and that the local spike is dropped because the implementing session may not touch a cluster; move its checks to section 5 (the PR's own run)
- [x] 1.3 Add the archive section (section 6), so the archive rides the implementing PR; verify `openspec validate add-embedded-operator-e2e-job --strict` passes
- [x] 1.4 Local gate green (above), then commit `docs(openspec): reconcile add-embedded-operator-e2e-job with RELEASING.md`

## 2. The suite can be told the cluster is required

- [x] 2.1 Add `skipOrFailf` to `tests/e2e/operator_test.go` (design.md Decision 5), with the `OPM_E2E_REQUIRE_CLUSTER=1: ` prefix on failure; route both `t.Skipf` calls in `requireKindCluster` through it, rewording both messages so neither says "skipping" and each names the cause, the `kind-opm-dev` context and `task cluster:create`; verify `go vet ./tests/e2e/...` passes
- [x] 2.2 Route the "could not check" `t.Skipf` in `requireOperatorApplierGrant` (`tests/e2e/instance_operator_owned_test.go`) through `skipOrFailf`, dropping its trailing "skipping operator e2e" and leaving the explicit-denial `t.Fatalf` and `requireReconcilingOperator` unchanged; verify by reading the diff that no other skip in the suite changed
- [x] 2.3 Check both `requireKindCluster` branches with `go test ./tests/e2e/ -run 'TestE2E_Operator_InstallUninstallLifecycle|TestE2E_InstanceBuild_ReadsTheNamedContextsClusterPlatform' -v`, each run with `GOMODCACHE=$(go env GOMODCACHE)`, `GOCACHE=$(go env GOCACHE)` and `CUE_CACHE_DIR` exported to their real values: (a) `HOME` an empty temp dir (no kubeconfig); (b) `HOME` a temp dir whose `.kube/config` has a context that is not `kind-opm-dev` (copy a minimal kubeconfig with one dummy context). For each, with the variable unset the tests report SKIP, and with `OPM_E2E_REQUIRE_CLUSTER=1` they report FAIL with a message starting `OPM_E2E_REQUIRE_CLUSTER=1:` and naming `kind-opm-dev` and `task cluster:create`
- [x] 2.4 Local gate green (above; cluster-backed tests skip as before with the variable unset), then commit `test(e2e): fail instead of skip when the cluster is required`

## 3. cluster:operator waits for a Ready Platform

- [x] 3.1 Replace the `status.operatorVersion` loop at the end of `cluster:operator` in `Taskfile.yml` with a loop that passes only when `Platform/cluster` has `status.observedGeneration` equal to `metadata.generation` and its `Ready` condition is `True` (design.md Decision 6); on success print the operator version as today, on timeout print both generations, the `Ready` and `Stalled` conditions with reason and message, and the existing log hint; keep the timeout at 120 seconds (section 5 raises it if the first PR run needs more)
- [x] 3.2 Update the comment above the loop and the `cluster:operator` summary so neither claims `status.operatorVersion` proves reconciliation; verify with `task --summary cluster:operator`
- [x] 3.3 Without a cluster, exercise the new wait against a stub `kubectl` placed first on `PATH` (a scratch script answering the loop's `get platform cluster` queries from fixed JSON), driving the loop as `task cluster:operator` runs it: (a) `generation` 2, `observedGeneration` 2, `Ready=True` passes and prints the operator version; (b) `generation` 2, `observedGeneration` 1, `Ready=True` (left from generation 1) keeps waiting and fails printing both generations, `Stalled` and its reason; (c) `Ready=False`, `Stalled=True` reason `MaterializeFailed` for the current generation fails naming that reason; (d) no Platform yet keeps waiting rather than failing at once. Use a short timeout override for the runs if the loop takes one, else run the loop body extracted to a scratch script
- [x] 3.4 Local gate green (above), then commit `chore(taskfile): wait for a ready platform in cluster:operator`

## 4. The e2e-cluster workflow

- [x] 4.1 Write `.github/scripts/e2e-cluster-applies.sh`: inputs are the event name, the head ref, a file of label names and a file of changed paths; it prints the decision report (design.md Decision 2) and writes `applies=true|false` to `$GITHUB_OUTPUT` when set, otherwise to stdout; verify locally with hand-written inputs for each spec scenario: operator path, release branch `release-please--branches--main--components--opm`, `deps-cascade` label, head ref `deps/cascade` with no label, an own input (one each of `Taskfile.yml`, `hack/fixtures.sh`, `tests/e2e/instance_build_test.go`, `tests/e2e/mod_build_test.go`), unrelated change, and `workflow_dispatch`
- [x] 4.2 Write `.github/workflows/e2e-cluster.yml`: triggers per the `e2e-cluster-workflow` spec (`pull_request` on `main` with `opened`, `synchronize`, `reopened`, `labeled`, `unlabeled`, plus `workflow_dispatch`), `concurrency` on workflow and ref with `cancel-in-progress`, one job `E2E (kind, embedded operator)` on `ubuntu-latest` with `timeout-minutes: 45` and `permissions: { contents: read, pull-requests: read }`; step 1 an ungated shallow `actions/checkout` at the SHA `pr.yml` pins; step 2 `decide`, with `GH_REPO: ${{ github.repository }}` and every other expression in `env:`, fetching labels and paginated files with `gh` and running the script; every later step gated on its output
- [x] 4.3 In the same job, gated steps in this order: setup-go (the pin `pr.yml` uses), setup-task (the pin `pr.yml` uses), install kind (the version and checksum `pr.yml` uses), install `cue` v0.17.1 and `crane` (a pinned `go install` of `github.com/google/go-containerregistry/cmd/crane`), build `opm`, start the `opm-registry` container, seed it, `task cluster:create`, `task cluster:operator`, then `go test ./tests/e2e/... -v -timeout 25m`, with job-level `CUE_REGISTRY`, `OPM_REGISTRY`, `KIND_CUE_REGISTRY`, `CUE_CACHE_DIR`, `OPM_BIN` and `OPM_E2E_REQUIRE_CLUSTER=1` (design.md Decision 4)
- [x] 4.4 Add an `if: failure() && steps.decide.outputs.applies == 'true'` diagnostics step printing the operator logs, `kubectl get platform cluster -o yaml`, `kubectl get moduleinstances -A -o yaml` and recent events; verify the workflow parses with `actionlint` (run it with `go run github.com/rhysd/actionlint/cmd/actionlint@latest` if it is not installed) and that every action is pinned by commit SHA
- [x] 4.5 Add the new workflow to the cli dev guidance where `pr.yml`'s jobs and the e2e loop are described (AGENTS.md or the doc that names `task cluster:operator`), stating when the job applies and that `OPM_E2E_REQUIRE_CLUSTER=1` reproduces it locally against a `kind-opm-dev` the developer owns and has prepared; verify with `grep -rn e2e-cluster.yml` that it is referenced
- [x] 4.6 Local gate green (above) and actionlint clean, then commit `ci(e2e): run the cluster-backed e2e suite against the embedded operator`

## 5. Fixes from the implementation review

- [x] 5.1 Capture failure evidence while it exists: in `tests/e2e/instance_operator_owned_test.go`, register `logOperatorOwnedDiagnostics` as a `t.Cleanup` right after each `resetOperatorOwnedInstance` cleanup (per subtest in `TestE2E_Delete_OperatorOwnedDelegates`, whose subtests reset the instance in turn), logging on `t.Failed()` the test's ModuleInstance, `Platform/cluster` and the operator's last 300 log lines; name the `Ready` condition in `makeOperatorOwned`'s timeout message; keep the workflow's diagnostics step for preparation failures (design.md Decision 5); verify `go vet ./tests/e2e/...` and lint pass, then commit `test(e2e): log operator-owned failure evidence before cleanup`
- [x] 5.2 Add `^internal/cmd/operator/` to the decision script's apply paths and the dev guidance; verify a change to only `internal/cmd/operator/install.go` applies, then commit `ci(e2e): apply the cluster job to operator command changes`
- [x] 5.3 Make a pull request that is no longer open never apply: the decide step reads the state live and passes `PR_STATE`; verify a merged release-please PR reports "not applicable: pull request is closed" and an open one still applies, then commit `ci(e2e): skip the cluster job on closed pull requests`
- [x] 5.4 In `cluster:operator:wait-ready`, read each condition's status, reason and message separately so an absent condition prints `<not set>`, and name `BuildFailed` (`MaterializeFailed` on alpha.14) as the example stall reason; verify against a stub `kubectl`, then commit `chore(taskfile): print absent platform conditions as not set`
- [x] 5.5 Update the specs, proposal and design for 5.1 to 5.4, cite RELEASING.md "Owner settings" › "Rulesets on main", drop the stale "being reconciled" and spike wording, and class `hack/platform/` as an own input with its reason; verify `openspec validate add-embedded-operator-e2e-job --strict`, then commit `chore(openspec): revise add-embedded-operator-e2e-job after implementation review`

## 6. The first PR run proves the job (moved to the pull request)

The PR's own e2e-cluster run is the first execution on a real cluster (design.md, Migration Plan).
The archive rides the PR and is committed before that run, so these checks moved to the pull
request body as merge gates and post-merge checks instead of commits on this change.

- [x] 6.1 Moved to the PR body (merge gate): the PR's own "E2E (kind, embedded operator)" run must be green before merge, with no cluster-backed test skipped; its embedded operator version, job duration and seconds to `Ready=True` are noted on the PR. If the run stalls, that is an alpha.14-class stall and needs a `fix(deps)` change first
- [x] 6.2 Moved to the PR body (merge gate): if the runner needed more than 120 seconds to reach `Ready`, raise `PLATFORM_READY_TIMEOUT`'s default in `Taskfile.yml` in a commit on the PR branch before merge
- [x] 6.3 Moved to the PR body (post-merge human check): on a cluster the developer owns (never the shared `kind-opm-dev` while it carries other workloads), simulate a stall by applying a Platform that subscribes to an unpublished catalog version, and verify `task cluster:operator`'s wait fails printing both generations, `Stalled` and its reason

## 7. Archive

- [x] 7.1 Archive the change on this branch (openspec archive), so the archive rides the implementing PR; never push to main (owner decision 2026-10-01, RELEASING.md, "Owner settings"); verify `task openspec:check` passes, then commit `chore(openspec): archive add-embedded-operator-e2e-job`
