# ADR-012: Release Inventory Identity Model

## Status

Accepted

## Context

The release inventory tracks which Kubernetes resources are owned by a release (see ADR-008) and must detect stale resources when the set changes between applies.

Comparing inventory entries requires a definition of identity: when are two entries "the same resource"?

Kubernetes API versions change over time (e.g., `extensions/v1beta1` migrating to `apps/v1`). If Version is part of identity, an API migration would make the old-version entry appear stale and the new-version entry appear new — triggering false deletion followed by re-creation.

Component renames (see ADR-009) present a separate problem: the same Kubernetes resource (same Group/Kind/Namespace/Name) may appear under a different Component label after a refactor. The pruning safety check needs to compare resources by Kubernetes identity alone, ignoring the Component field.

## Decision

Define two distinct equality models for inventory entries:

1. **OPM identity** (Group + Kind + Namespace + Name + Component): used for stale set computation — determines whether a previously owned resource is still present in the current render.
2. **Kubernetes identity** (Group + Kind + Namespace + Name): used for the component-rename safety check — determines whether a "stale" resource is actually the same Kubernetes object under a different component name.

Exclude Version from both identity models. Including Version was rejected because it causes false orphans during Kubernetes API version migrations.

Construct inventory entries from rendered resources by extracting Group/Kind from GVK, Namespace/Name from metadata, and Component from the OPM label.

Store the inventory as a Kubernetes Secret named `opm.<release-name>.<release-id>` where release-id is a deterministic UUID v5.

Apply five labels to the inventory Secret: `app.kubernetes.io/managed-by: open-platform-model`, `module-release.opmodel.dev/name`, `module-release.opmodel.dev/namespace`, `module-release.opmodel.dev/uuid`, and `opmodel.dev/component: inventory`.

Model the inventory as current ownership only — no change history, timestamps, source paths, or remediation counters. The inventory represents what is owned now, not what happened before.

Include `ReleaseMetadata` and `ModuleMetadata` in the persisted record to enable future migration from Secrets to CRDs without changing the storage shape.

Compute inventory digest deterministically: same entries in different order produce the same digest, enabling reliable change detection.

CRUD operations: `GetInventory` tries GET by name first (common case), then falls back to label lookup (migration case). `WriteInventory` uses full PUT semantics with optimistic concurrency via `resourceVersion`. `DeleteInventory` is idempotent (404 treated as success).

See also ADR-008 for how inventory is used during deployment and ADR-009 for the pruning safety checks.

## Consequences

**Positive:** Version-excluded identity prevents false orphans during API migrations — a common Kubernetes lifecycle event.

**Positive:** Two equality models serve different purposes cleanly — OPM identity for ownership tracking, Kubernetes identity for rename safety.

**Positive:** Ownership-only model keeps the inventory simple and focused.

**Positive:** Embedded metadata and deterministic digest enable future CRD migration and efficient change detection.

**Negative:** Two identity models add conceptual complexity — contributors must understand when each applies.

**Negative:** Label-based fallback in GetInventory adds a second code path for edge cases.

**Trade-off:** No change history means debugging "what changed between applies" requires external tooling (git log, audit logs) rather than the inventory itself.
