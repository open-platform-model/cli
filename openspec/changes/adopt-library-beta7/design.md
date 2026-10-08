## Context

See proposal.md for the motivation. The cli holds no weight table of its own: `internal/kubernetes.SortObjects` (apply, delete, tree), `internal/inventory` (prune) and `internal/output` (`module build` documents) all read `opm/k8s/object.Weight`. The bump therefore changes the order with no cli code edit. The only failing check on the bumped pin is `internal/kubernetes/order_parity_test.go` (`TestWeightTableMatchesRetiredCopy`, `TestSortMatchesRetiredCopy`); every other unit test passes on beta.7.

## Goals / Non-Goals

**Goals:**

- The cli builds and passes its unit tests on library v1.0.0-beta.7 with the pin set of the cascade branch.
- The test keeps its purpose: a library release that moves a weight fails on the bump PR, so an order change stays a reviewed edit in the cli.

**Non-Goals:**

- Adopting `opm/k8s/ownership` or `opm/k8s/lifecycle`.
- Replacing the last message-text match (`internal/publish/compat.go`, `unprovidedImport`) with `*ResolutionError`.
- Any change under `internal/cmd/` or to `internal/kubernetes/resource.go`.

## Decisions

### The bump runs through the repo's cascade task

`task -x deps:cascade` in the work tree, not a hand edit and not the cascade branch's commits. The result MUST equal `git diff origin/main...origin/deps/cascade` (two files). Alternative: `go get` by hand. Rejected: the task is the repo's one writer of these pins and also checks the other lanes are current.

### The test pins the library's table as the cli's reviewed copy

The literals are rewritten to the beta.7 values, and the names drop "retired": the retired cli table is history, and the test no longer compares against it. The test SHALL cover every `Weight*` constant (the three new ones included), every exact group-version-kind row, the three cluster definitions in a version the exact rows do not hold, every kind-only row, the `*Class` suffix rule and the default. The sort test's object set gains one object for each new weight class (ClusterRoleBinding, ResourceQuota, LimitRange, CronJob) and a PersistentVolumeClaim, so the expected sequences show each moved class. Alternative: delete the test and rely on the library's own guard. Rejected: the brief and the test's stated purpose are that the cli reviews an order change.

### The spec requirement is replaced, not modified

The scenario "The library table equals the table the CLI applied by" cannot stay, and OpenSpec refuses a MODIFIED requirement that drops a scenario. The delta removes the requirement and adds one under a new name.

### No enhancement declaration

The decision this follows (0012:D5:R1) was delivered by the library change. The cli change only takes the release, so it writes no `enhancement.yaml`.

## Research & Decisions

### Other breaking changes of beta.7

**Context**: the library CHANGELOG at v1.0.0-beta.7 lists three breaking changes and several features; each must be checked against the cli.
**Explored**: the library diff v1.0.0-beta.6..v1.0.0-beta.7, the cli's imports of the library, and a probe module built with a beta.6 and a beta.7 cli.
**Findings**:

1. Adopt rule (`ownership.CanApply`, `ownership.CanDelete`): no effect today. The cli imports neither `opm/k8s/ownership` nor `opm/k8s/lifecycle`.
2. Lifecycle package: new package, not imported. No effect today.
3. Typed resolution errors: `*ResolutionError` wraps author-defect resolution failures with the cause's message unchanged, and the fetch forms are still matched first. `internal/cuemod` reads only `*FetchError` kinds, which do not change; `unprovidedImport` in `internal/publish/compat.go` checks "not a FetchError" plus the message text, both still true. No effect today.
4. Required values refused by the kernel: user-visible, no cli edit needed to keep building. `opm module vet` already refused such values on beta.6. `opm module build` (and the apply paths, which render the same way) now refuse too. The refusal at build prints `incomplete value string` without the path `values.<field>` that `module vet` prints.

**Decision**: no code edit for 1 to 4. Finding 4's missing path is a follow-up, not part of this change.
**Rationale**: the change must alter nothing beta.7 does not force.

## Risks / Trade-offs

- [PersistentVolumeClaims now apply after the workloads that mount them, and delete before them] → Kubernetes tolerates both: a Pod waits for its claim, and the claim's protection finalizer holds the delete until no Pod uses it. This is the owner's accepted order.
- [NetworkPolicy and Ingress apply after workloads] → a short window where a new workload runs before its policy exists. Same order as the operator's Flux apply.
- [Webhook configurations apply last and delete first] → custom resources of the same instance are applied before the webhook that validates them exists.
- [The cluster-backed e2e suite does not run for this branch name] → the PR needs the `deps-cascade` label so that `E2E (kind, embedded operator)` does its work.
