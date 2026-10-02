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
local spike run on the PR's own e2e-cluster job instead, and section 5 records them. Any later
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

- [ ] 3.1 Replace the `status.operatorVersion` loop at the end of `cluster:operator` in `Taskfile.yml` with a loop that passes only when `Platform/cluster` has `status.observedGeneration` equal to `metadata.generation` and its `Ready` condition is `True` (design.md Decision 6); on success print the operator version as today, on timeout print both generations, the `Ready` and `Stalled` conditions with reason and message, and the existing log hint; keep the timeout at 120 seconds (section 5 raises it if the first PR run needs more)
- [ ] 3.2 Update the comment above the loop and the `cluster:operator` summary so neither claims `status.operatorVersion` proves reconciliation; verify with `task --summary cluster:operator`
- [ ] 3.3 Without a cluster, exercise the new wait against a stub `kubectl` placed first on `PATH` (a scratch script answering the loop's `get platform cluster` queries from fixed JSON), driving the loop as `task cluster:operator` runs it: (a) `generation` 2, `observedGeneration` 2, `Ready=True` passes and prints the operator version; (b) `generation` 2, `observedGeneration` 1, `Ready=True` (left from generation 1) keeps waiting and fails printing both generations, `Stalled` and its reason; (c) `Ready=False`, `Stalled=True` reason `MaterializeFailed` for the current generation fails naming that reason; (d) no Platform yet keeps waiting rather than failing at once. Use a short timeout override for the runs if the loop takes one, else run the loop body extracted to a scratch script
- [ ] 3.4 Local gate green (above), then commit `chore(taskfile): wait for a ready platform in cluster:operator`

## 4. The e2e-cluster workflow

- [ ] 4.1 Write `.github/scripts/e2e-cluster-applies.sh`: inputs are the event name, the head ref, a file of label names and a file of changed paths; it prints the decision report (design.md Decision 2) and writes `applies=true|false` to `$GITHUB_OUTPUT` when set, otherwise to stdout; verify locally with hand-written inputs for each spec scenario: operator path, release branch `release-please--branches--main--components--opm`, `deps-cascade` label, head ref `deps/cascade` with no label, an own input (one each of `Taskfile.yml`, `hack/fixtures.sh`, `tests/e2e/instance_build_test.go`, `tests/e2e/mod_build_test.go`), unrelated change, and `workflow_dispatch`
- [ ] 4.2 Write `.github/workflows/e2e-cluster.yml`: triggers per the `e2e-cluster-workflow` spec (`pull_request` on `main` with `opened`, `synchronize`, `reopened`, `labeled`, `unlabeled`, plus `workflow_dispatch`), `concurrency` on workflow and ref with `cancel-in-progress`, one job `E2E (kind, embedded operator)` on `ubuntu-latest` with `timeout-minutes: 45` and `permissions: { contents: read, pull-requests: read }`; step 1 an ungated shallow `actions/checkout` at the SHA `pr.yml` pins; step 2 `decide`, with `GH_REPO: ${{ github.repository }}` and every other expression in `env:`, fetching labels and paginated files with `gh` and running the script; every later step gated on its output
- [ ] 4.3 In the same job, gated steps in this order: setup-go (the pin `pr.yml` uses), setup-task (the pin `pr.yml` uses), install kind (the version and checksum `pr.yml` uses), install `cue` v0.17.1 and `crane` (a pinned `go install` of `github.com/google/go-containerregistry/cmd/crane`), build `opm`, start the `opm-registry` container, seed it, `task cluster:create`, `task cluster:operator`, then `go test ./tests/e2e/... -v -timeout 25m`, with job-level `CUE_REGISTRY`, `OPM_REGISTRY`, `KIND_CUE_REGISTRY`, `CUE_CACHE_DIR`, `OPM_BIN` and `OPM_E2E_REQUIRE_CLUSTER=1` (design.md Decision 4)
- [ ] 4.4 Add an `if: failure() && steps.decide.outputs.applies == 'true'` diagnostics step printing the operator logs, `kubectl get platform cluster -o yaml`, `kubectl get moduleinstances -A -o yaml` and recent events; verify the workflow parses with `actionlint` (run it with `go run github.com/rhysd/actionlint/cmd/actionlint@latest` if it is not installed) and that every action is pinned by commit SHA
- [ ] 4.5 Add the new workflow to the cli dev guidance where `pr.yml`'s jobs and the e2e loop are described (AGENTS.md or the doc that names `task cluster:operator`), stating when the job applies and that `OPM_E2E_REQUIRE_CLUSTER=1` reproduces it locally against a `kind-opm-dev` the developer owns and has prepared; verify with `grep -rn e2e-cluster.yml` that it is referenced
- [ ] 4.6 Local gate green (above) and actionlint clean, then commit `ci(e2e): run the cluster-backed e2e suite against the embedded operator`

## 5. The first PR run proves the job (after the branch is pushed)

The PR's own e2e-cluster run is the first execution on a real cluster (design.md, Migration Plan).
These tasks are done on the PR branch once that run has finished.

- [ ] 5.1 From the PR's first applying run, record under design.md Context ("First PR run"): the embedded `PinnedOperatorVersion`, the job duration, the seconds `task cluster:operator` took to see `Ready=True`, the Platform's `generation` and `observedGeneration` (generation above 1, both equal), and the suite's pass/skip/fail counts (no cluster-backed test skipped); if the run stalled, stop: that is an alpha.14-class stall and needs a `fix(deps)` change first
- [ ] 5.2 If the runner needed more than 120 seconds to reach `Ready`, raise the wait in `Taskfile.yml` and note the new value in design.md Decision 6
- [ ] 5.3 On a cluster the developer owns (never the shared `kind-opm-dev` while it carries other workloads), simulate a stall by applying a Platform that subscribes to an unpublished catalog version, and verify `task cluster:operator`'s wait fails printing both generations, `Stalled` and its reason; record the result in design.md
- [ ] 5.4 Local gate green (above), then commit `chore(openspec): record the first e2e-cluster run`

## 6. Archive

- [ ] 6.1 Archive the change on this branch (openspec archive), so the archive rides the implementing PR; never push to main (owner decision 2026-10-01, RELEASING.md, "Owner settings"); verify `task openspec:check` passes, then commit `chore(openspec): archive add-embedded-operator-e2e-job`
