## Why

Three follow-ups the order-instance-apply-by-weight change and the wave-1 reviews left open.

**The tree does not sort by weight.** The `mod-tree` main spec requires `opm instance tree` to list each component's resources by ascending weight, then by name ("Tree sorts resources within components by weight"). `groupByComponent` (`internal/kubernetes/tree.go:255-270`) keeps the inventory's order, which is the render order of the last apply; only component names and child nodes are sorted. The requirement's prose also claims the inventory is written in weight order, which is false: `CurrentInventoryEntries` follows render order. The order-instance-apply-by-weight change fixed only the code comment and deferred the sort.

**A dry run fails on a Namespace the same apply creates.** A dry run skips a custom resource whose CustomResourceDefinition the same apply creates, but it still sends a namespaced object whose Namespace the same apply creates. The Namespace's own dry run persists nothing, so the object fails namespace admission with `namespaces "<ns>" not found`, and the dry run reports an error that the real apply would not hit. The same happens for every namespaced object under `--create-namespace`: `Client.EnsureNamespace` (`internal/kubernetes/client.go:105-132`) creates nothing in a dry run and only reports that it would. The `deploy` spec records this as a documented limit; the owner's walkthrough listed it as a follow-up, to be handled like a custom resource of a new CRD: skip it with a warning.

**Three tests do not guard what they claim.** From the cli-a1 and cli-i5 reviews, verified at `origin/main` 5180cad1:

1. The `BudgetStart` wiring in `internal/workflow/apply/apply.go:153-157` is unguarded: deleting `BudgetStart: budgetStart` still passes `go test ./internal/workflow/apply/`.
2. `TestExecute_CRDNotEstablishedSkipsPruneAndInventoryWrite` (`internal/workflow/apply/staging_test.go`) sleeps one second inside a 1.2 s budget and asserts on wall-clock rounding. It is timing-dependent and adds over a second to the unit suite.
3. No command-level test runs an `OPM_NAMESPACE` override through `config.ResolveKubernetes` into `opm instance vet`; only the render unit (`refuseNamespaceOverride` with `Source: SourceEnv`) covers it. `internal/config/resolver_test.go` sets and unsets environment variables with `os.Setenv`/`os.Unsetenv`, which leaks into later tests and wipes a developer's own `OPM_REGISTRY`, `OPM_CONFIG`, `OPM_KUBECONFIG`, `OPM_CONTEXT` and `OPM_NAMESPACE` for the rest of that file's tests (`loader_test.go` does the same for `HOME`, `OPM_REGISTRY` and `OPM_CONFIG`; it stays a follow-up).

## What Changes

- **The tree sorts by weight, then name.** `groupByComponent` sorts each component's resources ascending by `pkg/resourceorder` weight, then by name. Kind and then namespace break the remaining ties only to make the order deterministic; the owner decision names weight, then name. The text, JSON and YAML outputs of `opm instance tree` follow. The `mod-tree` requirement keeps its scenarios; its false sentence about the inventory's order goes.
- **A dry run skips objects in a Namespace the same apply creates.** On `--dry-run`, `opm instance apply` and `opm module apply` do not send a namespaced object whose namespace is (a) a `Namespace` the same apply renders and that did not exist before the apply, or (b) the instance namespace that `--create-namespace` would create. The command logs a warning naming the object and the namespace, counts it in the dry-run summary's skipped count, and exits 0 as before. A custom resource whose CustomResourceDefinition is new keeps its own warning. Cluster-scoped objects and objects in an existing namespace are sent as before. The `--dry-run` help of both commands says what a dry run skips.
- **Test hardening, no behavior change.** `internal/workflow/apply` reads the apply start through a package clock seam, so the staging test sets a budget start an hour back instead of sleeping and asserts the reported elapsed time. A deleted `BudgetStart` line then fails it. A new command test runs `opm instance vet --offline` with `OPM_NAMESPACE` set by `t.Setenv` against the skip-unprovided fixture and expects exit 2 naming `OPM_NAMESPACE`. `resolver_test.go` moves to `t.Setenv`.

Not in this change: a Namespace readiness wait; skipping on a dry run for any reason other than a new CustomResourceDefinition or a new namespace; moving the weight table out of `pkg/resourceorder` (cli-e2e5, which follows this change); operator-managed apply paths, which render and apply in the operator.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mod-tree`: the weight-then-name sort is implemented, and the requirement no longer claims the inventory is stored in weight order.
- `deploy`: a dry run skips a namespaced object whose Namespace the same apply creates, or that `--create-namespace` would create; the custom-resource skip requirement drops the sentence that called this a limit.

## Impact

- **Release class: `fix`, PATCH; ships as the next `1.0.0-beta.N`.** No flag, command or exit code is added or removed. `opm instance tree` prints resources in the order its spec already promises (scripts that parse the JSON or YAML by position see a new order). A dry run that used to report `namespace not found` errors for a new namespace now reports those objects as skipped and exits 0.
- Commands: `opm instance tree`; `opm instance apply --dry-run` and `opm module apply --dry-run` (help text and behavior).
- Packages: `internal/kubernetes` (`tree.go`, `apply.go`), `internal/workflow/apply` (`apply.go`), `internal/cmd/instance` and `internal/cmd/module` (`apply.go` help, a new vet test), `internal/config` (`resolver_test.go`), `tests/integration/apply-staging`.
- Coordination: cli#307 (another session) edits `internal/workflow/apply/apply.go`; it is not a gate. Hunks there stay small. The one expected rebase conflict is #307's `RunClusterGates` hunk, which sits next to the `EnsureNamespaceIfRequested` signature this change edits; whichever PR merges second rebases. cli-e2e5 deletes `pkg/resourceorder` and merges after this change.
- No enhancement decision backs this change; it carries no `enhancement.yaml`.
