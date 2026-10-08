## 1. Resolver in internal/kubernetes

- [ ] 1.1 Add `ResourceResolver`, `KindNotServedError`, `IsKindNotServed` and the discovery-backed resolver (positive cache, mutex, caller's context) to `internal/kubernetes/resource.go`; add `Client.Resources`, `Client.ResourceFor` and `Client.ResourceClientFor`; wire the resolver in `NewClient`
- [ ] 1.2 Add `internal/kubernetes/kubetest` with the resolver for fake clients
- [ ] 1.3 Unit tests against an `httptest` server: irregular plural, one request per group-version, refresh on a miss, unserved group-version, unserved kind, Forbidden and ServiceUnavailable kept in the chain and not NotFound, `/api/v1` path for the core group, subresources skipped, a client with no resolver
- [ ] 1.4 `task fmt`, `task vet`, `task lint` and `task test:unit` green, then commit `fix(kubernetes): add a resource resolver backed by API discovery`

## 2. Callers use the resolver; guessing removed

- [ ] 2.1 Write the failing tests first: delete with an unserved recorded kind keeps the record and reports the kind; prune with an unserved stale kind keeps the entry; discovery of an inventory entry with a failed discovery request is unreadable; first-install check reads an unserved kind as absent and stops on a discovery failure; diff reads an unserved kind as added; apply reports the unserved kind; apply waits for discovery after Established
- [ ] 2.2 `internal/kubernetes`: `apply.go` (resolve, wait for discovery after Established), `delete.go`, `diff.go`, `wait.go`
- [ ] 2.3 `internal/inventory`: `discover.go`, `stale.go`
- [ ] 2.4 `internal/operator`: `install.go`, `wait.go`, `migration_plan.go`, `migration_execute.go`
- [ ] 2.5 Remove `GVRFromUnstructured`, `KindToResource`, `HeuristicPluralize`, `knownKindResources` and their tests; set the `kubetest` resolver on every fake client in tests
- [ ] 2.6 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `fix: resolve resource names by API discovery instead of guessed plurals`
