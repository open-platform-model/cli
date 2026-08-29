## Context

See proposal.md. `operator.Install` builds a plan, applies each object with `kubernetes.ApplyOne`, then calls `operator.Wait` with `DefaultPredicate`. `operator.Wait` already polls a list of objects against a `ReadyPredicate`, re-reading each object from the cluster. `operator.Uninstall` deletes with `PropagationPolicy: Foreground`, so a Deployment lingers with a `foregroundDeletion` finalizer until its ReplicaSet and Pods are gone; server-side apply against such an object succeeds and the object is then removed by the garbage collector.

## Goals / Non-Goals

**Goals:** an install never applies onto a terminating object; the failure mode, when it does occur, is named rather than a silent timeout on a vanished resource.

**Non-Goals:** waiting in `uninstall`; handling a terminating Namespace (uninstall keeps it; a user-deleted namespace surfaces as an apply error today and still will); any new flag.

## Decisions

**D1. One pre-apply pass on the resolved plan, before the first apply.** The guard runs after `InstallPlan`/`CRDsOnlyPlan`/RBAC merge and before the apply loop, so every plan shape is covered and nothing is applied when the wait fails. Alternative (check each object just before its own apply): leaves earlier objects applied when a later one is stuck; rejected, a partially applied install is the state the existing "stop at first error" rule already refuses.

**D2. Reuse `Wait` with an absence predicate rather than a new loop.** `Wait` polls a list until a predicate holds; the readiness predicate is a function of the live object. Absence needs the predicate to see NotFound, so `Wait` (or a thin variant) treats NotFound as "predicate satisfied" when running in absence mode, and `AbsentPredicate` returns false for any live object. Only objects observed with `deletionTimestamp` set enter the list, so an object that merely exists is never waited on. Alternative: a dedicated `waitAbsent` loop; rejected as a second poller to keep in step with `Wait`'s timeout and reporting.

**D3. Single `--timeout` budget.** The guard consumes from the same deadline the readiness wait uses (one `context.WithTimeout` around both). A separate budget would be a new flag with no natural default; the user already reasons about one timeout per command.

**D4. Reporting.** Each terminating object logs one line `waiting for <Kind>/<ns>/<name> to finish terminating` before the wait, in the same resource-line format as apply. On budget exhaustion the error names the resource and the timeout, mirroring the readiness-timeout error.

## Research & Decisions

### Does SSA onto a terminating object really succeed?
**Context**: the proposal's failure mode had to be confirmed, not inferred.
**Explored**: reproduced on kind 2026-08-29: `uninstall` (13 deletes, returns at once), `install` within a second reported `Deployment ~ configured`, waited 3m, and the Deployment was absent afterwards; a second `install` created it and succeeded.
**Decision**: guard on `deletionTimestamp`. **Rationale**: it is the one field that distinguishes "exists" from "exists but doomed", and it is set the moment a delete is accepted.

## Risks / Trade-offs

- [A terminating object with a stuck finalizer makes install wait the full timeout before failing] → the error names the object; the previous behavior waited the same time and then lied about why. Acceptable.
- [The absence-mode `Wait` diverges from readiness `Wait` over time] → implement absence as a mode of the same function with a shared poll loop, covered by unit tests for both modes.
