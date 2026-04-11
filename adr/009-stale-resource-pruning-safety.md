# ADR-009: Stale Resource Pruning Safety

## Status

Accepted

## Context

When a module is re-applied after removing or renaming a component, the previously applied resources from that component become stale and should be pruned.

Naive pruning — deleting everything in the old inventory that is not in the new inventory — is dangerous: renaming a component changes its OPM identity but the underlying Kubernetes resources may be identical, leading to unintended deletion of still-desired resources.

On first install (no previous inventory), rendered resources might collide with existing untracked resources or resources that are still terminating from a previous deletion.

A misconfigured module could render zero resources — if the previous inventory is non-empty, naive pruning would delete the entire deployment.

Pruning order matters: deleting a CRD before its custom resources causes cascading failures.

## Decision

Compute the stale set as previous inventory entries minus current inventory entries, using OPM identity (Group + Kind + Namespace + Name + Component).

Apply a component-rename safety check: filter the stale set to remove entries where the current set has the same Kubernetes identity (Group + Kind + Namespace + Name) but a different Component label. This prevents a component rename from triggering destructive deletion of resources that are still desired. Deleting without the safety check was rejected because component renames are a common refactoring operation that should not risk data loss.

On first install (no previous inventory), run a pre-apply existence check: verify each rendered resource does not already exist as untracked (no OPM labels) or terminating (has `deletionTimestamp`). Fail the apply if either condition is found.

Implement an empty-render safety gate: fail with an error if the current render produces zero resources and the previous inventory is non-empty, unless `--force` is provided.

Prune stale resources after successful apply, in reverse weight order (highest weight first — custom resources before CRDs). Skip pruning entirely if any apply failed, allowing convergence on retry.

Exclude namespace resources from pruning by default.

Support `--no-prune` to skip pruning while still writing the inventory.

Support `--max-history` to cap change history entries (default 10).

Do not write inventory on apply failure — this allows a retry to converge naturally without stale inventory state.

See also ADR-008 for the overall deployment model and ADR-012 for identity definitions.

## Consequences

**Positive:** Component renames are safe — authors can refactor component names without risking deletion of desired resources.

**Positive:** First-install pre-apply check prevents collisions with existing untracked or terminating resources.

**Positive:** Empty-render gate prevents accidental mass deletion from misconfigured modules.

**Positive:** Reverse-weight pruning respects dependency order (delete dependents before dependencies).

**Negative:** Multiple safety checks add complexity to the apply path.

**Negative:** Skipping inventory write on failure means a subsequent successful apply may re-prune resources that were successfully applied in the failed run.

**Trade-off:** Namespace exclusion from pruning is safe but means orphaned namespaces must be cleaned up manually.
