## Context

The previous inventory extraction change made `pkg/inventory` public, but it preserved a history-oriented CLI persistence model: release metadata, module metadata, a newest-first change index, and per-change history entries with source path/version, raw values, manifest digest, timestamp, and inventory entries.

That was a useful intermediate step, but it is not the contract we want controller code to build against. The controller use case makes the boundary clearer:

- inventory should represent current ownership only
- source/config/render/history state should live in release status
- storage should be treated as an implementation detail, not the shape of the public inventory API

The CLI still needs inventory for apply/prune and inventory-scoped discovery, but it does not need the public inventory package to keep exposing history and Secret-specific concepts forever.

## Goals / Non-Goals

**Goals:**

- Make the public `pkg/inventory` package model current owned resources only.
- Keep the inventory contract suitable for embedding in future `ModuleRelease.status.inventory` and `BundleRelease.status.inventory`.
- Preserve the identity and stale-set semantics the CLI already depends on.
- Simplify inventory persistence to match the new ownership-only public contract.
- Update CLI workflows so they no longer depend on inventory change-history fields for core behavior.

**Non-Goals:**

- Implement the future controller or release CRDs.
- Preserve the future controller and status work for later changes.
- Finalize the entire future status/history API for controller-managed releases.
- Add rollback/remediation storage in this change.

## Decisions

### 1. Inventory becomes ownership-only

**Decision:** The public inventory contract consists of an `Inventory` object and `InventoryEntry` records describing the current set of owned resources. Optional summary fields (`revision`, `digest`, `count`) may be included.

**Why:** This is the smallest shared contract that both the CLI and future controller need for prune safety, resource scoping, and discovery.

**Included in the ownership contract:**

- `entries[]`
- per-entry `group`, `kind`, `namespace`, `name`
- optional per-entry `version`
- optional per-entry `component`
- optional top-level `revision`, `digest`, `count`

**Explicitly excluded from the ownership contract:**

- raw values
- source path/version metadata
- per-change timestamps
- remediation counters
- history index and change map

### 2. Release/source/history state moves out of inventory

**Decision:** Public inventory no longer carries change history or source metadata. The CLI must be prepared to read source/version/history from release status later, not from `pkg/inventory` itself.

**Why:** Inventory answers "what do I own?". It should not also answer "what happened?" and "what source produced this?".

**Rationale:** This split aligns the CLI with the future controller design where:

- `status.inventory` answers ownership
- `status.lastAttempted*` / `status.lastApplied*` answer reconcile state
- `status.history` answers what happened recently

### 3. Storage stops defining the public contract

**Decision:** Secret naming, labels, and codec behavior are implementation details of inventory persistence, not the shape of the core inventory contract.

**Why:** The future controller may embed inventory directly in CR status. The public package should model ownership, not a specific storage representation.

**Alternatives considered:**

- **Keep Secret codec in `pkg/inventory` indefinitely**: rejected because it keeps the public package centered on one storage mechanism instead of the reusable contract.
- **Preserve the old history-bearing storage shape in parallel**: rejected because the CLI is still under heavy development and can adopt the cleaner ownership-only shape now.

### 4. CLI workflows continue to use inventory for prune and scoped discovery

**Decision:** CLI apply/delete/status/list workflows continue to use ownership inventory, but must stop depending on history-bearing inventory fields for metadata extraction and decision-making.

**Why:** The CLI still needs a persisted ownership set for stale detection and no-source discovery. That part of the current design remains correct.

**Practical consequence:**

- `apply` uses ownership inventory to compute stale resources
- `delete` and `status` use ownership inventory to enumerate tracked resources
- display metadata that previously came from inventory history must move to release-specific state or be dropped until release status exists

### 5. Inventory v2 should be embeddable in CR status unchanged

**Decision:** The inventory shape defined by this change should be valid both as a public Go type and as a future CR `status.inventory` payload.

**Why:** This is the main preparation step for controller work. We want one ownership model, not one for Secrets and another for CR status.

## Example APIs

### Public Go types

The public package should look more like this:

```go
package inventory

type Inventory struct {
	Revision int              `json:"revision,omitempty"`
	Digest   string           `json:"digest,omitempty"`
	Count    int              `json:"count,omitempty"`
	Entries  []InventoryEntry `json:"entries"`
}

type InventoryEntry struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"v,omitempty"`
	Component string `json:"component,omitempty"`
}
```

Pure helpers should stay focused on ownership semantics:

```go
package inventory

func NewEntryFromResource(r *unstructured.Unstructured) InventoryEntry
func IdentityEqual(a, b InventoryEntry) bool
func K8sIdentityEqual(a, b InventoryEntry) bool
func ComputeStaleSet(previous, current []InventoryEntry) []InventoryEntry
func ComputeDigest(entries []InventoryEntry) string
```

The old history-bearing API is what this change is explicitly moving away from:

```go
// removed from the public inventory contract
type ChangeEntry struct{ ... }
type ChangeSource struct{ ... }

func ComputeChangeID(...)
func UpdateIndex(...)
func PruneHistory(...)
func PrepareChange(...)
```

### Example persisted inventory document

If inventory persistence still uses a Secret for the CLI, the stored payload should reflect the same ownership-only shape directly:

```json
{
  "revision": 7,
  "digest": "sha256:aa22d4a6d8d0c7a6a4e8a6c9b52d0d3b7c1b5c56d1e1f9b622f0d7288f2e6abc",
  "count": 3,
  "entries": [
    {
      "group": "apps",
      "kind": "Deployment",
      "namespace": "apps",
      "name": "jellyfin",
      "v": "v1",
      "component": "server"
    },
    {
      "group": "",
      "kind": "Service",
      "namespace": "apps",
      "name": "jellyfin",
      "v": "v1",
      "component": "server"
    },
    {
      "group": "networking.k8s.io",
      "kind": "Ingress",
      "namespace": "apps",
      "name": "jellyfin",
      "v": "v1",
      "component": "server"
    }
  ]
}
```

### Example future CR status embedding

The same shape should embed directly under a future release status:

```yaml
status:
  inventory:
    revision: 7
    digest: sha256:aa22d4a6d8d0c7a6a4e8a6c9b52d0d3b7c1b5c56d1e1f9b622f0d7288f2e6abc
    count: 3
    entries:
      - group: apps
        kind: Deployment
        namespace: apps
        name: jellyfin
        v: v1
        component: server
      - group: ""
        kind: Service
        namespace: apps
        name: jellyfin
        v: v1
        component: server
```

## Future Controller Notes

This inventory redesign is intentionally shaped so the future controller can reuse it unchanged in its CRDs.

### ModuleRelease / BundleRelease status usage

The future controller should use the shared inventory shape under release status, for example:

- `ModuleRelease.status.inventory`
- `BundleRelease.status.inventory`

That inventory should mean exactly one thing in controller-managed releases: the current set of Kubernetes resources owned by the release.

It should be used by the controller for:

- prune decisions (`previous owned set - current rendered set`)
- tracked-resource discovery for status and debugging
- limiting drift checks and targeted live lookups to owned resources
- reporting resource ownership back to the CLI without re-rendering

It should not be used by the controller as the place to store:

- release source information
- resolved config values
- reconcile/remediation counters
- bounded action history
- the full rendered output

Those concerns belong in other status fields, for example:

- `status.source`
- `status.lastAttemptedSourceDigest`
- `status.lastAppliedConfigDigest`
- `status.lastAppliedRenderDigest`
- `status.history`
- `status.conditions`

### Why this split fits the controller

The controller can rely on deterministic CUE rendering:

- source artifact digest + config digest + render pipeline inputs determine desired output
- inventory tracks only what is currently owned in the cluster
- reconcile status tracks what source/config/render revision was attempted or applied

That gives the controller a cleaner model than the current history-bearing inventory Secret:

- `spec` says what is desired
- `status.inventory` says what is owned
- `status.history` and digest fields say what happened

### Example future ModuleRelease status shape

```yaml
status:
  observedGeneration: 3
  source:
    ref:
      apiVersion: source.toolkit.fluxcd.io/v1
      kind: OCIRepository
      name: jellyfin-module
      namespace: apps
    artifactRevision: 1.2.3@sha256:4b2c...
    artifactDigest: sha256:4b2c...
  lastAttemptedSourceDigest: sha256:4b2c...
  lastAttemptedConfigDigest: sha256:18a1...
  lastAttemptedRenderDigest: sha256:b91e...
  lastAppliedSourceDigest: sha256:4b2c...
  lastAppliedConfigDigest: sha256:18a1...
  lastAppliedRenderDigest: sha256:b91e...
  inventory:
    revision: 7
    digest: sha256:aa22...
    count: 3
    entries:
      - group: apps
        kind: Deployment
        namespace: apps
        name: jellyfin
        v: v1
        component: server
  history:
    - sequence: 7
      action: apply
      phase: Succeeded
      sourceDigest: sha256:4b2c...
      configDigest: sha256:18a1...
      renderDigest: sha256:b91e...
  conditions:
    - type: Ready
      status: "True"
      reason: ReconciliationSucceeded
```

The key point is that the controller should embed the same ownership inventory shape directly, while keeping controller-specific reconcile state outside the inventory contract.

## Risks / Trade-offs

- **[CLI output regressions]** -> Some list/status metadata currently inferred from inventory history may disappear unless explicitly replaced. Mitigation: update those commands deliberately as part of this change and constrain inventory expectations in specs.
- **[Public API churn]** -> Consumers of the newly-public package may need updates. Mitigation: do this now, before external controller code is built on the older shape.
- **[Status metadata gap]** -> If the CLI stops reading version/history from inventory before replacement fields exist, output can regress. Mitigation: capture the dependency in modified `mod-list` and `mod-status` specs now.
- **[Hidden storage coupling]** -> If storage helpers remain mixed into the core package, the change only renames the problem. Mitigation: separate ownership types/helpers from persistence responsibilities.

## Migration Plan

1. Introduce ownership-only inventory types in `pkg/inventory`.
2. Redesign inventory persistence to store the ownership-only shape directly.
3. Update CLI apply/delete/status/list workflows to consume ownership inventory rather than history-bearing types.
4. Remove change-history assumptions from the CLI code paths touched by inventory.
5. Add tests covering ownership-only inventory semantics and persistence behavior.

## Open Questions

- Should release metadata (`release name`, `release ID`, `module name`) remain colocated with persisted inventory, or move into a separate public release summary package in a follow-up change?
- Should inventory `revision` be controller/CLI managed everywhere, or omitted until release status exists?
- Should `component` remain part of inventory identity long term, or become purely informational once status/history is richer?
