## MODIFIED Requirements

### Requirement: Flag surface matches `opm instance apply` plus `--name`

The `module apply` subcommand SHALL accept the following flags with the listed behavior:

| Flag | Type | Default | Behavior |
| --- | --- | --- | --- |
| `-f`, `--values` | repeatable string | empty | Values files overriding the module's `debugValues` |
| `--platform` | string | resolved by precedence | Platform module directory override |
| `--name` | string | `<module>-debug` | Synthetic `metadata.name` override |
| `-n`, `--namespace` | string | from config | Target namespace (also propagates to synthetic `metadata.namespace`) |
| `--kubeconfig` | string | from env/config | Path to kubeconfig file |
| `--context` | string | current-context | Kubernetes context to use |
| `--dry-run` | bool | false | Server-side dry-run; no cluster changes |
| `--create-namespace` | bool | false | Create the target namespace if it does not exist |
| `--no-prune` | bool | false | Skip pruning of stale resources |
| `--force` | bool | false | Allow a 0-resource render to prune previously tracked resources |
| `--wait` | bool | false | After a successful apply and inventory write, block until every applied resource is healthy; ignored on `--dry-run` |
| `--timeout` | duration | 5m | One budget, starting when the apply starts, bounding the CustomResourceDefinition establish wait and the `--wait` readiness wait; also bounds the operator-reconcile wait for an operator-managed instance |

#### Scenario: Values files override debugValues

- **WHEN** the user runs `opm module apply ./my-module -f overrides.cue`
- **THEN** the subcommand SHALL use `overrides.cue` as the source of values
- **AND** SHALL NOT fall back to the module's `debugValues`

#### Scenario: Namespace flag participates in instance identity

- **WHEN** the user runs `opm module apply ./foo -n staging`
- **AND** the user later runs `opm module apply ./foo -n production`
- **THEN** the two invocations SHALL produce two distinct instance UUIDs
- **AND** SHALL write two independent inventory records in their respective namespaces

#### Scenario: Name flag participates in instance identity

- **WHEN** the user runs `opm module apply ./foo --name myapp`
- **AND** the user later runs `opm module apply ./foo` (no `--name`)
- **THEN** the two invocations SHALL produce two distinct instance UUIDs
- **AND** SHALL not interfere with each other's inventory

#### Scenario: Dry-run makes no cluster changes

- **WHEN** the user runs `opm module apply ./my-module --dry-run`
- **THEN** the subcommand SHALL perform a server-side dry-run apply
- **AND** SHALL NOT write or modify any inventory record
- **AND** SHALL NOT prune any resources
- **AND** SHALL log a summary of resources that would be applied

#### Scenario: Wait blocks until the applied resources are healthy

- **WHEN** the user runs `opm module apply ./my-module --wait` against a CLI-managed instance
- **THEN** after the apply and inventory write the subcommand SHALL poll every applied resource until each is healthy per `kubernetes.IsHealthy(kubernetes.EvaluateHealth(...))`
- **AND** SHALL print a success line and exit 0 once all are healthy
- **WHEN** `--timeout` elapses first, or an applied resource has disappeared
- **THEN** the subcommand SHALL exit non-zero with an error that lists the resources not yet healthy
- **AND** the applied resources and the inventory SHALL be left in place

#### Scenario: Wait is skipped on dry-run

- **WHEN** the user runs `opm module apply ./my-module --wait --dry-run`
- **THEN** the subcommand SHALL NOT poll the cluster for readiness

#### Scenario: Create-namespace auto-creates the target namespace

- **WHEN** the user runs `opm module apply ./my-module -n new-ns --create-namespace`
- **AND** namespace `new-ns` does not exist
- **THEN** the subcommand SHALL create the `new-ns` namespace before applying resources

#### Scenario: No-prune preserves stale resources

- **WHEN** a previous inventory recorded resources that are no longer rendered
- **AND** the user runs `opm module apply ./my-module --no-prune`
- **THEN** the stale resources SHALL be left in the cluster
- **AND** the inventory SHALL still be updated to reflect the new resource set

#### Scenario: Force allows 0-resource render to prune all

- **WHEN** a previous inventory has N>0 entries
- **AND** the user runs `opm module apply ./my-module` with a module that now renders 0 resources
- **AND** `--force` is NOT provided
- **THEN** the subcommand SHALL refuse to proceed and SHALL return an error explaining the situation
- **WHEN** the same conditions hold and `--force` IS provided
- **THEN** the subcommand SHALL prune all previously tracked resources
