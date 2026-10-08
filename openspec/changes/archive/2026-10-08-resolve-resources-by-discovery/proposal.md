## Why

The cli turns a kind into a resource name by guessing its plural (`internal/kubernetes/resource.go`: a table of well-known kinds, then English rules). A wrong guess addresses a resource the cluster does not serve, and the API server answers 404. `instance delete`, `checkDeletable` and the stale-resource prune read 404 as "already gone", so the object stays in the cluster and the cli drops it from the record. The catalog's `#Objects` admits any kind, so a kind outside the table is reachable.

## What Changes

- The cli asks the cluster's API discovery for the resource name of a group, version and kind. One resolver on the Kubernetes client serves every command that addresses an object: apply, delete, prune, diff, status, tree, list, events, and the operator install, uninstall and migration paths.
- The guessing functions (`KindToResource`, `HeuristicPluralize`, the known-kinds table, `GVRFromUnstructured`) are removed.
- A kind the cluster does not serve is an error that names the kind, group and version. Delete and prune never read it as "already gone"; the record keeps the entry.
- A discovery request that fails (denied, unavailable, any other error) is an error with the API error in its chain, so the command exits with the existing codes (4 denied, 3 unavailable, 1 other). It is never read as "no such kind".
- A read that only asks "does this object exist" before an apply (the first-install check, the diff, the operator's terminating check and migration plan of rendered objects) reads an unserved kind as "the object does not exist", as it read the 404 before. The apply itself still fails with the error that names the kind.
- After the CustomResourceDefinitions of an apply are established, the apply waits, under the same deadline, until discovery serves the kinds the rest of the apply needs.

SemVer: PATCH (a bug fix; no flag, no output contract and no exit code table changes). Before GA it ships as the next beta.N.

No new dependency: discovery is part of `k8s.io/client-go`, already required. Complexity added (Principle VII): one resolver type with an in-memory cache. It replaces a table and a pluralizer of about the same size, and it is the only way to get a correct name for a kind the cli has never seen.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `k8s-helpers`: resource names come from API discovery; the guessed-plural requirements are removed; unserved kinds and discovery failures are specified.

## Impact

- `internal/kubernetes`: `resource.go` (resolver, errors), `client.go` (wiring), `apply.go`, `delete.go`, `diff.go`, `wait.go`.
- `internal/inventory`: `discover.go`, `stale.go`.
- `internal/operator`: `install.go`, `wait.go`, `migration_plan.go`, `migration_execute.go`.
- New test-support package `internal/kubernetes/kubetest` (a resolver for fake clients); every test that builds a `kubernetes.Client` sets it.
- Cost: one discovery GET per distinct group and version a command touches, cached in memory for the life of the command. Dry runs make the same requests.
- Not touched: flags and help text, `internal/publish`, `internal/config`, PersistentVolumeClaim handling, the inventory format.
