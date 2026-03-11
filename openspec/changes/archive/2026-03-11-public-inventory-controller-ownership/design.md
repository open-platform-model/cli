## Context

The current inventory implementation mixes two concerns in one internal package: a reusable inventory contract (`InventorySecret`, metadata, serialization, change history, Secret naming) and CLI-specific Kubernetes behavior (CRUD, discovery, pruning, logging). That has worked for the CLI, but it blocks a future Kubernetes controller from reusing the exact inventory format that the CLI already depends on for `list`, `status`, `diff`, and `delete`.

The controller use case also introduces a new coordination problem. Inventory must become a shared interoperability artifact without becoming a shared mutable ownership mechanism. The user has decided that ownership is exclusive: CLI-managed releases remain CLI-managed, controller-managed releases remain controller-managed, and neither side may silently take over the other's release. At the same time, both sides should still be able to inspect releases and surface warnings based on provenance.

## Goals / Non-Goals

**Goals:**

- Expose the reusable inventory contract in a public `pkg/inventory` package.
- Preserve the current inventory Secret name, labels, serialized keys, and change history structure.
- Add write-once provenance metadata so inventory records whether a release was created by the CLI or by a controller.
- Keep existing releases backward compatible by treating missing provenance as legacy CLI ownership.
- Update CLI mutating workflows to block takeover of controller-managed releases while preserving read-only inspection.
- Define the same ownership rule for future controllers so the policy is symmetric.

**Non-Goals:**

- Implement the future controller in this change.
- Replace the inventory Secret with a CRD.
- Redesign the full inventory persistence strategy for controllers.
- Add new flags or user-configurable ownership override paths.
- Introduce shared ownership or takeover/adoption flows.

## Decisions

### 1. Split inventory into public contract and CLI-specific operations

**Decision:** Move the reusable inventory model, Secret codec, naming helpers, provenance enum/constants, and pure change-history helpers into `pkg/inventory`. Keep Kubernetes client operations and CLI logging in internal packages for now.

**Why:** The controller needs the inventory contract, not the CLI's client wrapper or output layer. This preserves separation of concerns and creates a clean public boundary without prematurely freezing all Kubernetes helper APIs.

**Alternatives considered:**

- **Move the entire package to `pkg/inventory`**: rejected because it would publish `internal/kubernetes`-shaped APIs and CLI logging behavior that are not controller-friendly.
- **Duplicate the inventory model in controller code later**: rejected because it would split the wire contract and create drift risk.

### 2. Store provenance in release metadata, not labels

**Decision:** Add `createdBy` to `ReleaseMetadata` and treat it as the durable source of truth for ownership.

**Why:** Provenance is metadata about release origin, not a discovery selector. Keeping it in release metadata preserves the current five-label inventory contract, avoids changing label-based discovery semantics, and fits the existing write-once metadata model.

**Alternatives considered:**

- **Add a new inventory label**: rejected because current specs and tests explicitly fix the inventory Secret at five labels, and provenance is not needed for selectors.
- **Rely on managedFields / field managers**: rejected because inventory Secrets are currently written with create/update semantics rather than SSA, and field ownership would not reliably represent original creator intent.

### 3. Ownership is exclusive, with legacy inventories inferred as CLI-owned

**Decision:** Define three read states for provenance handling: `cli`, `controller`, and legacy missing field. Missing `createdBy` is interpreted as CLI-owned.

**Why:** No controller exists yet, so all existing inventories were created by the CLI. Interpreting missing provenance as legacy CLI ownership keeps upgrade behavior simple and avoids noisy warnings or forced migrations.

**Alternatives considered:**

- **Treat missing provenance as unknown**: rejected because it complicates rollout and makes legacy behavior ambiguous for no practical gain.
- **Require migration of all existing inventories**: rejected as unnecessary operational churn.

### 4. Enforce no-takeover at mutating command boundaries

**Decision:** CLI apply/delete-style workflows check inventory ownership before mutating an existing release. Read-only workflows continue to function for all releases, but they surface ownership information so users understand who manages the release.

**Why:** Ownership policy belongs at the workflow boundary where side effects happen. This keeps read-only interoperability intact while preventing accidental takeover.

**Alternatives considered:**

- **Block all CLI access to controller-managed releases**: rejected because inspection is still valuable.
- **Allow overrides or force takeover**: rejected because the user explicitly does not want takeover paths.

### 5. Keep inventory as an interoperability artifact for controllers

**Decision:** The future controller will use `pkg/inventory` to write inventory Secrets that the CLI can read, but inventory will not become the controller's primary reconciliation state.

**Why:** The controller must remain responsible for its own reconcile logic and state derivation. Inventory exists so CLI commands can discover and reason about managed resources without re-rendering or label scans.

**Alternatives considered:**

- **Make inventory the controller's source of truth**: rejected because it overloads a compatibility artifact and couples controller design too tightly to CLI history semantics.

## Risks / Trade-offs

- **[Public API creep]** -> Once `pkg/inventory` exists, its exported API becomes harder to change. Mitigation: publish only the contract and pure helpers first; keep client-specific operations internal until they stabilize.
- **[Backward-compatibility drift]** -> Small wire-format changes could break existing inventories or CLI discovery. Mitigation: preserve Secret name, keys, labels, and JSON tags; add provenance as an optional field only.
- **[Ownership checks in the wrong layer]** -> If takeover checks are scattered, behavior will drift across commands. Mitigation: centralize ownership resolution/checks in shared workflow helpers used by mutating commands.
- **[Controller semantics mismatch]** -> A future controller may write inventory differently than the CLI. Mitigation: document successful-write semantics now and keep `pkg/inventory` focused on the shared contract.

## Migration Plan

1. Extract reusable inventory types/helpers into `pkg/inventory` without changing behavior.
2. Update CLI imports to use `pkg/inventory` while preserving current inventory behavior.
3. Extend `ReleaseMetadata` with optional `createdBy` and preserve write-once semantics.
4. Teach readers to interpret missing `createdBy` as legacy CLI ownership.
5. Add ownership checks to CLI mutating workflows and ownership visibility to read-only outputs.
6. Add tests covering legacy inventories, CLI-owned releases, and controller-owned releases.

Rollback is straightforward during development: remove the provenance field usage and keep the old internal package wiring. Because the wire change is additive and optional, inventories written before rollout remain readable.

## Open Questions

- Should ownership also be mirrored into a Secret annotation for easier `kubectl` inspection, or is `releaseMetadata.createdBy` sufficient for the first iteration?
- Which read-only outputs should surface ownership most prominently: just `list` and `status`, or also `tree`/`diff`/`events` later?
- How much inventory history should the future controller preserve on each successful reconcile?
