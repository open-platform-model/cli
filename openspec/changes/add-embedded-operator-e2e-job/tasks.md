## 1. Spike: the embedded operator reaches Ready on a fresh cluster

design.md carries two unverified assumptions: that the embedded `v1.0.0-beta.2` operator brings
`Platform/cluster` (as `hack/kind-platform.yaml` pins it) to `Ready=True`, and how long that takes.
Section 3 depends on the first; if it fails, stop: that is an H3-class finding and needs a `fix(deps)`
change before this one can continue.

- [ ] 1.1 On a throwaway cluster that leaves `kind-opm-dev` alone, run `task cluster:create CLUSTER_NAME=opm-spike` then `time task cluster:operator CLUSTER_NAME=opm-spike`; verify with `kubectl --context kind-opm-spike get platform cluster -o jsonpath='{.status.conditions}'` that `Ready` is `True`, and note the seconds from apply to `Ready`
- [ ] 1.2 Repeat 1.1's `cluster:operator` with a registry container: `docker run -d --name opm-registry -p 5000:5000 registry:2`, seed it with `CUE_REGISTRY='testing.opmodel.dev=localhost:5000+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works' hack/fixtures.sh seed`, then `task cluster:operator CLUSTER_NAME=opm-spike KIND_CUE_REGISTRY='testing.opmodel.dev=opm-registry:5000+insecure,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'`; verify the operator Deployment's args carry that `--registry` and the Platform is still `Ready=True`
- [ ] 1.3 Tear down with `task cluster:delete CLUSTER_NAME=opm-spike` and `docker rm -f opm-registry`, write both measurements and the observed conditions under design.md Context ("Spike, section 1"), and verify `openspec validate add-embedded-operator-e2e-job --strict` passes
- [ ] 1.4 `task lint` and `task openspec:check` green, then commit `chore(openspec): record the embedded-operator e2e spike`

## 2. The suite can be told the cluster is required

- [ ] 2.1 Add `skipOrFail` to `tests/e2e/operator_test.go` (design.md Decision 5) and route both `t.Skipf` calls in `requireKindCluster` through it; verify `go vet ./tests/e2e/...` passes
- [ ] 2.2 Route the "could not check" `t.Skipf` in `requireOperatorApplierGrant` (`tests/e2e/instance_operator_owned_test.go`) through `skipOrFail`, leaving the explicit-denial `t.Fatalf` and `requireReconcilingOperator` unchanged; verify by reading the diff that no other skip in the suite changed
- [ ] 2.3 With no reachable `kind-opm-dev` (point `HOME` at an empty temp dir for the run), verify `go test ./tests/e2e/ -run 'TestE2E_Operator_InstallUninstallLifecycle|TestE2E_InstanceBuild_ReadsTheNamedContextsClusterPlatform' -v` reports SKIP, and the same command with `OPM_E2E_REQUIRE_CLUSTER=1` reports FAIL naming `task cluster:create`
- [ ] 2.4 `task lint` and `task test` green (cluster-backed tests skip or pass as before with the variable unset), then commit `test(e2e): fail instead of skip when the cluster is required`

## 3. cluster:operator waits for a Ready Platform

- [ ] 3.1 Replace the `status.operatorVersion` loop at the end of `cluster:operator` in `Taskfile.yml` with a loop on the `Ready` condition of `Platform/cluster` (design.md Decision 6); on success print the operator version as today, on timeout print the `Ready` and `Stalled` conditions with reason and message and the existing log hint; keep the timeout at 120 seconds unless section 1 measured more
- [ ] 3.2 Update the comment above the loop and the `cluster:operator` summary so neither claims `status.operatorVersion` proves reconciliation; verify with `task --summary cluster:operator`
- [ ] 3.3 Verify on a throwaway cluster (`CLUSTER_NAME=opm-spike`, as in 1.1) that the task succeeds and a second run is a no-op; then simulate a stall by applying a Platform that subscribes to an unpublished catalog version and verify the task fails printing `Stalled` and its reason; tear the cluster down
- [ ] 3.4 `task lint` and `task test` green, then commit `chore(taskfile): wait for a ready platform in cluster:operator`

## 4. The e2e-cluster workflow

- [ ] 4.1 Write `.github/scripts/e2e-cluster-applies.sh`: inputs are the event name, the head ref, a file of label names and a file of changed paths; it prints the decision report (design.md Decision 2) and writes `applies=true|false` to `$GITHUB_OUTPUT` when set, otherwise to stdout; verify locally with hand-written inputs for each spec scenario (operator path, release branch, `deps-cascade` label, own input, unrelated change, `workflow_dispatch`)
- [ ] 4.2 Write `.github/workflows/e2e-cluster.yml`: triggers and permissions per the `e2e-cluster-workflow` spec, `concurrency` on workflow and ref with `cancel-in-progress`, one job `E2E (kind, embedded operator)` on `ubuntu-latest` with `timeout-minutes: 45`; a decide step that fetches labels and paginated files with `gh` and runs the script; every later step gated on its output
- [ ] 4.3 In the same job, gated steps in this order: checkout and setup-go (the pins `pr.yml` uses), setup-task (the pin `pr.yml` uses), install kind (the version and checksum `pr.yml` uses), install `cue` v0.17.1 and `crane` (a pinned `go install` of `github.com/google/go-containerregistry/cmd/crane`), build `opm`, start the `opm-registry` container, seed it, `task cluster:create`, `task cluster:operator`, then `go test ./tests/e2e/... -v -timeout 25m`, with job-level `CUE_REGISTRY`, `OPM_REGISTRY`, `KIND_CUE_REGISTRY`, `CUE_CACHE_DIR`, `OPM_BIN` and `OPM_E2E_REQUIRE_CLUSTER=1` (design.md Decision 4)
- [ ] 4.4 Add an `if: failure() && steps.decide.outputs.applies == 'true'` diagnostics step printing the operator logs, `kubectl get platform cluster -o yaml`, `kubectl get moduleinstances -A -o yaml` and recent events; verify the workflow parses with `actionlint` (run it with `go run github.com/rhysd/actionlint/cmd/actionlint@latest` if it is not installed) and that every action is pinned by commit SHA
- [ ] 4.5 Add the new workflow to the cli dev guidance where `pr.yml`'s jobs and the e2e loop are described (AGENTS.md or the doc that names `task cluster:operator`), stating when the job applies and that `OPM_E2E_REQUIRE_CLUSTER=1` reproduces it locally against a prepared `kind-opm-dev`; verify with `grep -rn e2e-cluster.yml` that it is referenced
- [ ] 4.6 `task lint`, `task test` and `task openspec:check` green, then commit `ci(e2e): run the cluster-backed e2e suite against the embedded operator`
