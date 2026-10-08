## Context

`internal/kubernetes/resource.go` builds a `GroupVersionResource` from a kind with a table of 37 kinds and an English pluralizer. Twelve non-test call sites use it, in `internal/kubernetes`, `internal/inventory` and `internal/operator`. No non-test code uses API discovery or a RESTMapper. The `kubernetes.Client` is built once per command (`NewClient` caches it) and holds a dynamic client and a clientset.

Read paths split into two groups. cli#332 and cli#338 made a failed read of a recorded object "unreadable" (never "missing") and a failed delete a kept record entry. This design reuses those paths; it adds no new failure path.

## Goals / Non-Goals

**Goals:**

- A resource name always comes from the cluster. No guessed plural reaches a cluster.
- An unserved kind and a failed discovery request are two different, typed outcomes, and neither reads as "already gone" on delete or prune.
- Low and predictable request cost.

**Non-Goals:**

- No change to flags, help text, the inventory format, PersistentVolumeClaim handling or exit code tables.
- No disk cache for discovery.
- No fallback from a recorded API version to another served version of the same kind.

## Research & Decisions

### Which discovery client

**Context**: The cli always knows group, version and kind of the object it addresses (a rendered object or an inventory entry). It needs only the resource name.

**Explored**: `k8s.io/client-go` v0.37.1 `restmapper.NewDeferredDiscoveryRESTMapper` with `memory.NewMemCacheClient`; `discovery.ServerResourcesForGroupVersion`; a direct GET of `/api/v1` or `/apis/<group>/<version>` through the discovery REST client.

**Options considered**:

1. Deferred discovery RESTMapper (kubectl's pattern). Pro: standard. Con: it loads every group of the cluster (2 requests with aggregated discovery, one per group-version without), and `restmapper.GetAPIGroupResources` drops groups whose discovery failed, so a denied or unavailable group reads as "no match". That is the empty answer this change must not accept.
2. `ServerResourcesForGroupVersion`. Pro: one request per group-version. Con: for `v1` it returns an empty list on Forbidden and NotFound, and it takes no context.
3. A direct GET of the group-version document with the caller's context. Pro: one request per group-version, the exact API error, the caller's deadline. Con: about 100 lines of own code.

**Decision**: Option 3.

**Rationale**: It is the only option where a failed discovery request cannot look like an unserved kind. It is also the cheapest: an instance touches a handful of group-versions.

### Resolver contract

```go
// ResourceResolver resolves the resource that serves a kind.
type ResourceResolver interface {
    ResourceFor(ctx context.Context, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error)
}

// KindNotServedError: discovery answered; the kind is not served.
// It never unwraps to an API status error, so apierrors.IsNotFound is false.
type KindNotServedError struct{ GVK schema.GroupVersionKind }

func IsKindNotServed(err error) bool

// On Client:
func (c *Client) ResourceFor(ctx, gvk) (schema.GroupVersionResource, error)
func (c *Client) ResourceClientFor(ctx, gvk, namespace) (dynamic.ResourceInterface, error)
```

- A 404 from the group-version document, or a document without the kind, is `*KindNotServedError`: `kind "Widget" of example.io/v1 is not served by the cluster`.
- Any other failure is `discovering resources of <group/version>: %w` with the API error wrapped, so `apierrors.IsForbidden` and the exit code mapping keep working.
- Cache: positive answers only, per group-version, in memory, guarded by a mutex. A miss always asks the server again (and replaces the cached document), because a CustomResourceDefinition can add a kind during the command. The cost of a miss is one request, the same as the 404 read it replaces.
- Subresources (`deployments/status`) are skipped. The first resource whose `kind` matches wins.
- No retry layer is added. client-go's own handling of `Retry-After` stays the only one. The caller's context bounds every request.
- A `Client` built without a resolver and without a clientset returns an error from `ResourceFor`; it never guesses.

### Outcome per call site

| Call site | Unserved kind | Discovery failure |
| --- | --- | --- |
| `applyOne` | per-resource error (existing path) | per-resource error |
| `checkDeletable`, `deleteResource` | per-resource error, record kept | same |
| `DiscoverResourcesFromInventory` | unreadable entry | unreadable entry |
| `PruneStaleResources` | failed entry in `PruneError` | failed entry |
| `FirstInstallCheck` | object does not exist | error, apply stops |
| `fetchLiveState` (diff) | object does not exist (added) | diff error (existing path) |
| `pollObjects` (wait) | error, wait stops | error, wait stops |
| operator `terminatingObjects`, `getLive` of rendered objects | object does not exist | error |
| operator `getLive` of legacy entries, `MoveOwnership`, `deleteProven` | error | error |
| operator `pendingObjects` | pending (existing rule: any error) | pending |

**Rationale for "does not exist" on pre-apply reads**: a module can render a CustomResourceDefinition and an object of its kind. Before the first apply the kind is not served, and the old code read the 404 as "absent". An object of an unserved kind and version cannot be read or taken over; the apply that follows still fails on it with the named error. On delete and prune the same answer is unsafe: the entry can name an API version the cluster stopped serving while the object lives on under another version.

### Discovery after Established

**Context**: The API server updates the discovery document of a group after a CustomResourceDefinition becomes Established, not in the same step. The old code sent the PATCH straight to the guessed path, so it did not depend on discovery.

**Decision**: `Apply` waits, after `waitEstablished` and under the same deadline, until the resolver serves each kind that the second stage uses and a first-stage definition defines. `KindNotServedError` keeps the wait going; any other error stops it. The operator install path applies its definitions with `ApplyOne` and waits with `Wait`; its instance apply goes through `Apply` and gets the same wait.

**Assumption not verified here**: the size of that lag on a real cluster. No cluster is available to this change; the repo's e2e job covers the path.

## Error handling and exit codes

No command gets a new exit code. The resolver's errors go through the paths of cli#332 and cli#338. Where such a path sets the exit code from the error (`ExitCodeFromK8sError`: the diff of rendered objects, the first-install check and the other apply refusals), the code is 4 for Forbidden or Unauthorized, 3 for ServerTimeout or ServiceUnavailable, 1 otherwise; a `KindNotServedError` maps to 1. Where the path has a fixed code, it keeps it: `instance delete` exits 1 for any per-resource failure (`reportInstanceDelete`), and `instance status` shows an unreadable resource as health Unknown and exits 2.

Example, delete of a record with an unserved kind:

```text
WARN reading Widget/demo: kind "Widget" of example.io/v1 is not served by the cluster
```

## Test support

`dynamicfake` stores objects under `meta.UnsafeGuessKindToResource`. The new package `internal/kubernetes/kubetest` gives fake clients a resolver that uses the same function, so resolver and tracker agree. It is imported only by tests. Unit tests of the real resolver run against an `httptest` server and count requests.

## Risks / Trade-offs

- An instance whose record names a kind the cluster no longer serves cannot be deleted with `opm instance delete` until the record is corrected or the kind is served again. This is the requested behaviour; the alternative forgets live objects. An escape flag is an owner decision and is not part of this change.
- A cluster that denies discovery to the user now fails commands that worked before with a guessed name. Kubernetes grants discovery to every authenticated user by default (`system:discovery`).
- One extra request per group-version per command.
