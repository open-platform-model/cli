## Why

The cli judges readiness with its own evaluator, `internal/kubernetes/health.go`: the status vocabulary (`HealthStatus` and seven constants), `EvaluateHealth`, `IsHealthy` and the `QuickInstanceHealth` fold. The library's Kubernetes tier now holds one definition of that decision in `opm/k8s/health` (library PR 199, released in `v1.0.0-beta.6`, which the cli already pins). The library package was ported from this file at cli commit `bd4d1a7c` with its rules and status strings unchanged, and its maintainer note says that until the cli deletes its copy, every fix has to be ported by hand between the two.

The owner decided that the readiness evaluator moves to `opm/k8s/health` and that the cli switches to it. A frontend that adopts a tier package carries no copy of its own and no alias, and its checks refuse a reintroduced copy (0012:D3:R6). The operator adopts the same package separately.

## What Changes

- **Delete the copy.** `internal/kubernetes/health.go` and `internal/kubernetes/health_test.go` are deleted. The library's own tests cover every rule the deleted tests covered (Deployment, StatefulSet, DaemonSet, Job, CronJob, passive kinds, PVC, custom resources, `IsHealthy`, the aggregate).
- **Use the library directly.** Every caller holds `health.Status` and calls `health.Evaluate`, `health.IsHealthy` and `health.Aggregate` from `github.com/open-platform-model/library/opm/k8s/health`, with no alias: `internal/kubernetes/{status,tree,wait,pods}.go`, `internal/operator/wait.go`, `internal/workflow/query/{status,list}.go`, `internal/output/styles.go`, the tests that use the constants, and the `tests/integration/{inst-list,inst-tree}` programs.
- **One aggregate rule.** `opm instance list` folds each instance with `health.Aggregate` where it called `QuickInstanceHealth`. `opm instance status` and the component rollup of `opm instance tree` also fold through `health.Aggregate` where each carried its own loop. The library's test `TestAggregate_MatchesTheCLILoop` proves the fold equals `QuickInstanceHealth`; the status and tree loops compute the same three-way verdict, and section 1's tests pin their results before they move.
- **Leftovers stay local.** Three private names in `health.go` serve non-health code: `workloadKinds` (which kinds `--verbose` lists pods for), `conditionStatusTrue` (the CRD `Established` predicate and the pod Ready flag) and the pod `Ready` condition type, which reused `HealthReady`. They move beside their callers; the pod check reads `corev1.PodReady` and `corev1.ConditionTrue`.
- **Pin the output.** Before the switch, golden tests pin the exact JSON and YAML that `opm instance status`, `opm instance list` and `opm instance tree` print, covering all seven status strings and the raw PVC and pod phases. The switch keeps every expected literal byte-identical.
- **Refuse a copy.** A unit test fails when any non-test Go file in the repo declares the retired evaluator names again (0012:D3:R6). A `depguard` rule cannot do this: the copy lived inside `internal/kubernetes`, which stays.
- **Docs and specs.** `AGENTS.md` names `opm/k8s/health` beside the other tier packages. The `health-export` capability, which specified the cli's own exported evaluator, is retired; the new `health-evaluation` capability states that readiness is the library's and that the status strings are stable output. `mod-apply`'s `--wait` scenario names the library predicate.

User-visible behaviour: none. Every status string, table, JSON and YAML output, exit code and `--wait` verdict stays the same.

SemVer class after GA: PATCH (an internal refactor with no change to any command, output or exported Go package; `internal/kubernetes` is internal). Before GA it ships in the next `1.0.0-beta.N`. Release class of the PR: `refactor`.

## Not in this change

- **Stall reporting.** The library's `health.ProgressDeadlineExceeded` lets a frontend tell a stalled Deployment from one still rolling out. The cli keeps reporting both as `NotReady`; using the predicate in `--wait` or `status` would be a visible change, so it is a later change if wanted.
- **Any library change or library pin change.** The pin stays `v1.0.0-beta.6`. The library's maintainer note about porting fixes to the cli's copy becomes stale once this merges; correcting it is a library follow-up.
- **`--verbose` pod listing for StatefulSets and DaemonSets.** `listWorkloadPods` documents all three workload kinds but `workloadKinds` holds only `Deployment`. This change moves the map unchanged and does not widen it.

## Capabilities

### New Capabilities

- `health-evaluation`: readiness is judged by the library's `opm/k8s/health`; the cli carries no evaluator of its own and its status strings are stable output.

### Modified Capabilities

- `health-export`: retired; every requirement specified the cli's own exported evaluator, which is deleted.
- `mod-apply`: the `--wait` scenario names `health.IsHealthy(health.Evaluate(...))` from the library.

## Impact

- Code: delete `internal/kubernetes/health.go` and `health_test.go`; switch the callers listed above; add the golden tests and the refusal test.
- API: none exported. `internal/kubernetes` loses `HealthStatus`, its constants, `EvaluateHealth`, `IsHealthy` and `QuickInstanceHealth`; `HealthyPredicate` stays.
- Output and stored data: unchanged.
- Dependencies: none new; `opm/k8s/health` is in the pinned library.
- Archive: `retire_capabilities: true` in `.openspec.yaml` retires the emptied `health-export` spec.
- Release: none of its own. It can ship in any cli release on library `v1.0.0-beta.6` or later.
