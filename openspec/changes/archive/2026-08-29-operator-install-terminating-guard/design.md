## Context

See proposal.md. `operator.Install` builds a plan, applies each object with `kubernetes.ApplyOne`, then calls `operator.Wait` with `DefaultPredicate`. `operator.Wait` already polls a list of objects against a `ReadyPredicate`, re-reading each object from the cluster. `operator.Uninstall` deletes with `PropagationPolicy: Foreground`, so a Deployment lingers with a `foregroundDeletion` finalizer until its ReplicaSet and Pods are gone; server-side apply against such an object succeeds and the object is then removed by the garbage collector.

## Goals / Non-Goals

**Goals:** an install never applies onto a terminating object; an object that vanishes after apply fails the install at once with its name rather than as a silent timeout.

**Non-Goals:** waiting in `uninstall`; handling a terminating Namespace (uninstall keeps it; a user-deleted namespace surfaces as an apply error today and still will); any new flag.

## Decisions

**D1. One pre-apply pass on the resolved plan, before the first apply.** The guard runs after `InstallPlan`/`CRDsOnlyPlan`/RBAC merge and before the apply loop, so every plan shape is covered and nothing is applied when the wait fails. Alternative (check each object just before its own apply): leaves earlier objects applied when a later one is stuck; rejected, a partially applied install is the state the existing "stop at first error" rule already refuses.

**D2. Reuse `Wait` with an absence predicate rather than a new loop.** `Wait` polls a list until a predicate holds; the readiness predicate is a function of the live object. Absence needs the predicate to see NotFound, so `Wait` (or a thin variant) treats a NotFound read (and only NotFound; any other Get error stays pending) as "predicate satisfied" when running in absence mode, and `AbsentPredicate` returns false for any live object. Only objects observed with `deletionTimestamp` set enter the list, so an object that merely exists is never waited on. Alternative: a dedicated `waitAbsent` loop; rejected as a second poller to keep in step with `Wait`'s timeout and reporting. Alternative: switch `uninstall` to `Background` propagation so the Deployment disappears at once and the race never forms; considered and rejected. Foreground buys uninstall nothing (it never waits), but background moves the collision from the Deployment to its ReplicaSet, which keeps the same pod-template-hash name and is resolved by the Deployment controller through `collisionCount`. That is handled, but it is a messier cluster state than a clean slate, and it only covers uninstall; the guard covers any prior delete.

**D2a. Readiness mode fails fast on a vanished object.** Install has just applied every object it waits on, so a NotFound read during the readiness wait is a definitive failure, not a pending state: the object was applied and has since disappeared. Readiness mode reports that immediately, naming the object, instead of polling NotFound until the deadline. This is what makes the failure honest for every cause (a concurrent delete, a namespace deletion, the guard's own check-then-apply window), not only for the terminating case the guard catches. Get is a consistent read, so a NotFound right after a successful apply is real, not stale.

**D3. Single `--timeout` budget.** The guard consumes from the same deadline the readiness wait uses (one `context.WithTimeout` around both). A separate budget would be a new flag with no natural default; the user already reasons about one timeout per command. Because `wait` today derives its "timed out after" text from the `timeout` argument, the shared-deadline form must derive the reported duration from the real deadline (or from the elapsed time), or a guard that used 100s of a 150s budget would report "timed out after 150s" 50s into the readiness wait.

**D4. Reporting.** Each terminating object logs one resource line before the wait, `<Kind>/<ns>/<name>  waiting to finish terminating`, in the same column layout as the apply lines (`output.FormatResourceLine` with a status of `waiting to finish terminating`), so the pre-apply wait and the apply read as one list. On budget exhaustion the error names the resource and the timeout, mirroring the readiness-timeout error.

**D5. An unreadable planned object fails before apply.** The guard classifies each planned object by a Get: NotFound is "absent", a live read without `deletionTimestamp` is "live", with it "terminating". Any other Get error (RBAC denial, apiserver unavailable) leaves the object unclassified, and the guard returns `checking <Kind>/<name> before apply: <err>` with nothing applied rather than guessing. Alternative (treat it as live and proceed, as `ApplyOne` does for its own pre-apply Get): rejected, because the apply that follows would most likely hit the same error after partially applying, which is the state D1 exists to avoid; failing on the read names the cause one step earlier with a clean cluster.

## Research & Decisions

### Does SSA onto a terminating object really succeed?
**Context**: the proposal's failure mode had to be confirmed, not inferred.
**Explored**: reproduced on kind 2026-08-29: `uninstall` (13 deletes, returns at once), `install` within a second reported `Deployment ~ configured`, waited 3m, and the Deployment was absent afterwards; a second `install` created it and succeeded.
**Decision**: guard on `deletionTimestamp`. **Rationale**: it is the one field that distinguishes "exists" from "exists but doomed", and it is set the moment a delete is accepted.

## Risks / Trade-offs

- [A terminating object with a stuck finalizer makes install wait the full timeout before failing] → the error names the object; the previous behavior waited the same time and then lied about why. Acceptable.
- [The absence-mode `Wait` diverges from readiness `Wait` over time] → implement absence as a mode of the same function with a shared poll loop, covered by unit tests for both modes.
