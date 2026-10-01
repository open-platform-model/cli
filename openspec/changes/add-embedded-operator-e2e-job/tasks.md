Local gate (design.md, Migration Plan), named in every commit task below: `task fmt`, `task lint`,
`task test:unit`, `go vet ./...`, `task openspec:check`, and the e2e suite with no cluster:
`HOME=$(mktemp -d) GOMODCACHE=$(go env GOMODCACHE) GOCACHE=$(go env GOCACHE) CUE_CACHE_DIR=<real cache> go test ./tests/e2e/... -timeout 25m`
with `OPM_E2E_REQUIRE_CLUSTER` unset, so the cluster tests skip. This replaces `task test`, whose
`test:integration` needs `kind-opm-dev` and whose `test:e2e` would tear down that shared cluster's
operator. The cluster half of `task test` is this PR's own e2e-cluster run, and never runs on a
cluster the implementer does not own.

Local cluster steps never touch the developer's workspace `opm-registry` container (port 5000) or
`kind-opm-dev`: they use `opm-spike-registry` on port 5001 and `CLUSTER_NAME=opm-spike`, and they
record the current kube-context first (`kubectl config current-context`) and restore it with
`kubectl config use-context` afterwards, because `kind create` switches it.

## 1. Spike: the embedded operator reaches Ready on a fresh cluster

design.md carries two unverified assumptions: that the operator embedded at spike time (record its
`PinnedOperatorVersion`) brings `Platform/cluster`, as `hack/kind-platform.yaml` pins it, to
`Ready=True` for its current generation, and how long that takes. Section 3 depends on the first; if
it fails, stop: that is an alpha.14-class stall and needs a `fix(deps)` change before this one can
continue.

- [ ] 1.1 Record the current kube-context. On a throwaway cluster that leaves `kind-opm-dev` alone, run `task cluster:create CLUSTER_NAME=opm-spike` then `time task cluster:operator CLUSTER_NAME=opm-spike`; verify with `kubectl --context kind-opm-spike get platform cluster -o jsonpath='{.metadata.generation} {.status.observedGeneration} {.status.conditions}'` that `observedGeneration` equals `generation`, that `generation` is above 1 (the task's apply changed the spec the install created), and that `Ready` is `True`; note the seconds from apply to `Ready`
- [ ] 1.2 Repeat 1.1's `cluster:operator` with a spike registry container: `docker run -d --name opm-spike-registry -p 5001:5000 registry:2`, seed it with `CUE_REGISTRY='testing.opmodel.dev=localhost:5001+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works' hack/fixtures.sh seed`, then `task cluster:operator CLUSTER_NAME=opm-spike REGISTRY_CONTAINER=opm-spike-registry KIND_CUE_REGISTRY='testing.opmodel.dev=opm-spike-registry:5000+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'`; verify the operator Deployment's args carry that `--registry`, and that the Platform is `Ready=True` with `observedGeneration` equal to `generation`
- [ ] 1.3 Tear down with `task cluster:delete CLUSTER_NAME=opm-spike` and `docker rm -f opm-spike-registry` (only that container), restore the recorded kube-context with `kubectl config use-context`, and verify `docker ps` still shows the workspace `opm-registry` untouched; write the embedded operator version, both measurements and the observed conditions and generations under design.md Context ("Spike, section 1"), and verify `openspec validate add-embedded-operator-e2e-job --strict` passes
- [ ] 1.4 Local gate green (above), then commit `chore(openspec): record the embedded-operator e2e spike`

## 2. The suite can be told the cluster is required

- [ ] 2.1 Add `skipOrFail` to `tests/e2e/operator_test.go` (design.md Decision 5), with the `OPM_E2E_REQUIRE_CLUSTER=1: ` prefix on failure; route both `t.Skipf` calls in `requireKindCluster` through it, rewording both messages so neither says "skipping" and each names the cause, the `kind-opm-dev` context and `task cluster:create`; verify `go vet ./tests/e2e/...` passes
- [ ] 2.2 Route the "could not check" `t.Skipf` in `requireOperatorApplierGrant` (`tests/e2e/instance_operator_owned_test.go`) through `skipOrFail`, dropping its trailing "skipping operator e2e" and leaving the explicit-denial `t.Fatalf` and `requireReconcilingOperator` unchanged; verify by reading the diff that no other skip in the suite changed
- [ ] 2.3 Check both `requireKindCluster` branches with `go test ./tests/e2e/ -run 'TestE2E_Operator_InstallUninstallLifecycle|TestE2E_InstanceBuild_ReadsTheNamedContextsClusterPlatform' -v`, each run with `GOMODCACHE=$(go env GOMODCACHE)`, `GOCACHE=$(go env GOCACHE)` and `CUE_CACHE_DIR` exported to their real values: (a) `HOME` an empty temp dir (no kubeconfig); (b) `HOME` a temp dir whose `.kube/config` has a context that is not `kind-opm-dev` (copy a minimal kubeconfig with one dummy context). For each, with the variable unset the tests report SKIP, and with `OPM_E2E_REQUIRE_CLUSTER=1` they report FAIL with a message starting `OPM_E2E_REQUIRE_CLUSTER=1:` and naming `kind-opm-dev` and `task cluster:create`
- [ ] 2.4 Local gate green (above; cluster-backed tests skip as before with the variable unset), then commit `test(e2e): fail instead of skip when the cluster is required`

## 3. cluster:operator waits for a Ready Platform

- [ ] 3.1 Replace the `status.operatorVersion` loop at the end of `cluster:operator` in `Taskfile.yml` with a loop that passes only when `Platform/cluster` has `status.observedGeneration` equal to `metadata.generation` and its `Ready` condition is `True` (design.md Decision 6); on success print the operator version as today, on timeout print both generations, the `Ready` and `Stalled` conditions with reason and message, and the existing log hint; keep the timeout at 120 seconds unless section 1 measured more
- [ ] 3.2 Update the comment above the loop and the `cluster:operator` summary so neither claims `status.operatorVersion` proves reconciliation; verify with `task --summary cluster:operator`
- [ ] 3.3 Verify on a throwaway cluster (`CLUSTER_NAME=opm-spike`, recording and restoring the kube-context as in section 1) that the task succeeds with `observedGeneration` equal to `generation`, and that a second run is a no-op; then simulate a stall by applying a Platform that subscribes to an unpublished catalog version (which bumps the generation over a `Ready=True` one) and verify the task's wait fails printing both generations, `Stalled` and its reason, rather than passing on the older generation's `Ready=True`; tear the cluster down and restore the kube-context
- [ ] 3.4 Local gate green (above), then commit `chore(taskfile): wait for a ready platform in cluster:operator`

## 4. The e2e-cluster workflow

- [ ] 4.1 Write `.github/scripts/e2e-cluster-applies.sh`: inputs are the event name, the head ref, a file of label names and a file of changed paths; it prints the decision report (design.md Decision 2) and writes `applies=true|false` to `$GITHUB_OUTPUT` when set, otherwise to stdout; verify locally with hand-written inputs for each spec scenario: operator path, release branch `release-please--branches--main--components--opm`, `deps-cascade` label, head ref `deps/cascade` with no label, an own input (one each of `Taskfile.yml`, `hack/fixtures.sh`, `tests/e2e/instance_build_test.go`, `tests/e2e/mod_build_test.go`), unrelated change, and `workflow_dispatch`
- [ ] 4.2 Write `.github/workflows/e2e-cluster.yml`: triggers per the `e2e-cluster-workflow` spec (`pull_request` on `main` with `opened`, `synchronize`, `reopened`, `labeled`, `unlabeled`, plus `workflow_dispatch`), `concurrency` on workflow and ref with `cancel-in-progress`, one job `E2E (kind, embedded operator)` on `ubuntu-latest` with `timeout-minutes: 45` and `permissions: { contents: read, pull-requests: read }`; step 1 an ungated shallow `actions/checkout` at the SHA `pr.yml` pins; step 2 `decide`, with `GH_REPO: ${{ github.repository }}` and every other expression in `env:`, fetching labels and paginated files with `gh` and running the script; every later step gated on its output
- [ ] 4.3 In the same job, gated steps in this order: setup-go (the pin `pr.yml` uses), setup-task (the pin `pr.yml` uses), install kind (the version and checksum `pr.yml` uses), install `cue` v0.17.1 and `crane` (a pinned `go install` of `github.com/google/go-containerregistry/cmd/crane`), build `opm`, start the `opm-registry` container, seed it, `task cluster:create`, `task cluster:operator`, then `go test ./tests/e2e/... -v -timeout 25m`, with job-level `CUE_REGISTRY`, `OPM_REGISTRY`, `KIND_CUE_REGISTRY`, `CUE_CACHE_DIR`, `OPM_BIN` and `OPM_E2E_REQUIRE_CLUSTER=1` (design.md Decision 4)
- [ ] 4.4 Add an `if: failure() && steps.decide.outputs.applies == 'true'` diagnostics step printing the operator logs, `kubectl get platform cluster -o yaml`, `kubectl get moduleinstances -A -o yaml` and recent events; verify the workflow parses with `actionlint` (run it with `go run github.com/rhysd/actionlint/cmd/actionlint@latest` if it is not installed) and that every action is pinned by commit SHA
- [ ] 4.5 Add the new workflow to the cli dev guidance where `pr.yml`'s jobs and the e2e loop are described (AGENTS.md or the doc that names `task cluster:operator`), stating when the job applies and that `OPM_E2E_REQUIRE_CLUSTER=1` reproduces it locally against a `kind-opm-dev` the developer owns and has prepared; verify with `grep -rn e2e-cluster.yml` that it is referenced
- [ ] 4.6 Local gate green (above) and actionlint clean, then commit `ci(e2e): run the cluster-backed e2e suite against the embedded operator`
