# ADR-008: Inventory-First Deployment

## Status

Accepted

## Context

The CLI needs to apply rendered Kubernetes resources to a cluster and later delete or inspect them. Discovering which resources belong to an instance can be done two ways: re-rendering the module from source and comparing, or reading a persisted inventory of previously applied resources.

Re-rendering requires source files to be available at delete/inspect time, which is not always the case — users may delete a module's source after deploying it.

Label-based cluster scanning (listing all resources with a given label) is imprecise: it cannot distinguish resources created by the instance from resources created by controllers or operators that inherited the labels.

Apply operations need conflict handling and idempotency guarantees.

## Decision

Use inventory-first resource discovery: `mod delete`, `mod status`, `mod tree`, and `mod list` read the persisted instance inventory to enumerate owned resources. Label-based scanning is a fallback only when no inventory exists.

Re-rendering for resource discovery was rejected because it requires source files at delete/inspect time and is slower than reading a stored inventory.

Use Kubernetes server-side apply (SSA) with force conflicts for all apply operations. Client-side apply was rejected because SSA is idempotent and handles field ownership conflicts.

Apply resources in ascending weight order (CRDs before CRs, ConfigMaps before Deployments); delete in descending weight order.

Write a persisted instance inventory record (as a Kubernetes Secret) after each successful apply, containing the current owned resource set and deployed module version. Failed applies skip the inventory write to allow convergence on retry.

Refuse deletion of controller-managed instances (`createdBy: "controller"`) to prevent CLI/operator conflicts.

Refuse applying an empty render result when a previous inventory exists (safety gate against accidental mass deletion), unless `--force` is provided.

Support `--no-prune` to skip stale resource removal while still writing the inventory.

See also ADR-009 for the pruning safety checks and ADR-012 for the inventory identity model.

## Consequences

**Positive:** Delete and inspect commands work without source files — users can clean up after removing module source.

**Positive:** Inventory-based discovery is precise — no false positives from inherited labels.

**Positive:** SSA with force provides idempotent, conflict-resilient applies.

**Positive:** Empty render safety gate prevents accidental mass deletion from misconfigured modules.

**Negative:** Apply becomes stateful — each apply writes an inventory Secret, adding operational overhead.

**Negative:** Two discovery paths (inventory-first, label-fallback) create more code paths to maintain.

**Trade-off:** Refusing controller-managed instance deletion protects against CLI/operator conflicts but means users must use the controller to manage those instances.
