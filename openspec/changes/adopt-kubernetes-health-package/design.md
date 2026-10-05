## Context

The library's `opm/k8s/health` (library PR 199, in `v1.0.0-beta.6`) is a rename of the cli's `internal/kubernetes/health.go`. Read side by side at the base of this change (`cd10f2d2`, cli PR 329 merged):

| cli `internal/kubernetes` | library `opm/k8s/health` | Same rules |
| --- | --- | --- |
| `HealthStatus` | `Status` | yes |
| `HealthReady`, `HealthNotReady`, `HealthComplete`, `HealthUnknown`, `HealthMissing`, `HealthApplied`, `HealthBound` | `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied`, `Bound` | same strings |
| `EvaluateHealth` and its private evaluators | `Evaluate` | yes, line for line |
| `IsHealthy` | `IsHealthy` | yes |
| `QuickInstanceHealth(resources, n)` | `Aggregate(statuses, n)` over `Evaluate` | yes (library test `TestAggregate_MatchesTheCLILoop`) |
| none | `ProgressDeadlineExceeded` | not used here |

So the evaluation itself does not change; the risks are elsewhere:

- three private names in `health.go` are read by code that is not health evaluation (`pods.go`, `wait.go`);
- two aggregate loops (`GetInstanceStatus` in `status.go`, `aggregateStatus` in `tree.go`) restate the fold by hand;
- the status strings are serialised by `opm instance status|list|tree -o json|yaml`, and a reviewer has to be able to see that not one byte moves.

Callers at the base, re-checked by grep (18 production sites): `internal/kubernetes/status.go` (`resourceHealth.Status`, `StatusResult.AggregateStatus`, the missing and unreadable rows, the aggregate switch, the summary loop, `buildResourceHealth`), `tree.go` (`Component.Status`, `ResourceNode.Status`, `buildResourceNode`, `aggregateStatus`, `replicaSetHealth`, `podToNode`), `wait.go` (`HealthyPredicate`, `CRDEstablishedPredicate`'s `conditionStatusTrue`), `pods.go` (`workloadKinds`, `HealthReady` as the pod condition type, `conditionStatusTrue`), `internal/operator/wait.go` (doc comment), `internal/workflow/query/list.go` (`instanceHealthResult.status`, `HealthUnknown`, `QuickInstanceHealth`), `internal/workflow/query/status.go` (`IsHealthy`), `internal/output/styles.go` (comment naming `kubernetes.IsHealthy`). Tests: `status_test.go`, `tree_test.go`, `diff_integration_test.go`, `query/unreadable_test.go`; programs: `tests/integration/inst-list`, `tests/integration/inst-tree`.

## Goals / Non-Goals

**Goals:**
- No cli source declares a health status type, a per-kind evaluator of inventory objects, a healthy-set rule or an aggregate fold; all are the library's. Three readiness checks are not decisions of the package and stay: the CRD `Established` predicate (`wait.go`), the operator install wait's kind dispatch (`internal/operator/wait.go`), and the display-only ReplicaSet and pod child rows of the tree (`replicaSetHealth`, `podToNode`), which never reach an aggregate.
- Every JSON and YAML byte `opm instance status`, `list` and `tree` print is unchanged, proven by golden tests written before the switch.
- A reintroduced copy fails `task test:unit`.

**Non-Goals:**
- Any change to readiness rules, exit codes, colours or `--wait` behaviour.
- Using `health.ProgressDeadlineExceeded`.

## Research & Decisions

### KH1. The library package is imported as `health`, with no alias type

Every file that needs the vocabulary imports `github.com/open-platform-model/library/opm/k8s/health` under its own name, `health`. The cli has no package of that name, unlike `inventory`, so no `k8s` prefix is needed. Struct fields change type from `HealthStatus` to `health.Status`; the JSON and YAML tags stay. `internal/kubernetes` declares no `HealthStatus = health.Status` alias and no re-exported constants (0012:D3:R6). The one local variable named `health` (`buildResourceHealth` in `status.go`) is renamed `verdict` so it does not shadow the package.

`internal/output` does not import the package. `FormatHealthStatus` keeps its string literals and its comment names `health.IsHealthy` as the healthy set: `output` is imported by every command, and importing a Kubernetes-tier package there would pull apimachinery into it for seven strings the constant table (KH4) already pins.

### KH2. Leftover private names move beside their callers, unchanged in meaning

- `workloadKinds` (only `Deployment`) moves to `pods.go` as `podListingKinds`, with a comment saying it names the kinds `--verbose` lists pods for. Its content does not change, so `--verbose` output does not change (see proposal, "Not in this change").
- `conditionStatusTrue` moves to `wait.go`, its remaining user for the CRD `Established` predicate.
- `extractPodInfoFromPod` compares `cond.Type == corev1.PodReady` and `cond.Status == corev1.ConditionTrue`. Both are the same strings it compared before (`"Ready"`, `"True"`); the old code borrowed `HealthReady` for a pod condition type, which only coincided by spelling.

### KH3. The aggregate goes through `health.Aggregate` everywhere

**Context**: three places fold statuses into one verdict.

**Explored**: the base code of `GetInstanceStatus` (`status.go`), `aggregateStatus` (`tree.go`) and `EvaluateInstanceHealth` (`list.go`), read against `health.Aggregate` in library `v1.0.0-beta.6`.

**Options considered**:
1. Switch only `list.go` (the one `QuickInstanceHealth` caller) and leave the status and tree loops. Smallest diff, but leaves two hand-written folds of a rule the library now owns.
2. Fold all three through `health.Aggregate`. Each loop computes exactly Aggregate's verdict: Unknown when nothing is counted, Ready when every counted object is healthy, NotReady otherwise.

**Decision**: option 2.

**Rationale**: 0012:D3:R6 asks for no implementation of the package's decisions, and the fold is one. Equivalence, checked by reading the base code:
- `GetInstanceStatus`: every row (live, missing as `Missing`, unreadable as `Unknown`) carries a status; `allReady` is false exactly when some row's status fails `IsHealthy` (`Missing` and `Unknown` both fail it). So `health.Aggregate(rowStatuses, 0)` gives the same verdict, and its ready count equals `Summary.Ready`. `Summary.Total` and `Summary.NotReady` stay computed as now (total minus ready).
- `aggregateStatus` in `tree.go`: Unknown when there are no evaluated nodes (both branches return Unknown), NotReady when any node fails `IsHealthy`, else Ready. That is `health.Aggregate(nodeStatuses, 0)` with the counts dropped. The function keeps its name and its depth-0 comment; its `resourceCount` parameter only fed a branch that returned the same value as the other, so it goes.
- `list.go`: `health.Aggregate(statuses of live, len(missing)+len(unreadable))`, the library's proven replacement for `QuickInstanceHealth`. The `tests/integration/inst-list` program does the same inline.

Pod rows in the tree keep their raw phase (`health.Status(info.Phase)`), as before; `aggregateStatus` only ever reads the top-level resource nodes, never pod children, so a phase string never reaches the fold.

### KH4. Byte stability is pinned before the switch

Section 1 adds golden tests on the base code, so the switch in section 2 is reviewed against literals that already passed against the old evaluator:

- `internal/kubernetes/status_golden_test.go`: `FormatStatus` of one `StatusResult` whose rows carry each of `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied`, `Bound` and a raw PVC phase `Pending`, compared whole against a JSON literal and a YAML literal.
- `internal/kubernetes/tree_golden_test.go`: `FormatTree` JSON and YAML of a tree with a component per aggregate verdict and a pod child with raw phase `Running`, compared whole.
- `internal/workflow/query/list_golden_test.go`: `RenderInstanceListOutput` JSON and YAML of summaries carrying `Ready`, `NotReady` and `Unknown`, captured from output and compared whole.
- One table test pinning each constant's string (`"Ready"`, `"NotReady"`, `"Complete"`, `"Unknown"`, `"Missing"`, `"Applied"`, `"Bound"`), so a library rename shows up as a cli test failure too.
- An aggregate table over `GetInstanceStatus` (nil client) and `aggregateStatus`: empty, all healthy, one NotReady, one missing, one unreadable, Applied-only.
- A table over `EvaluateInstanceHealth` with a fake dynamic client, since the list golden test renders hand-built summaries and never runs the fold: all healthy, one NotReady Deployment, missing plus unreadable (`NotReady (3/5)` through `formatStatusColumn`), an empty inventory (`Unknown` 0/0). The discovery-failure branch is unreachable from a test, since discovery returns no error at this base.
- `PrintInstanceStatus` on a Deployment whose `Available` is True but whose rollout is behind: row `NotReady`, exit 2.

In section 2 only identifiers in these files change (`HealthReady` becomes `health.Ready`); the task checks with `git diff` that no expected literal line changed.

### KH5. The refusal check is a parse of the repo's Go files

**Context**: 0012:D3:R6 asks the frontend's checks to refuse a reintroduced copy. The earlier adoptions refused a deleted *package path* with `depguard`. Here the copy lived in `internal/kubernetes`, which stays, so there is no import path to deny.

**Explored**: the cli's `.golangci.yml` rules (`depguard` and `forbidigo` settings) and how the `pkg/core`, `pkg/resourceorder` and `pkg/inventory` adoptions refused their copies.

**Options considered**:
1. A `depguard` or `forbidigo` rule. `depguard` works on import paths; `forbidigo` forbids uses of identifiers, not declarations, and would not catch a renamed copy either.
2. A unit test that parses every non-test `.go` file under the repo root (skipping `.git`, `.claude`, `testdata` and `vendor`) with `go/parser` and fails on a top-level declaration named `HealthStatus`, `EvaluateHealth`, `QuickInstanceHealth`, `IsHealthy`, or any function whose name starts with `evaluate` and ends with `Health`. In `internal/kubernetes`, where the copy lived, it also fails on a top-level type alias or value bound directly to a name of the library package (`type Status = health.Status`, `var Healthy = health.IsHealthy`), under whatever name the file imports it.
3. No check, relying on review.

**Decision**: option 2, in `internal/kubernetes/health_refusal_test.go`, with a doc comment citing 0012:D3:R6 and naming the library package to use instead.

**Rationale**: it runs in `task test:unit` and CI with no new tool. It is name-based, so it catches the copy coming back as it was, not a rewritten evaluator under new names; that residual is held by review, as 0012 holds other imports. The test proves it fires by running the same scans over in-memory sources that declare `func EvaluateHealth` and re-export the library's names.

### KH6. Specs: retire `health-export`, add `health-evaluation`

`health-export` specifies `internal/kubernetes` exports that no longer exist; MODIFYing it would leave its Purpose naming `health.go`. It is retired (all four requirements REMOVED, `retire_capabilities: true`), and `health-evaluation` states the new contract: the library judges readiness, the cli declares no evaluator, the status strings are stable output. `mod-apply`'s flag-surface requirement is MODIFIED with every scenario kept (OpenSpec 1.12), changing only the predicate named in the `--wait` scenario. `status-exit-codes` names `IsHealthy` without a package and stays true, so it is left alone.

## Migration Plan

No migration: no output, flag or stored data changes.

Archive checklist (the archive rides the PR): `retire_capabilities: true` retires the emptied `health-export` spec, and the archiver replaces the TBD Purpose of the new `health-evaluation` main spec with: the library's `opm/k8s/health` judges readiness for the cli, the cli keeps no evaluator of its own, and the status strings are stable output.

## Risks / Trade-offs

- **A future library change to a status string** would silently change cli output. → The constant-string table and the golden tests fail on the cli's next library bump, before release.
- **The tree's `resourceCount` parameter goes.** → It never changed the result; `tree_test.go`'s depth-0 cases pin Unknown.
- **The refusal test is name-based** (KH5). → Accepted; review holds the rest.

## Exit codes and errors

Unchanged. `opm instance status` exits 2 when `health.IsHealthy(result.AggregateStatus)` is false, as it did with the cli's `IsHealthy`. No new error is introduced.
