# RFC-0001: Release Inventory

| Field       | Value                              |
|-------------|------------------------------------|
| **Status**  | Draft                              |
| **Created** | 2026-02-11                         |
| **Authors** | OPM Contributors                   |

## Summary

Introduce a lightweight release inventory stored as a Kubernetes Secret that tracks which resources belong to a ModuleRelease. This enables automatic pruning of stale resources during `opm mod apply` and provides a precise source of truth for `diff`, `delete`, and `status` commands. The Secret also maintains a history of changes, enabling future rollback capabilities.

## Motivation

### Current State: OPM is Stateless

Today, OPM uses **label-based discovery only**. When resources are applied, OPM labels are injected, but no record of the applied set is stored anywhere. All subsequent operations (`delete`, `status`, `diff`) rediscover resources by querying the Kubernetes API with label selectors.

```text
┌───────────────────────────────────────────────────────┐
│                 OPM Today                             │
│                                                       │
│   opm mod apply                                       │
│       │                                               │
│       ▼                                               │
│   ┌──────────┐     Server-Side      ┌──────────────┐  │
│   │ Rendered │────  Apply  ───────▶│  K8s API     │  │
│   │ manifests│     (+ labels)       │  (resources) │  │
│   └──────────┘                      └──────────────┘  │
│                                          │            │
│   opm mod status / delete / diff         │            │
│       │                                  │            │
│       └──── label selector query ────────┘            │
│                                                       │
│   NO STATE STORED. Labels are the only record.        │
└───────────────────────────────────────────────────────┘
```

This works for simple cases but has known gaps:

1. **No orphan cleanup**: If a value change causes a resource to be renamed, the
   old resource becomes an orphan. `apply` does not clean it up.
2. **No stored values**: Unlike Helm, OPM does not record what values were used
   for a deployment. You cannot reconstruct "what was applied" without the    original module source and values.
3. **Noisy diff**: `opm mod diff` scans ALL API types with label selectors,
   which is slow and can produce false positives from label overlap.
4. **No automatic pruning**: Orphaned resources are detected by `diff` but not
   cleaned up by `apply`.

### The Rename Problem

The core motivating scenario. Consider an application "Jellyfin":

```text
┌──────────────────────────────────────────────────────────────────────┐
│                    THE RENAME SCENARIO                               │
│                                                                      │
│  Apply v1:                        Apply v2 (value change):           │
│                                                                      │
│  values:                          values:                            │
│    name: "minecraft"                 name: "minecraft-server"        │
│                                                                      │
│  Produces:                        Produces:                          │
│  ┌───────────────────────┐         ┌──────────────────────────────┐  │
│  │ StatefulSet/minecraft │         │ StatefulSet/minecraft-server │  │
│  │ Service/minecraft     │         │ Service/minecraft-server     │  │
│  │ PVC/config            │         │ PVC/config                   │  │
│  └───────────────────────┘         └──────────────────────────────┘  │
│                                                                      │
│  What SHOULD happen:                                                 │
│  1. Create StatefulSet/minecraft-server                              │
│  2. Create Service/minecraft-server                                  │
│  3. DELETE StatefulSet/minecraft    ← orphaned from v1               │
│  4. DELETE Service/minecraft        ← orphaned from v1               │
│                                                                      │
│  What happens WITHOUT inventory:                                     │
│  1. Create StatefulSet/minecraft-server  OK                          │
│  2. Create Service/minecraft-server      OK                          │
│  3. StatefulSet/minecraft still exists  FAIL ORPHAN                  │
│  4. Service/minecraft still exists      FAIL ORPHAN                  │
│                                                                      │
│  Result: TWO instances of Jellyfin running.                          │
│  Nobody told the old one to go away.                                 │
└──────────────────────────────────────────────────────────────────────┘
```

Tools like bare `kubectl apply` and `kustomize` cannot handle this. The resource is lost and will not be removed or recreated. A release inventory solves this by tracking the previous set and computing the diff.

## Prior Art

### Industry Approaches

Research was conducted across all major Kubernetes deployment tools to understand the landscape of release state storage:

#### Helm (Secrets — heavy)

Helm stores the **entire** release state in a Secret per revision:

```text
Secret: sh.helm.release.v1.<name>.v<revision>
  type: helm.sh/release.v1
  data:
    release: <base64-gzipped blob containing:>
      - chart metadata
      - user-supplied values
      - computed values
      - rendered manifests (the full YAML!)
      - hooks
      - status, timestamps, version
```

Multiple revisions are kept for rollback (default: 10). This enables full rollback from stored manifests but can hit the 1MB etcd size limit with large charts.

#### Timoni (Secrets — lightweight, single typed object)

Timoni stores a **lightweight** inventory in a Secret. Unlike Helm, it does NOT store full rendered manifests. All state is wrapped in a single `data.instance` field containing a typed JSON blob:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: timoni.<instance-name>
  namespace: <instance-namespace>
  labels:
    app.kubernetes.io/component: instance
    app.kubernetes.io/created-by: timoni
    app.kubernetes.io/name: <name>
type: timoni.sh/instance
data:
  instance: <single JSON blob>
```

The JSON blob is a typed object with `kind`/`apiVersion`, structured like a Kubernetes resource:

```json
{
  "kind": "Instance",
  "apiVersion": "timoni.sh/v1alpha1",
  "metadata": {
    "name": "cert-manager",
    "namespace": "cert-manager",
    "labels": { "bundle.timoni.sh/name": "core" }
  },
  "module": {
    "name": "timoni.sh/flux-helm-release",
    "repository": "oci://ghcr.io/stefanprodan/modules/flux-helm-release",
    "version": "2.7.3-1",
    "digest": "sha256:39b1153f..."
  },
  "values": "{\n\trepository: {\n\t\turl: \"https://charts.jetstack.io\"\n\t}\n}",
  "lastTransitionTime": "2026-01-19T08:34:38Z",
  "inventory": {
    "entries": [
      { "id": "cert-manager_cert-manager_helm.toolkit.fluxcd.io_HelmRelease", "v": "v2" },
      { "id": "cert-manager_cert-manager_source.toolkit.fluxcd.io_HelmRepository", "v": "v1" }
    ]
  }
}
```

Key characteristics:

- **Single blob**: All state in one `data.instance` field, not multiple data keys.
- **Typed with `kind`/`apiVersion`**: Enables schema versioning and future CRD migration.
- **Compact inventory IDs**: `<namespace>_<name>_<group>_<Kind>` with version as
  a separate `"v"` field — version is NOT part of the identity.
- **Values as CUE string**: Stored in native CUE format, not converted to JSON.
- **Module digest**: OCI content-addressable digest for reproducibility.
- **lastTransitionTime**: Timestamp of the last apply operation.
- **No rendered manifests**: Since module source + values are known, manifests
  can always be re-rendered. This keeps the Secret small (~2-5KB per instance   vs ~100KB+ per revision for Helm).

#### Carvel kapp (ConfigMaps)

kapp uses ConfigMaps for both inventory and change history:

```text
ConfigMap: {app-name}-ctrl           # Current inventory
ConfigMap: {app-name}-ctrl-change-*  # Change history (up to 200)
```

Resources are tracked via labels (`kapp.k14s.io/app`, `kapp.k14s.io/association`).

#### Flux Kustomize Controller (CRD status)

Flux records inventory in `.status.inventory` of the `Kustomization` CRD:

```yaml
status:
  inventory:
    entries:
      - id: default_my-deploy_apps_Deployment_v1
```

#### ArgoCD (CRD + annotations/labels)

ArgoCD uses an `Application` CRD as the source of truth. Resources are tracked via configurable methods: labels, annotations, or both. The annotation format encodes `name:namespace:group/kind`.

#### Crossplane (CRD + ownerReferences)

Crossplane uses Composite resources with `ownerReferences` on all composed resources, enabling native Kubernetes garbage collection.

#### Server-Side Apply managedFields (Kubernetes native)

SSA with a named field manager tracks which **fields** are owned, but not which
**resources** belong to a release. Solves field ownership, not inventory.

#### External Storage (Helm SQL driver, Pulumi)

Helm supports a SQL driver. Pulumi stores state in cloud backends. These have no cluster size limits but introduce external dependencies.

### Comparison Matrix

```text
┌──────────────┬────────┬────────┬────────┬─────────┬────────┬───────────────┐
│              │Secret  │ CM     │ CRD    │ Annot.  │ SSA    │ External DB   │
│              │        │        │        │ /Labels │ mgdFld │               │
├──────────────┼────────┼────────┼────────┼─────────┼────────┼───────────────┤
│ No CRDs req. │  [x]   │  [x]   │  [ ]   │  [x]    │  [x]   │  [x]          │
│ Native GC    │  [ ]   │  [ ]   │  [x]*  │  [ ]    │  [ ]   │  [ ]          │
│ Inventory    │  [x]   │  [x]   │  [x]   │  [ ]    │  [ ]   │  [x]          │
│ Values store │  [x]   │  [x]   │  [x]   │  [ ]    │  [ ]   │  [x]          │
│ Rollback     │  [x]** │  [x]** │  [x]   │  [ ]    │  [ ]   │  [x]          │
│ Field owner. │  [ ]   │  [ ]   │  [ ]   │  [ ]    │  [x]   │  [ ]          │
│ Size limits  │ 1MB    │ 1MB    │ 1MB    │ 256KB   │  N/A   │  ∞            │
│ Cluster-free │  [ ]   │  [ ]   │  [ ]   │  [ ]    │  [ ]   │  [x]          │
│ CLI-only     │  [x]   │  [x]   │  [x]   │  [x]    │  [x]   │  [x]          │
│ Controller   │  [ ]   │  [ ]   │  [x]   │  [ ]    │  [x]   │  [ ]          │
│ Complexity   │  Low   │ Low    │ Med    │ Lowest  │ Lowest │  High         │
├──────────────┴────────┴────────┴────────┴─────────┴────────┴───────────────┤
│ * CRD with ownerReferences    ** If multiple revisions stored              │
└────────────────────────────────────────────────────────────────────────────┘
```

### Why Timoni-Style

OPM and Timoni are sibling projects in spirit — both CUE-based, both CLI-driven, both module-oriented. Timoni's lightweight inventory approach is the best fit because:

1. **You already have the module source + values** — you can re-render anytime,
   so storing full manifests (like Helm) is wasteful.
2. **Object IDs are sufficient** for the core use case: knowing what to prune.
3. **No CRDs required** — works on any cluster, no chicken-and-egg bootstrap.
4. **Small footprint** — ~2-5KB vs 100KB+ per Helm revision.
5. **Closest architectural analog** to OPM's existing patterns.

```text
┌─────────────────────────────────────────────────────┐
│           Helm vs Timoni Secret Storage             │
│                                                     │
│  Helm:     ████████████████████████  (heavy)        │
│            Full manifests + values + metadata       │
│            ~100KB+ per revision × N revisions       │
│                                                     │
│  Timoni:   ████                      (light)        │
│            Object IDs + values + module ref         │
│            ~2-5KB per instance                      │
│                                                     │
│  OPM:      ████                      (light)        │
│            Follow Timoni's approach                 │
│            + change history in single Secret        │
└─────────────────────────────────────────────────────┘
```

### Learnings from Timoni

After studying Timoni's actual implementation (not just its documentation), we identified specific design patterns worth adopting:

1. **Single typed object, not multiple data fields.** Timoni wraps all state in
   one JSON blob with `kind`/`apiVersion`. This allows the entire format to    evolve — bump `apiVersion` and handle old + new formats during migration.    With separate `data` fields, versioning each independently becomes messy.    The typed blob also maps directly to a future CRD: the JSON blob IS the CRD    spec, just stored in a Secret for now.

2. **Version separated from identity.** Timoni's inventory entries use a compact
   ID for identity and a separate `"v"` field for API version. This prevents    false orphans during Kubernetes API version migrations (e.g., Ingress moving    from `v1beta1` to `v1`). If version were part of the identity, an API upgrade    would make the old entry look stale and trigger a spurious delete.

3. **Values in native format.** Timoni stores CUE values as a CUE-formatted
   string, not converted to JSON. This preserves the source language and keeps    values human-readable when inspecting the Secret.

4. **Module digest for reproducibility.** Timoni records the OCI digest
   alongside version. Version tags are mutable (someone can push to the same    tag), but digests are immutable — they prove exactly which module bits were    applied. OPM defers this: CUE modules are resolved by the CUE SDK, not a    custom OCI artifact, so module digests are not directly accessible. OPM uses    a `manifestDigest` (hash of rendered output) instead to achieve comparable    change detection.

5. **lastTransitionTime.** A simple timestamp of when the last apply happened.
   Cheap to add, useful for debugging and status reporting.

6. **Distinguishing label.** Timoni labels its inventory Secret with
   `app.kubernetes.io/component: instance` to distinguish it from application    resources. This prevents the inventory Secret from appearing alongside    workload resources in label-based queries.

OPM adopts these patterns, adapted to its own domain model and naming conventions. Key divergences: OPM uses CUE module paths instead of OCI references for module identification (since OPM uses CUE modules directly, not custom OCI artifacts), and explicit JSON fields instead of Timoni's compact underscore-separated string IDs for inventory entries (to avoid parsing ambiguity when names contain underscores).

## Design

### Secret Structure Overview

A single Kubernetes Secret per release contains all state: release metadata, a change history index, and individual change entries. This keeps all release state co-located and atomically updatable.

```text
┌─────────────────────────────────────────────────────────────────┐
│  Secret: opm.<release-name>.<release-id-uuid>                   │
│  type: opmodel.dev/release                                      │
│                                                                 │
│  data:                                                          │
│    metadata:          Release-level metadata (typed JSON blob)  │
│    index:             Ordered list of change IDs (newest first) │
│    change-sha1-<id>:  Per-change state (inventory, values, etc) │
│    change-sha1-<id>:  ...                                       │
│    change-sha1-<id>:  ...                                       │
└─────────────────────────────────────────────────────────────────┘
```

### Metadata Field

The `data.metadata` field contains release-level information as a typed JSON object with `kind`/`apiVersion`, following Timoni's pattern. This enables schema versioning and maps directly to a future CRD.

```json
{
  "kind": "ModuleRelease",
  "apiVersion": "core.opmodel.dev/v1alpha1",
  "name": "minecraft",
  "namespace": "default",
  "releaseId": "a3b8f2e1-7c4d-5a9e-b6f0-1234567890ab",
  "lastTransitionTime": "2026-02-11T14:30:00Z"
}
```

| Field                 | Content                                         | Purpose                            |
|-----------------------|-------------------------------------------------|------------------------------------|
| `kind`                | `"ModuleRelease"`                                     | Schema identification              |
| `apiVersion`          | `"core.opmodel.dev/v1alpha1"`                        | Schema versioning, CRD migration   |
| `name`                | Release name                                    | Human identification               |
| `namespace`           | Release namespace                               | Scoping                            |
| `releaseId`           | Deterministic UUIDv5                            | Unique release identity            |
| `lastTransitionTime`  | RFC 3339 timestamp of last apply                | Debugging, status reporting        |

### Index Field

The `data.index` field is a JSON array of change IDs, ordered newest first:

```json
["change-sha1-7f2c9d01", "change-sha1-a3b8f2e1"]
```

The first entry is always the current (latest) change. The CLI is responsible for maintaining this ordering.

### Change Entries

Each `data.change-sha1-<id>` field contains the full state for a single change:

```json
{
  "module": {
    "path": "opmodel.dev/modules/minecraft@v0",
    "version": "0.1.0",
    "name": "minecraft"
  },
  "values": "{\n\tname: \"minecraft\"\n\tdataPath: \"/mnt/server\"\n}",
  "manifestDigest": "sha256:e5b7a3f...",
  "timestamp": "2026-02-11T14:30:00Z",
  "inventory": {
    "entries": [
      { "group": "apps", "kind": "StatefulSet", "namespace": "default", "name": "minecraft", "v": "v1", "component": "app" },
      { "group": "", "kind": "Service", "namespace": "default", "name": "minecraft", "v": "v1", "component": "app" },
      { "group": "", "kind": "PersistentVolumeClaim", "namespace": "default", "name": "config", "v": "v1", "component": "app" }
    ]
  }
}
```

| Field                         | Content                                          | Purpose                            |
|-------------------------------|--------------------------------------------------|------------------------------------|
| `module.path`                 | CUE module path (e.g., `opmodel.dev/m@v0`)       | Module identity, re-render         |
| `module.version`              | Module version string (semver)                   | Audit trail                        |
| `module.name`                 | Module declared name                             | Human identification               |
| `values`                      | Resolved andunified CUE values as CUE string     | Audit trail, future rollback       |
| `manifestDigest`              | SHA256 of deterministically serialized manifests | Change detection, change ID input  |
| `timestamp`                   | RFC 3339 timestamp of this change                | Audit trail                        |
| `inventory.entries[]`         | Array of resource identity objects               | Pruning, diff, delete, status      |

**Values**: Stored as a CUE-formatted string (native format), not converted to
JSON. File paths are not stored — they are meaningless on a different machine or CI runner. The resolved values are the actual input that produced the rendered resources.

**Module path**: The CUE module path from `cue.mod/module.cue` (e.g.,
`opmodel.dev/modules/minecraft@v0`). This is the canonical module identity in the CUE ecosystem — the CUE SDK maps it to an OCI registry lookup automatically via the `CUE_REGISTRY` environment variable. OPM uses CUE modules directly rather than publishing custom OCI artifacts, so this path (not an OCI reference) is the correct identifier. The path is always present since CUE requires a `module:` declaration in every module.

**Module version**: For published modules, this is the semver version from
`metadata.version`. For local development modules that have not been versioned, the `version` field is replaced with `"local": true` to indicate the module was applied from a local filesystem path without a published version.

**Manifest digest**: A SHA256 hash of the deterministically serialized rendered
manifests. This captures any change to the module output — including template changes in local modules where `module.version` may not change between edits. This field is always present regardless of whether the module is published or local.

**Inventory entry identity**: Each entry has fields `group`, `kind`,
`namespace`, `name`, `component` (the identity) and `v` (the API version, stored separately). Set operations for pruning use the identity fields. The `v` field is used when fetching or deleting the resource from the cluster. Separating version from identity prevents false orphans when Kubernetes API versions change (e.g., Ingress migrating from `networking.k8s.io/v1beta1` to `networking.k8s.io/v1`).

The `component` field records which module component produced the resource (e.g., `"app"`, `"cache"`, `"worker"`). This enables `opm mod status` to group resources by component when displaying release health, using only the inventory — no need to read labels back from the cluster. Including component in identity means the inventory can precisely track which component owns which resource. However, because Kubernetes itself identifies resources by GVK + namespace + name (without component), a **component rename safety check** is required during pruning to prevent a component rename from triggering a spurious delete (see Apply Flow, Step 5b).

**What gets tracked**: The inventory contains **only resources that OPM directly
renders** — the output of the build pipeline. Derived resources that Kubernetes automatically creates (Endpoints for Services, ReplicaSets for Deployments, Pods for StatefulSets/Deployments, etc.) are NOT tracked. When OPM deletes a release, it deletes only the parent resources in the inventory. Kubernetes garbage collection handles cleanup of derived child resources automatically. This keeps the inventory precise, avoids unnecessary API calls, and respects Kubernetes ownership semantics — OPM owns what it renders, not what controllers create in response.

### Change ID

Each change is identified by a deterministic SHA1 hash, truncated to 8 hex characters. The data key format is `change-sha1-<8chars>`.

**Hash inputs:**

```text
change ID = SHA1(
  module.path +
  module.version +
  values (resolved CUE string) +
  manifestDigest (SHA256 of rendered manifests)
)
```

The `manifestDigest` is computed first as a SHA256 of the deterministically serialized rendered manifests, then included as an input to the change ID hash. This means the change ID captures all four dimensions of what defines a change:

```text
┌─────────────────────────────────────────────────────────────────┐
│  CHANGE IDENTITY — FOUR DIMENSIONS                              │
│                                                                 │
│  1. module.path        → Which module (CUE module identity)     │
│  2. module.version     → Which version of that module           │
│  3. values             → What configuration the user provided   │
│  4. manifestDigest     → What output was actually produced      │
│                                                                 │
│  Same inputs → same change ID (idempotent)                      │
│  Any dimension changes → different change ID                    │
└─────────────────────────────────────────────────────────────────┘
```

**Why all four inputs?**

The `manifestDigest` alone would be sufficient for local modules, but including `path` and `version` ensures that module upgrades are always recorded as distinct changes — even if the rendered output happens to be identical. Including `values` ensures that a value change that produces the same output (e.g., setting a default explicitly) is also recorded.

**Deterministic manifest serialization:** The `manifestDigest` requires
that identical resources always produce identical bytes when serialized. This is achievable with Go's standard library (see [Deterministic Manifest Serialization](#deterministic-manifest-serialization) for the full algorithm and analysis of existing codebase support).

**Idempotent re-applies:**

With a deterministic hash, reapplying the exact same configuration produces the same change ID. This means the existing change entry is overwritten (with an updated timestamp) rather than creating a duplicate entry:

```text
Apply #1: module=1.0.0, values=X, output=Y → change-sha1-a3b8f2e1 (created)
Apply #2: module=1.0.0, values=X, output=Y → change-sha1-a3b8f2e1 (overwritten)
Apply #3: module=1.1.0, values=X, output=Z → change-sha1-7f2c9d01 (new entry)
Apply #4: module=1.1.0, values=X, output=Z → change-sha1-7f2c9d01 (overwritten)

Index after: ["change-sha1-7f2c9d01", "change-sha1-a3b8f2e1"]
```

History only grows when the inputs **actually change**. Idempotent re-applies update the existing entry's timestamp and inventory, then bump it to the front of the index. This keeps the change history meaningful — it tracks state transitions, not CLI invocations.

### Deterministic Manifest Serialization

The `manifestDigest` is a SHA256 hash of the rendered manifests, serialized deterministically so that identical resource content always produces the same digest. This section documents the algorithm and the codebase properties that make it reliable.

#### Algorithm

```text
┌─────────────────────────────────────────────────────────────────┐
│  DETERMINISTIC MANIFEST DIGEST                                  │
│                                                                 │
│  Input: []*build.Resource (rendered resources from pipeline)    │
│                                                                 │
│  Step 1: Sort resources with total ordering                     │
│    Primary:    GVK weight (ascending)                           │
│    Secondary:  API group (alphabetical)                         │
│    Tertiary:   Kind (alphabetical)                              │
│    Quaternary: Namespace (alphabetical)                         │
│    Quinary:    Name (alphabetical)                              │
│                                                                 │
│    No two resources can share GVK + namespace + name in a       │
│    valid deployment, so this guarantees a unique position.      │
│                                                                 │
│  Step 2: Serialize each resource independently                  │
│    json.Marshal(resource.Object)                                │
│    → Go's encoding/json sorts map keys alphabetically           │
│    → Each individual serialization is deterministic             │
│                                                                 │
│  Step 3: Concatenate serialized bytes                           │
│    Append each resource's JSON bytes in sorted order            │
│    Use a newline separator between resources                    │
│                                                                 │
│  Step 4: Hash                                                   │
│    SHA256(concatenated bytes)                                   │
│    → manifestDigest = "sha256:<hex>"                            │
│                                                                 │
│  Properties:                                                    │
│    [x] Same resources in any input order → same digest          │
│    [x] Any content change → different digest                    │
│    [x] Any resource added/removed → different digest            │
│    [x] No external dependencies (Go stdlib only)                │
└─────────────────────────────────────────────────────────────────┘
```

#### Why This Works: Codebase Analysis

Investigation of the OPM codebase and Go standard library confirms that deterministic serialization is achievable with minimal changes.

**JSON key ordering is guaranteed.** Go's `encoding/json.Marshal` sorts map keys
alphabetically for `map[string]interface{}`. This is documented Go behavior (since Go 1). Since `unstructured.Unstructured` is backed by `map[string]interface{}`, and its `MarshalJSON()` method delegates to `json.Encode(t.Object)`, serializing any individual resource with `json.Marshal` already produces deterministic output. No custom serializer is needed.

**CUE evaluation is deterministic.** CUE structs have deterministic field
ordering. When decoded into Go maps via `cue.Value.Decode()`, the CUE field ordering is lost (Go maps do not preserve insertion order), but this does not matter because `json.Marshal` sorts keys independently.

**Resource normalization already sorts.** The `normalizeK8sResource()` functions
in `internal/build/executor.go` already sort map keys alphabetically when converting OPM-style maps to Kubernetes arrays (`mapToPortsArray`, `mapToEnvArray`, `mapToVolumeMountsArray`, `mapToVolumesArray`).

#### What Requires Attention

**Resource list ordering needs a total ordering.** The current pipeline sort in
`internal/build/pipeline.go:153-157` uses `sort.Slice` with weight-only comparison. Resources with the same weight (e.g., two Services) can appear in either order depending on Go map iteration in the executor. For the digest to be deterministic, the sort must have tiebreakers that produce a total ordering.

The existing `sortResourceInfos` in `internal/output/manifest.go:58-77` already implements the correct pattern — sorting by weight, then namespace, then name. The digest sort extends this with API group and kind as additional tiebreakers to handle resources of different types that share the same weight:

```text
Current (pipeline.go — insufficient):
  sort by: weight
  → Two Services in arbitrary order. Non-deterministic.

Required (for digest):
  sort by: weight → group → kind → namespace → name
  → Total ordering. Every resource has a unique position.

Existing model (manifest.go — close):
  sort by: weight → namespace → name
  → Good but doesn't distinguish different GVKs at same weight.
```

**Executor job ordering is non-deterministic.** The `Executor.ExecuteWithTransformers()`
in `internal/build/executor.go:58-72` iterates `match.ByTransformer`, which is a Go `map[string][]*LoadedComponent`. Go map iteration is non-deterministic, so the order resources are produced varies between runs. This does NOT affect the digest because the digest sort (Step 1) re-orders resources after execution. The executor ordering only affects the input to the sort, not the output.

**Server-generated fields are not present.** Since the digest is computed from
rendered output (before apply, not after), server-generated fields like `metadata.resourceVersion`, `metadata.uid`, `metadata.creationTimestamp`, `metadata.managedFields`, and `status` are not present in the serialized data. If a CUE template were to explicitly set any of these fields, they would be included in the digest — this is correct behavior (the template output changed).

#### Fields Included and Excluded

The digest is computed from the rendered resource as-is. Since rendering happens before apply, the resource contains only user-defined fields:

```text
Included (present in rendered output):
  [x] apiVersion, kind                (identity)
  [x] metadata.name, namespace        (identity)
  [x] metadata.labels                 (module-defined labels only)
  [x] metadata.annotations            (user-defined)
  [x] spec                            (user-defined)
  [x] data, stringData                (for ConfigMaps/Secrets)

Not present (server-generated, only exist after apply):
  [ ] metadata.resourceVersion
  [ ] metadata.uid
  [ ] metadata.creationTimestamp
  [ ] metadata.generation
  [ ] metadata.managedFields
  [ ] status
```

Note: OPM labels are injected by CUE transformers during the render pipeline, which is BEFORE the digest is computed. The digest therefore includes OPM labels as part of the rendered output. This is the correct behavior — labels are part of the module's declared intent, not a post-processing concern.

### Change History Pruning

The Secret accumulates change entries up to a configurable maximum:

- **Default**: Keep last 10 changes (same as Helm's default).
- **Configurable**: `--max-history=N` flag on `opm mod apply`.

On each successful apply:

1. Write/overwrite the change entry for the current change ID.
2. Prepend the change ID to the index (or move to front if already present).
3. If `len(index) > max_history`: remove oldest entries from both the index and
   the corresponding `data.change-*` keys.

**Size estimation**: 10 changes × ~2-5KB each = ~20-50KB. Well within etcd's
1MB Secret size limit.

### Secret Update Semantics

The inventory Secret is always updated with a **full PUT** (replace the entire Secret). Read the Secret, modify the in-memory representation (add/update change entry, update index, update metadata, prune old changes), then write the whole thing back.

This is atomic, simple, and safe. Since OPM is a CLI tool (not a controller), concurrent writers to the same release are not a realistic concern. If two humans run `opm mod apply` against the same release simultaneously, they already have bigger problems than Secret contention.

### Naming Convention

The inventory Secret name follows the pattern:

```text
opm.<release-name>.<release-id-uuid>
```

Example:

```text
opm.minecraft.a3b8f2e1-7c4d-5a9e-b6f0-1234567890ab
```

**Rationale:**

- The release name is there for humans (`kubectl get secrets` is scannable).
- The release-id UUID ensures uniqueness (no collisions if two modules share a
  name in the same namespace).
- The `opm.` prefix identifies it as an OPM-managed resource.

**Kubernetes naming constraints (RFC 1123 DNS subdomain):**

```text
Total max:      253 characters
Label max:      63 characters (each dot-separated segment)
Allowed chars:  lowercase alphanumeric, '-', '.'
Must start/end: alphanumeric

Fixed overhead:  "opm." (4) + "." (1) + UUID (36) = 41 chars
Remaining:       253 - 41 = 212 chars for release name
Label check:     "opm" (3 ok), release-name (≤63 ok), UUID (36 ok)
```

Release names are already constrained to ≤63 characters (they are used as Kubernetes label values), so this fits cleanly.

### Inventory Lookup

The inventory Secret is found by **name convention** (direct GET) with a
**label-based fallback**:

1. **Primary**: Construct the Secret name from render metadata
   (`opm.<name>.<release-id>`) and perform a direct `GET`. This is fast (single    API call). The release-id is deterministic (UUIDv5 computed from module name +    namespace), so it can always be reconstructed from the render output.

2. **Fallback**: If the direct GET fails (e.g., naming convention changed), list
   Secrets with label `module-release.opmodel.dev/uuid=<release-id>`.

### Labels on the Inventory Secret

The inventory Secret carries OPM labels plus a distinguishing component label:

```yaml
labels:
  app.kubernetes.io/managed-by: open-platform-model
  module.opmodel.dev/name: <name>
  module.opmodel.dev/namespace: <namespace>
  module-release.opmodel.dev/uuid: <release-id>
  opmodel.dev/component: inventory
```

The `opmodel.dev/component: inventory` label distinguishes the inventory Secret from application resources. This prevents it from appearing in label-based workload queries while still being discoverable by OPM tooling.

This ensures:

- `opm mod delete` can discover the inventory Secret via labels as a fallback.
- The inventory Secret does NOT pollute application resource queries.
- Standard Kubernetes tooling can identify it as OPM-managed.
- The component label can be used to filter: `kubectl get secrets -l opmodel.dev/component=inventory`.

### Full Example Secret

The following is a complete example of an inventory Secret after three applies: an initial install, a value change (rename), and a module version upgrade.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: opm.minecraft.a3b8f2e1-7c4d-5a9e-b6f0-1234567890ab
  namespace: default
  labels:
    app.kubernetes.io/managed-by: open-platform-model
    module.opmodel.dev/name: minecraft
    module.opmodel.dev/namespace: default
    module-release.opmodel.dev/uuid: a3b8f2e1-7c4d-5a9e-b6f0-1234567890ab
    opmodel.dev/component: inventory
type: opmodel.dev/release
stringData:
  metadata: |
    {
      "kind": "ModuleRelease",
      "apiVersion": "core.opmodel.dev/v1alpha1",
      "name": "minecraft",
      "namespace": "default",
      "releaseId": "a3b8f2e1-7c4d-5a9e-b6f0-1234567890ab",
      "lastTransitionTime": "2026-02-11T16:45:00Z"
    }
  index: |
    ["change-sha1-e92f4c01", "change-sha1-7f2c9d01", "change-sha1-a3b8f2e1"]
  change-sha1-a3b8f2e1: |
    {
      "module": {
        "path": "opmodel.dev/modules/minecraft@v0",
        "version": "0.1.0",
        "name": "minecraft"
      },
      "values": "{\n\tname: \"minecraft\"\n\tdataPath: \"/mnt/server\"\n}",
      "manifestDigest": "sha256:b5d4a7e2f1c8936d0e5a2b7c4f8d1e3a6b9c2d5e8f1a4b7c0d3e6f9a2b5c8d1e",
      "timestamp": "2026-02-11T14:00:00Z",
      "inventory": {
        "entries": [
          { "group": "apps", "kind": "StatefulSet", "namespace": "default", "name": "minecraft", "v": "v1", "component": "app" },
          { "group": "", "kind": "Service", "namespace": "default", "name": "minecraft", "v": "v1", "component": "app" },
          { "group": "", "kind": "PersistentVolumeClaim", "namespace": "default", "name": "config", "v": "v1", "component": "app" }
        ]
      }
    }
  change-sha1-7f2c9d01: |
    {
      "module": {
        "path": "opmodel.dev/modules/minecraft@v0",
        "version": "0.1.0",
        "name": "minecraft"
      },
      "values": "{\n\tname: \"minecraft-server\"\n\tdataPath: \"/mnt/server\"\n}",
      "manifestDigest": "sha256:c8e2a1f4b7d5093e6a2c8f1d4b7e0a3c6d9f2b5e8a1c4d7f0b3e6a9c2d5f8b1a",
      "timestamp": "2026-02-11T15:30:00Z",
      "inventory": {
        "entries": [
          { "group": "apps", "kind": "StatefulSet", "namespace": "default", "name": "minecraft-server", "v": "v1", "component": "app" },
          { "group": "", "kind": "Service", "namespace": "default", "name": "minecraft-server", "v": "v1", "component": "app" },
          { "group": "", "kind": "PersistentVolumeClaim", "namespace": "default", "name": "config", "v": "v1", "component": "app" }
        ]
      }
    }
  change-sha1-e92f4c01: |
    {
      "module": {
        "path": "opmodel.dev/modules/minecraft@v0",
        "version": "0.2.0",
        "name": "minecraft"
      },
      "values": "{\n\tname: \"minecraft-server\"\n\tdataPath: \"/mnt/server\"\n}",
      "manifestDigest": "sha256:d9f3b2e5a8c1d4f7b0e3a6c9d2f5b8e1a4c7d0f3b6e9a2c5d8f1b4e7a0c3d6f9",
      "timestamp": "2026-02-11T16:45:00Z",
      "inventory": {
        "entries": [
          { "group": "apps", "kind": "StatefulSet", "namespace": "default", "name": "minecraft-server", "v": "v1", "component": "app" },
          { "group": "", "kind": "Service", "namespace": "default", "name": "minecraft-server", "v": "v1", "component": "app" },
          { "group": "", "kind": "PersistentVolumeClaim", "namespace": "default", "name": "config", "v": "v1", "component": "app" },
          { "group": "", "kind": "ConfigMap", "namespace": "default", "name": "minecraft-server-config", "v": "v1", "component": "app" }
        ]
      }
    }
```

In this example:

- **change-sha1-a3b8f2e1**: Initial install. Module v1.0.0 with name "minecraft".
  Three resources created.
- **change-sha1-7f2c9d01**: Value change. Same module version, but name changed
  to "minecraft-server". Resource names changed — the old StatefulSet/minecraft and   Service/minecraft were pruned automatically.
- **change-sha1-e92f4c01**: Module upgrade to v1.1.0. Same values, but the new
  version added a ConfigMap resource. The index shows this as the latest change.

## Apply Flow

```text
┌─────────────────────────────────────────────────────────────────────┐
│                    opm mod apply — WITH INVENTORY                   │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 1. RENDER                                                    │   │
│  │    Build pipeline → resources[] + metadata                   │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 2. COMPUTE MANIFEST DIGEST                                   │   │
│  │    manifestDigest = SHA256(sorted serialized manifests)      │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 3. COMPUTE CHANGE ID                                         │   │
│  │    changeID = SHA1(repo + version + values + manifestDigest) │   │
│  │    key = "change-sha1-<first 8 hex chars>"                   │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 4. READ PREVIOUS INVENTORY                                   │   │
│  │    GET Secret/opm.<name>.<release-id> in <namespace>         │   │
│  │    → Read index → latest change = index[0]                   │   │
│  │    → previous_inventory = latest change's inventory entries  │   │
│  │    → if not found: previous_inventory = ∅ (first install)   │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 5a. COMPUTE STALE SET                                        │   │
│  │    current_inventory = set of IDs from rendered resources    │   │
│  │    stale = previous_inventory - current_inventory            │   │
│  │    (identity = group + kind + namespace + name + component)  │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 5b. COMPONENT RENAME SAFETY CHECK                            │   │
│  │    For each entry in stale:                                  │   │
│  │      If current_inventory contains same group+kind+ns+name  │   │
│  │      (differing only in component):                          │   │
│  │        → Component rename, NOT an orphan                     │   │
│  │        → Remove from stale set                               │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 5c. PRE-APPLY EXISTENCE CHECK (first-time apply only)        │   │
│  │    If no previous inventory exists:                          │   │
│  │      For each resource in current_inventory:                 │   │
│  │        GET resource from cluster                             │   │
│  │        If exists with deletionTimestamp → FAIL (terminating) │   │
│  │        If exists without OPM labels    → FAIL (untracked)   │   │
│  │    Skip this step if previous inventory exists               │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 6. APPLY RESOURCES (create/update first)                     │   │
│  │    Server-side apply all rendered resources                  │   │
│  │    Track: any errors?                                        │   │
│  └──────────────┬───────────────────────────────────────────────┘   │
│                 ▼                                                   │
│          ┌──────────────┐                                           │
│          │ ALL APPLIED  │                                           │
│          │ SUCCESSFULLY?│                                           │
│          └──────┬───────┘                                           │
│            YES  │    NO                                             │
│           ┌─────┘    └──────────────────────────────┐               │
│           ▼                                         ▼               │
│  ┌──────────────────────┐              ┌──────────────────────────┐ │
│  │ 7a. PRUNE STALE      │              │ 7b. SKIP PRUNE + WRITE   │ │
│  │   (unless --no-prune)│              │   Report partial failure │ │
│  │   Delete each stale  │              │   Inventory NOT updated  │ │
│  │   resource           │              │   Old resources remain   │ │
│  └──────────┬───────────┘              └──────────────────────────┘ │
│             ▼                                                       │
│  ┌──────────────────────────────────────────────┐                   │
│  │ 8. WRITE INVENTORY SECRET (full PUT)         │                   │
│  │   a. Create/overwrite change-sha1-<id> entry │                   │
│  │   b. Prepend change ID to index              │                   │
│  │      (or move to front if already present)   │                   │
│  │   c. Update metadata.lastTransitionTime      │                   │
│  │   d. Prune old changes if over max_history   │                   │
│  │   e. PUT the entire Secret                   │                   │
│  └──────────────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

**Create first, then prune.** New resources are applied before stale ones are
deleted. This is safer — the new Service exists before the old one is removed — but briefly doubles the resource count. This is acceptable.

**Write nothing on partial failure.** If any resource fails to apply, the
inventory is NOT updated and stale resources are NOT pruned. This means:

- Old resources remain running (safe — nothing is torn down prematurely).
- Retrying the apply converges naturally (same previous inventory, same stale
  computation).
- Ghost resources from partially successful applies are possible (see Scenario
  F below) but are caught by label-based discovery.

**Automatic pruning with opt-out.** Pruning is the default behavior. A
`--no-prune` flag allows users to skip pruning when desired.

**Component rename safety check.** Because `component` is part of the inventory
entry identity, a component rename (e.g., `"app"` → `"server"`) makes the old entry look stale — it appears in `previous - current`. However, Kubernetes identifies resources by GVK + namespace + name only (without component), so deleting the "stale" entry would destroy the live resource that the "new" entry refers to. Step 5b prevents this: before pruning, each stale entry is checked against the current set using only `group + kind + namespace + name`. If a match is found (differing only in component), the entry is removed from the stale set. This ensures component renames never trigger destructive deletes.

## Command Impact

```text
┌───────────┬──────────────────────────┬────────────────────────────────────┐
│ Command   │ Today (labels only)      │ With inventory                     │
├───────────┼──────────────────────────┼────────────────────────────────────┤
│ apply     │ Apply, no prune          │ Apply + prune stale + write inv.   │
│ diff      │ Full API scan, noisy     │ Targeted fetch from inv., precise  │
│ delete    │ Full API scan            │ Read inventory, precise delete     │
│ status    │ Full API scan            │ Read inventory, targeted fetch     │
├───────────┼──────────────────────────┼────────────────────────────────────┤
│ ALL       │ —                        │ Fall back to label-based discovery │
│           │                          │ if no inventory Secret found       │
└───────────┴──────────────────────────┴────────────────────────────────────┘
```

### Labels Complement the Inventory

Labels remain on all managed resources. They serve as:

- **Fallback** when inventory is missing (graceful degradation).
- **Human-readable identification** (`kubectl get ... -l module.opmodel.dev/name=minecraft`).
- **Input for future Layer 2 orphan detection** (see Deferred Work).

## Scenarios

### Scenario A: Normal Rename (the Jellyfin Case) [x]

```text
previous = {SS/minecraft, Svc/minecraft, PVC/config}
current  = {SS/minecraft-server, Svc/minecraft-server, PVC/config}
stale    = {SS/minecraft, Svc/minecraft}

Apply: SS/minecraft-server OK, Svc/minecraft-server OK, PVC/config OK
All succeeded → prune SS/minecraft, Svc/minecraft → write inventory
Result: Clean. [x]
```

### Scenario B: Partial Failure — No Inventory Write [x]

```text
previous = {SS/minecraft, Svc/minecraft, PVC/config}
current  = {SS/minecraft-server, Svc/minecraft-server, PVC/config}
stale    = {SS/minecraft, Svc/minecraft}

Apply: SS/minecraft-server OK, Svc/minecraft-server FAIL FAILS, PVC/config OK
Failure → skip prune, skip inventory write

Cluster state:
  SS/minecraft        still running (not pruned — correct!)
  Svc/minecraft       still running (not pruned — correct!)
  SS/minecraft-server  created (but Svc missing → incomplete)
  PVC/config         unchanged

User fixes and re-runs apply:
  previous still = {SS/minecraft, Svc/minecraft, PVC/config}  (not updated)
  current        = {SS/minecraft-server, Svc/minecraft-server, PVC/config}
  stale          = {SS/minecraft, Svc/minecraft}

  All apply OK this time → prune stale → write inventory
  Result: Clean on retry. [x]
```

### Scenario C: Component Removed Entirely [x]

```text
v1: Module with 3 components (app, cache, worker)
previous = {Deploy/app@app, Deploy/cache@cache, Deploy/worker@worker,
            Svc/app@app, Svc/cache@cache}

v2: Module removes "cache" component
current  = {Deploy/app@app, Deploy/worker@worker, Svc/app@app}
stale    = {Deploy/cache@cache, Svc/cache@cache}

Apply all OK → prune Deploy/cache, Svc/cache → write inventory
Result: Clean. [x]
```

### Scenario D: First-Time Apply (No Existing Inventory) [x]

```text
previous = ∅  (no Secret found)
current  = {SS/minecraft, Svc/minecraft, PVC/config}
stale    = ∅

Apply all OK → nothing to prune → write inventory (creates Secret)
Result: Clean. [x]
```

### Scenario E: Someone Deletes the Inventory Secret [x]

```text
previous = ∅  (Secret was deleted by someone)
current  = {SS/minecraft-server, Svc/minecraft-server, PVC/config}
stale    = ∅

Apply all OK → nothing to prune → write inventory (recreates Secret)

Old resources (SS/minecraft, Svc/minecraft) are ORPHANS.
BUT: they still have OPM labels, so label-based discovery
(`opm mod diff`, `opm mod status`) can find them.

Result: Graceful degradation. Inventory loss = no pruning,
        but not catastrophic. [x]
```

### Scenario F: Ghost Resource from Failed Apply

This is the known trade-off of the "write nothing on failure" policy.

```text
After Scenario B, we have:
  SS/minecraft        ← in inventory (previous)
  SS/minecraft-server  ← NOT in inventory (created but inventory not written)

User decides to revert value back to original name "minecraft":
  previous = {SS/minecraft, Svc/minecraft, PVC/config}  (unchanged from v1)
  current  = {SS/minecraft, Svc/minecraft, PVC/config}   (reverted)
  stale    = ∅

  Apply all OK → nothing to prune → write inventory

  BUT: SS/minecraft-server is STILL on the cluster!
  It was created during the failed attempt but never tracked.
```

This ghost resource:

- **Is NOT in the inventory** — inventory was never written for the failed apply.
- **HAS OPM labels** — labels are injected before apply, so the resource is
  labeled.
- **Is detectable** by `opm mod diff` and `opm mod status` (label-based
  discovery can find it as an orphan).
- **Is NOT auto-pruned** — inventory-based pruning cannot catch it.

```text
┌──────────────────────────────────────────────────────────────────┐
│  GHOST RESOURCE: SS/minecraft-server                             │
│                                                                  │
│  In inventory?  NO  (inventory was never written)                │
│  Has OPM labels? YES (labels are injected before apply)          │
│                                                                  │
│  Detectable by:                                                  │
│    [x] opm mod diff   (label-based orphan detection)             │
│    [x] opm mod status (label-based discovery)                    │
│    [ ] Inventory pruning (not in previous inventory)             │
│                                                                  │
│  This is exactly why labels complement the inventory.            │
│  The inventory handles the happy path (rename → prune).          │
│  Labels catch the edge cases (ghosts from failed applies).       │
└──────────────────────────────────────────────────────────────────┘
```

This is acceptable. Ghost resources are a rare edge case (partial failure followed by a revert). Label-based discovery handles them. A future "Layer 2" enhancement can add automatic ghost cleanup (see Deferred Work).

### Scenario G: Resource Kind Changes [x]

```text
v1 inventory: {Deployment/minecraft@app, Svc/minecraft@app}
v2 render:    {StatefulSet/minecraft@app, Svc/minecraft@app}

stale = {Deployment/minecraft@app}  ← correctly identified (GVK is part of identity)

Apply StatefulSet/minecraft OK → prune Deployment/minecraft → write inventory
Result: Clean. [x]
```

### Scenario H: Idempotent Re-Apply [x]

```text
Apply #1: module=1.0.0, values=X → change-sha1-a3b8f2e1 (created)
  Index: ["change-sha1-a3b8f2e1"]

Apply #2: module=1.0.0, values=X → change-sha1-a3b8f2e1 (same hash!)
  Entry overwritten with updated timestamp
  Index: ["change-sha1-a3b8f2e1"]  (no growth)

Apply #3: module=1.1.0, values=X → change-sha1-7f2c9d01 (new hash)
  New entry created
  Index: ["change-sha1-7f2c9d01", "change-sha1-a3b8f2e1"]

Result: History tracks state transitions, not CLI invocations. [x]
```

### Scenario I: Local Module Template Change [x]

```text
Apply #1: local module, values=X, manifests produce digest D1
  changeID = SHA1("" + "" + X + D1) → change-sha1-abc12345

Developer modifies a component template (adds a port, changes nothing else).

Apply #2: local module, values=X, manifests produce digest D2 (≠ D1)
  changeID = SHA1("" + "" + X + D2) → change-sha1-def67890 (different!)

  New change entry created. Previous inventory available for pruning.
  Result: Template changes in local modules are correctly tracked. [x]
```

Without `manifestDigest` in the hash inputs, both applies would produce the same change ID (repository and version are both empty, values are the same). The `manifestDigest` captures what actually changed — the rendered output.

### Scenario J: Component Rename (Safety Check) [x]

A module author renames a component from `"app"` to `"server"`. The resources it produces are identical — same GVK, same namespace, same name. Only the component provenance changes.

```text
v1: Component "app" produces StatefulSet and Service
  previous = {SS/minecraft@app, Svc/minecraft@app, PVC/config@app}

v2: Component renamed to "server" (same resources, same spec)
  current  = {SS/minecraft@server, Svc/minecraft@server, PVC/config@server}

  Raw stale (identity includes component):
    stale = {SS/minecraft@app, Svc/minecraft@app, PVC/config@app}

  Step 5b — Component rename safety check:
    SS/minecraft@app  → current has SS/minecraft@server  (same K8s resource) → REMOVE
    Svc/minecraft@app → current has Svc/minecraft@server (same K8s resource) → REMOVE
    PVC/config@app    → current has PVC/config@server    (same K8s resource) → REMOVE

  Final stale = ∅

Apply all OK → nothing to prune → write inventory
Result: Clean. No destructive delete from a component rename. [x]
```

Without the safety check, all three resources would be deleted and immediately recreated — causing pod restarts, potential PVC orphaning, and unnecessary downtime. The safety check recognizes that the Kubernetes resources are the same and only the OPM-level provenance changed.

### Scenario K: Pre-existing Untracked Resources

First-time apply (no inventory Secret) where resources with matching GVK + namespace + name already exist on the cluster.

```text
Inventory: none (first-time apply)
Rendered:  {SS/minecraft@app, Svc/minecraft@app, PVC/config@app}

Pre-apply existence check (step 5c) queries cluster:
  SS/minecraft   → does not exist    OK
  Svc/minecraft  → does not exist    OK
  PVC/config     → EXISTS            CONFLICT

Apply FAILS before any resource is touched:
  "Resource PersistentVolumeClaim/config in namespace default already exists
   but is not tracked by this release. Delete it manually or use --adopt
   to take ownership."

Result: Fail-safe. No silent adoption of untracked resources. [ ]
```

This prevents OPM from accidentally patching resources created by `kubectl`, Helm, or another OPM release. Without this check, SSA apply would merge into the existing resource and add it to the inventory — potentially corrupting configuration managed by another tool.

The `--adopt` flag is a future escape hatch for intentional adoption (see Open Questions).

### Scenario L: Resource in Terminating State

User runs delete, then immediately re-applies before deletion completes.

```text
Timeline:
  t0: opm mod delete → all resources enter Terminating
  t1: opm mod apply  → user didn't wait for deletion to finish

Pre-apply existence check (step 5c) queries cluster:
  SS/minecraft  → EXISTS with deletionTimestamp set  TERMINATING
  Svc/minecraft → does not exist                     OK
  PVC/config    → EXISTS with deletionTimestamp set  TERMINATING

Apply FAILS before any resource is touched:
  "Resource StatefulSet/minecraft in namespace default is being deleted.
   Wait for deletion to complete before applying."

Result: Fail-safe. No race condition with pending deletion. [ ]
```

Kubernetes accepts patches on terminating resources (SSA apply returns success) until finalizers complete. Without this check, the apply would "succeed" but the resource would be garbage-collected shortly after when finalizers finish — leaving the inventory pointing to resources that no longer exist.

This check applies to ALL resources, not just PVCs. Any resource in a terminating state should block the apply.

### Scenario M: Cross-Namespace Resource Migration

A value change moves resources from one namespace to another. The inventory tracks per-entry namespaces, so the stale set is correct. But pruning requires DELETE permission in the **old** namespace.

```text
v1: Module produces resources in namespace "games"
  previous = {SS/minecraft@app (ns:games), Svc/minecraft@app (ns:games)}

v2: User changes target namespace to "production"
  current  = {SS/minecraft@app (ns:production), Svc/minecraft@app (ns:production)}
  stale    = {SS/minecraft@app (ns:games), Svc/minecraft@app (ns:games)}

Apply to "production" OK → prune from "games" namespace → write inventory

Pruning requires DELETE permission in the OLD namespace ("games").
If RBAC denies it: prune fails, but resources in "production" were applied.

Result: Requires cross-namespace RBAC for pruning. [ ]
        Should warn user if prune fails due to permissions.
        Stale resources in "games" become orphans if RBAC is insufficient.
```

This also applies to modules that produce resources in multiple namespaces simultaneously (e.g., `ConfigMap` in `kube-system` and `Deployment` in `default`). The prune operation must handle each resource's actual namespace from the inventory entry, not a single `--namespace` flag.

### Scenario N: CRD and Custom Resource in Same Module

A module bundles a CRD and instances of that CRD. The weight system applies CRDs first (weight -100) and custom resources last (weight 1000). But there is no wait for CRD establishment between weight groups.

```text
Module produces a CRD and an instance of that CRD:
  current = {CRD/foos.example.com (weight:-100), Foo/my-foo (weight:1000)}

Apply CRD/foos.example.com → OK (weight -100, applied first)
Apply Foo/my-foo            → FAIL "the server could not find the
                                     requested resource" (API not ready)

Partial failure → skip prune, skip inventory write (Scenario B applies)

User retries:
  CRD is now established → Foo/my-foo applies OK → write inventory
  Result: Clean on second attempt. [x]

BUT: First-time users will be confused by the deterministic failure.
     Future mitigation: poll for CRD readiness between weight groups.
```

The inventory design handles this correctly — partial failure means no inventory write, and retry converges. This is a known first-apply pain point for CRD-producing modules, not an inventory format problem.

### Scenario O: Namespace as Module Output + Pruning

A module produces a Namespace resource alongside application resources. When the module is deleted or the Namespace becomes stale, pruning the Namespace triggers Kubernetes cascading deletion of **everything inside it** — including resources from other modules or tools.

```text
v1: Module produces a Namespace and resources within it
  current = {Namespace/games@infra, SS/minecraft@app (ns:games),
             Svc/minecraft@app (ns:games)}
  Apply all OK → write inventory

v2: Module removes the Namespace component (or module is deleted)
  stale = {Namespace/games@infra, SS/minecraft@app, Svc/minecraft@app}

  Prune Namespace/games:
    → K8s cascading delete destroys ALL resources in "games"
    → Including resources from OTHER modules/tools in that namespace
    → Data loss potential is high

Result: Namespace pruning is destructive beyond OPM's scope. [ ]
```

Recommendation: Exclude `kind: Namespace` from inventory-based pruning by default. Require an explicit `--prune-namespaces` flag to override. This mirrors Helm's approach of protecting namespaces from release-scoped deletion. Alternative: warn during prune if the stale set contains a Namespace and require interactive confirmation.

### Scenario P: Empty Render Wipes All Resources

A misconfiguration or conditional in CUE causes the module to render zero resources. The stale set becomes the entire previous inventory.

```text
v1: Module renders 5 resources normally
  previous = {SS/minecraft@app, Svc/minecraft@app, PVC/config@app,
              CM/settings@app, Secret/creds@app}

v2: CUE conditional evaluates to false, producing zero resources
    (e.g., user sets enabled: false, or a typo excludes the component)
  current = (empty set)
  stale   = {SS/minecraft@app, Svc/minecraft@app, PVC/config@app,
             CM/settings@app, Secret/creds@app}

Apply (nothing to apply) → prune ALL 5 resources → write inventory

Result: Complete wipe from an accidental empty render. [ ]
```

This is a dangerous failure mode because the empty render is silent — no apply errors occur, and the prune logic is technically correct. Recommendation: add a safety threshold. If the current render is empty and the previous inventory is non-empty (100% pruning), require `--force` or interactive confirmation. This protects against accidental misconfiguration without affecting intentional module removal (which uses `opm mod delete`).

### Additional Known Edge Cases

The following edge cases are acknowledged but do not require dedicated scenarios. They are tracked here for implementors:

1. **Labels not guaranteed on resources.** Label injection is the CUE
   transformer's responsibility. Go code does not validate label presence on    rendered resources. A resource without OPM labels is tracked by the    inventory but invisible to label-based fallback discovery. The inventory is    the authoritative record; labels are best-effort.

2. **Change ID collision.** SHA1 truncated to 8 hex chars = 32 bits of
   entropy. With `max_history=10`, collision is negligible (~65,000 entries    for 50% birthday-paradox probability). On collision, the old entry is    silently overwritten. Acceptable.

3. **Inventory Secret in label queries.** The inventory Secret carries OPM
   labels and would appear in `DiscoverResources()` results. The existing    discovery code must be updated to exclude resources with    `opmodel.dev/component: inventory` from workload queries.

4. **Partial delete leaving stale inventory.** If `opm mod delete` removes
   cluster resources but crashes before deleting the inventory Secret,    subsequent applies find the stale inventory. Pruning already-deleted    resources should no-op (treat 404 on delete as success, not an error).

5. **Concurrent applies (CI/CD).** Two simultaneous `opm mod apply` calls
   can overwrite each other's inventory. Mitigation: include    `resourceVersion` in the PUT for Kubernetes optimistic concurrency. Fail    on conflict with a clear error message.

6. **API version migration.** A stale entry stored with `v: "v1beta1"` may
   need deletion after the API version is removed from the cluster. The    delete call should use the current preferred version from API discovery,    not the stored version.

7. **Finalizers blocking pruning.** Stale resources with finalizers enter
   Terminating state on delete. The prune operation should use a non-blocking    delete (no wait) to avoid hanging the apply flow indefinitely.

8. **Multiple modules sharing resource names.** Two modules in the same
   namespace producing the same GVK + namespace + name is undefined behavior.    The pre-apply existence check (Step 5c) catches this on first apply but    not on subsequent applies where the inventory already exists.

9. **Simultaneous component rename + resource rename.** Both the component
   and resource name change at once. The safety check (Step 5b) correctly    does NOT fire — names differ, so these are genuinely different resources.    Old resources are pruned correctly. This is expected behavior.

10. **Secret size under large values.** Modules with large CUE values
    (embedded certificates, large config blocks) could push the Secret toward     the 1MB etcd limit. Consider truncating or omitting the `values` field in     change entries if the Secret exceeds a size threshold (~800KB).

## Architectural Limitations

The following scenarios represent fundamental limitations of the per-release, Secret-based inventory design. Some may be addressable within the current architecture; others may require controller-based coordination or be declared out of scope. These are documented for transparency — decisions on which to address will be made during implementation.

### Cross-Release Resource Conflicts

The inventory is per-release with no awareness of other releases in the same namespace.

**Shared resource collision.** If Release A and Release B both produce
`ConfigMap/shared-config`, each tracks it independently. When Release A removes it and prunes, Release B's resource is destroyed. Release B's inventory still references it. Next `status` sees a missing resource with no explanation.

The pre-apply existence check (Step 5c) catches this on **first apply only**. On subsequent applies where both inventories exist, there is no cross-release coordination.

**Resource transfer between releases.** Moving a resource from Release A to
Release B requires A to prune (removed from template) and B to create (added to template). Delete-then-create causes downtime. No "transfer ownership" mechanism exists.

**Scope:** May be out of scope. Requires cluster-wide coordination (controller
or CRD). Similar to Helm releases sharing resources — undefined behavior.

### Cluster-Scoped Resource Ownership

The inventory Secret is namespace-scoped. Modules can produce cluster-scoped resources (CRDs, ClusterRoles, ClusterRoleBindings, Namespaces).

**Conflict across namespaces.** Release A in `team-a` namespace and Release B
in `team-b` namespace both produce `ClusterRole/app-reader`. Each inventory considers it "theirs." If Release A is deleted, the ClusterRole is pruned, breaking Release B silently.

**Scope:** Likely out of scope for Secret-based inventory. Full solution
requires cluster-wide registry or ModuleRelease CRD (see Deferred Work). Mitigation: lint rule warning when modules produce cluster-scoped resources.

### Controller-Created Side-Effect Resources

The inventory tracks only resources OPM directly renders. Resources created as side effects by Kubernetes controllers are invisible.

**Examples:**
- OPM creates `Certificate/my-app-tls` (cert-manager) → cert-manager creates
  `Secret/my-app-tls`. OPM does not track that Secret.
- OPM deletes `Certificate/my-app-tls` during prune → TLS Secret orphaned if
  cert-manager does not use ownerReferences.
- OPM creates `HelmRelease` CR (Flux) → Flux creates chart resources. OPM has
  no visibility.

**Scope:** By design. OPM respects Kubernetes ownership semantics. Cleanup
responsibility falls on operators' ownerReference behavior. Not solvable without wrapping all resources in a ModuleRelease CRD parent.

### State Reconstruction After Inventory Loss

Scenario E acknowledges this: if the inventory Secret is deleted, previous state is lost. Next apply starts with `previous = ∅`.

**Consequences:**
- All previously tracked resources become permanent orphans (no pruning).
- Label-based discovery can find them (they carry OPM labels), but
  inventory-based pruning cannot act.
- No automated recovery path in current design.

**Root cause:** Inventory Secret is single point of truth with no redundancy.
Unlike Helm (stores full manifests), OPM stores only resource IDs and values. Cannot automatically determine previous render without inventory.

**Scope:** Partially addressable via Layer 2 label-based orphan detection (see
Deferred Work). Full solution requires external state store or CRD.

### Concurrent Apply Races (CI/CD Pipelines)

Edge case #5 mentions `resourceVersion` for optimistic concurrency, but this is noted as mitigation, not a complete solution.

**Read-modify-write cycle:**
```
Pipeline A: READ inventory → compute stale → apply → WRITE inventory
Pipeline B: READ inventory → compute stale → apply → WRITE inventory
```

If A and B overlap, B's WRITE overwrites A's inventory state. B's stale computation was based on stale data. Even with `resourceVersion`, the failure mode is "retry entire apply" (expensive).

**Scope:** For GitOps/CI pipelines triggering on every commit, this is a
realistic scenario. `resourceVersion` + retry is acceptable for CLI use. Full solution requires distributed lock or controller serialization.

### Prune Ordering and Dependent Teardown

The RFC specifies "create first, then prune" but does not define **prune ordering**. When stale set contains dependent resources:

- Stale `Ingress/app` routes to stale `Service/app`
- Stale `Service/app` references stale `Deployment/app`
- Stale `PVC/data` bound to stale `StatefulSet/app`

**Current state:** No ordering guarantee within the stale set.

**Proposed:** Reverse weight order (custom resources pruned first, CRDs last).
This matches "tear down instances before definitions" and aligns with apply ordering semantics.

**Gaps in reverse weight order:**
- Resources at same weight have no ordering guarantee. Usually fine (K8s
  handles Service-before-Deployment gracefully), but not guaranteed safe for   all types.
- Cross-resource finalizers not weight-aware. If CR with finalizer referencing
  a Service is pruned first, finalizer controller may race with Service prune.

**Recommendation:** Use reverse weight order with one addition — **Namespaces
always pruned last** (or excluded by default per Scenario O). This handles the one case where ordering truly matters.

**Scope:** Addressable within current design. Should be specified in
implementation.

### Data-Bearing Resource Lifecycle

The inventory treats all resources equally. No concept of "this resource carries persistent data."

**PVC rename = data loss.** Value change causes `PVC/minecraft-data` to become
`PVC/minecraft-server-data`. Inventory correctly identifies old PVC as stale and prunes it. Data is gone. Cannot distinguish "rename reference" from "replace with new empty resource."

**Rollback with data loss.** Change history enables future rollback, but
rolling back to a change with `PVC/minecraft-data` when current has `PVC/minecraft-server-data` means: create new PVC (empty), prune old PVC (with data). No data migration.

**Compounded by RFC-0003.** Immutable config hash-suffixes names. If a module
makes a PVC-related ConfigMap immutable and uses hashed name in PVC name template, every config change creates new PVC.

**Scope:** Requires resource-class awareness (PVC ≠ ConfigMap). Possible
mitigations:
- Detect PVC in stale set, require `--force-prune-pvcs` flag
- Lint rule: warn on PVC name templates using dynamic values
- Future: PVC migration operator pattern

Fundamental issue: no way to encode "this is stateful" in resource identity.

### Apply-Time Resource Conflicts (Immutable Fields)

Open Questions section (line 1519) acknowledges Kubernetes enforces field immutability (`spec.selector` on Deployments, `spec.clusterIP` on Services, `spec.volumeClaimTemplates` on StatefulSets).

**Current state:** Inventory knows what was previously applied but has no
mechanism to detect or resolve immutable field changes proactively.

**Only approach:** Try-and-detect at apply time (catch 422 "field is
immutable," fall back to delete+recreate). For StatefulSet `spec.volumeClaimTemplates` changes, required delete+recreate destroys all associated PVCs.

**Gap:** Inventory can track state change but cannot orchestrate safe migration
path (drain pods, backup PVCs, delete, recreate, restore).

**Scope:** Partially addressable. Try-and-detect works. Safe migration for
stateful resources requires higher-level orchestration beyond inventory's responsibility.

### Multi-Namespace Atomic Operations

Scenario M covers cross-namespace migration: RBAC may block pruning in old namespace. Deeper problem is **atomicity**.

**Apply-then-prune flow:** Apply to namespace A, then prune from namespace B.
If prune fails (RBAC, network, API server error), apply already succeeded. Resources now in both namespaces; inventory points at new namespace. Stale resources in old namespace orphaned with no automatic retry.

**Modules producing resources in multiple namespaces simultaneously** (e.g.,
RBAC in `kube-system`, workloads in `default`) have no transactional guarantee. Partial failure leaves resources scattered with inconsistent state.

**Scope:** Fundamental Kubernetes limitation (no cross-namespace transactions).
Best effort: fail entire apply if any namespace is unreachable. Cannot fully solve without distributed transaction coordinator.

### Runtime Drift Detection (Live State vs Desired State)

The inventory records what OPM **rendered and applied**. Does not detect
**runtime drift** — changes made by external actors after apply.

**Examples:**
- `kubectl edit deployment/minecraft` changes replica count. Inventory shows
  OPM-declared state. `opm mod diff` (if comparing inventory to live) can   detect, but inventory itself is oblivious.
- Admission webhook mutates resources during apply (adds annotations, changes
  labels). Inventory records pre-mutation state, not what landed in etcd.

**SSA's `managedFields`** partially addresses field-level ownership, but
inventory does not interact with it. No mechanism to "enforce" inventory state or detect when live diverges.

**Scope:** May be out of scope. Drift detection is typically controller
responsibility. CLI tool records intent; controllers enforce. For CLI-only use, drift detection would require `opm mod status` to compare inventory to live (already planned).

## Deferred Work

The following are explicitly out of scope for this RFC and deferred to future enhancements:

### Layer 2 Label-Based Orphan Detection in Apply

A second pruning pass during `apply` that uses label-based discovery (full API scan) to find ghost resources not tracked by the inventory. This catches edge cases like Scenario F.

```text
Layer 1: Inventory-based (fast, precise)
  stale = previous_inventory - current_inventory
  → Handles renames, component removal
  → Only works if inventory exists and is up-to-date

Layer 2: Label-based (broad, catches ghosts)
  orphans = label_discovered - current_render
  → Handles ghosts from failed applies
  → Handles lost inventory
  → More expensive (full API resource scan)
  → Already implemented in findOrphans() in diff.go
```

### Rollback Support

The change history model in this RFC provides the foundation for rollback. A future `opm mod rollback --revision <change-id>` command could:

1. Read the target change entry from the Secret.
2. Re-render the module using the stored values and module reference.
3. Apply normally (which creates a new change entry — rollback is just another
   change).

This is deferred because it requires the module OCI reference to be resolvable at rollback time, and the interaction with local modules needs further design.

### ModuleRelease CRD with ownerReferences

The long-term vision: a `ModuleRelease` Custom Resource Definition with `ownerReferences` on all child resources. This enables:

- Native Kubernetes garbage collection (delete parent → children auto-delete).
- Controller-based reconciliation.
- First-class `kubectl get modulereleases` experience.

The `data.metadata` blob uses `kind: Release` and `apiVersion: core.opmodel.dev/v1alpha1` specifically so that the schema can migrate to a CRD with minimal changes. The inventory Secret is a stepping stone that may become permanent for CLI-only users while the CRD serves the controller path.

## Open Questions

### Immutable Field Handling During Apply

Kubernetes enforces field immutability via hardcoded `ValidateUpdate()` functions in per-resource API server strategies. When Server-Side Apply attempts to change an immutable field, the API server returns:

```text
HTTP 422 Unprocessable Entity
reason: "Invalid"
causes[].message: "field is immutable"
causes[].field:   "<field path>"   (e.g., "spec.clusterIP", "spec.selector")
```

This is an apply-time behavior question, not an inventory format question. The inventory does not need to change to support immutable field handling. However, the apply flow must account for it.

#### The Problem

Certain spec fields cannot be updated in place. A change to these fields requires deleting the resource and recreating it, which is destructive — it can cause pod restarts, PVC orphaning, and service interruption. Common immutable fields:

```text
┌──────────────────┬──────────────────────────────────────────────────────┐
│ Resource         │ Immutable Fields                                     │
├──────────────────┼──────────────────────────────────────────────────────┤
│ All              │ metadata.name, metadata.namespace, metadata.uid      │
│ Deployment       │ spec.selector                                        │
│ StatefulSet      │ spec.selector, spec.volumeClaimTemplates,            │
│                  │ spec.podManagementPolicy, spec.serviceName            │
│ DaemonSet        │ spec.selector                                        │
│ Job              │ spec.selector, spec.template, spec.completions       │
│ Service          │ spec.clusterIP, spec.clusterIPs,                     │
│                  │ spec.ipFamilyPolicy, spec.ipFamilies                  │
│ PVC              │ spec.storageClassName, spec.accessModes,             │
│                  │ spec.volumeMode, spec.volumeName (after binding)      │
│ Secret/ConfigMap │ All data fields (if immutable: true)                 │
└──────────────────┴──────────────────────────────────────────────────────┘
```

There is **no machine-readable immutability metadata** in the Kubernetes OpenAPI schema. KEP-1101 proposed adding `x-kubernetes-immutable` as an OpenAPI extension for built-in resources but was never implemented. For CRDs, Kubernetes 1.25+ supports CEL validation rules (`self == oldSelf`), but this is opt-in per CRD and not available for built-in types.

#### Industry Approaches

```text
┌──────────────────────┬─────────────────────────────────────────────────┐
│ Approach             │ Used By                                         │
├──────────────────────┼─────────────────────────────────────────────────┤
│ Try-and-detect       │ Flux, kapp (fallback-on-replace)                │
│                      │ SSA apply → catch 422 "field is immutable"      │
│                      │ → fall back to delete+recreate                  │
│                      │ Pro: works for any resource including CRDs      │
│                      │ Con: reactive, not proactive; cannot distinguish│
│                      │   immutability from other Invalid errors        │
├──────────────────────┼─────────────────────────────────────────────────┤
│ Hardcoded field list │ Pulumi                                          │
│                      │ Maintain table of {GVK, fieldPath} pairs        │
│                      │ Pro: proactive warnings before applying         │
│                      │ Con: manual maintenance, version-dependent      │
├──────────────────────┼─────────────────────────────────────────────────┤
│ User annotation      │ kapp, Flux, ArgoCD                              │
│                      │ Per-resource opt-in to force recreate           │
│                      │ Pro: user declares intent explicitly            │
│                      │ Con: requires user knowledge per resource       │
├──────────────────────┼─────────────────────────────────────────────────┤
│ Blunt --force        │ Helm                                            │
│                      │ Delete+recreate ALL resources in release        │
│                      │ Pro: simple, always works                       │
│                      │ Con: destructive, causes downtime for all       │
└──────────────────────┴─────────────────────────────────────────────────┘
```

#### Likely Direction for OPM

The **try-and-detect** pattern (kapp's `fallback-on-replace`) is the most pragmatic approach:

1. SSA apply the resource.
2. If the API server returns 422 with `"field is immutable"` in
   `causes[].message`, detect this as an immutable field conflict.
3. In interactive mode: warn the user and offer delete+recreate.
4. In non-interactive mode (CI): fail with a clear message.
5. With an explicit `--force` flag: automatically delete+recreate.

This could be augmented with a known-immutable-fields table for **proactive warnings** in `opm mod diff` — showing the user which fields would require recreation before they run apply.

This question is deferred to implementation. The inventory format does not need to change to support any of these approaches.

### Pre-Apply Resource Existence Checks

When no inventory Secret exists (first-time apply), OPM must verify that the resources it intends to create do not already exist on the cluster. This prevents two failure modes:

1. **Untracked resource adoption**: A resource with the same GVK + namespace +
   name exists but was created by another tool (kubectl, Helm, another OPM    release). Applying over it would silently adopt it into this release's    inventory.

2. **Terminating resource race**: A resource is being deleted (has
   `deletionTimestamp` set). SSA apply "succeeds" on terminating resources, but    the resource will be garbage-collected when finalizers complete.

Both cases should fail the apply with a clear error message.

#### Design Considerations

**Performance**: The check requires one GET per resource in the rendered set. For
a typical module (3-10 resources), this adds 3-10 API calls before apply begins. This is acceptable for first-time applies only. Subsequent applies (where an inventory exists) can skip this check — the inventory already tracks what OPM owns.

**Terminating detection on subsequent applies**: The first-time-only scope means
a re-apply after `opm mod delete` could still hit the terminating race if the inventory Secret is deleted before the resources finish terminating. The terminating check should also run when the inventory exists but the previous inventory is empty (e.g., the Secret was just created). This needs further design.

**TOCTOU race**: A resource could be created between the existence check and the
apply. This is inherent to any check-then-act pattern. The risk is low for CLI-driven workflows (no concurrent controllers creating the same resources). SSA's conflict detection provides a secondary safety net.

**`--adopt` flag**: A future escape hatch that allows intentional adoption of
existing untracked resources. When set, the pre-apply check would skip the "untracked resource" failure and instead add the resource to the inventory. Terminating resources should still fail even with `--adopt`.

**`--wait-for-deletion` flag**: An alternative to failing on terminating
resources — wait for deletion to complete, then proceed. This is a convenience for the common "delete then re-apply" workflow. Deferred to implementation.

## References

- [Helm Storage System](https://helm.sh/docs/topics/advanced/) — Helm's Secret/ConfigMap/SQL storage drivers
- [Timoni Module Specification](https://timoni.sh/module/) — Timoni's lightweight inventory Secret
- [Carvel kapp Resource Management](https://carvel.dev/kapp/) — kapp's ConfigMap-based tracking
- [Flux Kustomization Inventory](https://fluxcd.io/flux/components/kustomize/kustomizations/) — Flux's `.status.inventory`
- [ArgoCD Resource Tracking](https://argo-cd.readthedocs.io/en/latest/user-guide/resource_tracking/) — ArgoCD's label/annotation tracking methods
- [Kubernetes Server-Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) — SSA managedFields documentation
