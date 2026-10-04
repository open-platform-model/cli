## MODIFIED Requirements

### Requirement: A dry run skips a custom resource whose CRD the same apply creates

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a custom resource whose group and kind are defined by a CustomResourceDefinition of the same apply that does not yet exist on the cluster. The command SHALL log a warning naming the skipped resource and its CustomResourceDefinition, SHALL count it as skipped in the dry-run summary, and SHALL NOT treat it as an error. A dry run SHALL NOT wait for any CustomResourceDefinition. A custom resource whose CustomResourceDefinition already exists on the cluster SHALL be sent as usual. A custom resource that is also in a namespace the same apply creates SHALL be warned about once, for its CustomResourceDefinition.

#### Scenario: A dry run with a new CRD skips its custom resource

- **WHEN** `opm instance apply --dry-run` runs for a module rendering CustomResourceDefinition `foos.example.com` and one `Foo`, and the cluster has no such CustomResourceDefinition
- **THEN** the `Foo` is not sent, a warning names `Foo` and `foos.example.com`, the summary reports one resource skipped, and the command exits 0

#### Scenario: A dry run with an existing CRD sends its custom resource

- **WHEN** the same dry run runs against a cluster where `foos.example.com` already exists
- **THEN** the `Foo` is sent to the server-side dry run and nothing is skipped

## ADDED Requirements

### Requirement: A dry run skips a namespaced object whose namespace the same apply creates

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a namespaced object whose namespace the apply would create: a `Namespace` (core group) the same apply renders and that does not exist on the cluster, or the instance namespace that `--create-namespace` would create. A dry run persists neither, so the server's namespace admission would refuse every object in them although the real apply would not. The command SHALL log a warning naming the skipped object and its namespace and saying that a dry run cannot validate it, SHALL count it as skipped in the dry-run summary, SHALL NOT treat it as an error, and SHALL exit 0 when nothing else fails. A namespaced object in a namespace that already exists, and every cluster-scoped object, SHALL be sent as usual. Without `--create-namespace`, a missing instance namespace that no rendered `Namespace` creates SHALL NOT cause a skip: the dry run reports the error the real apply would hit. Outside a dry run nothing is skipped.

#### Scenario: A dry run with a new Namespace skips the objects in it

- **WHEN** `opm instance apply --dry-run` runs for a module rendering `Namespace` `demo` and ConfigMap `cfg` in `demo`, and the cluster has no namespace `demo`
- **THEN** the Namespace is sent to the server-side dry run, `cfg` is not sent, a warning names `ConfigMap/cfg` and `demo`, the summary reports one resource skipped, no error is reported, and the command exits 0

#### Scenario: A dry run with --create-namespace skips the objects in the new instance namespace

- **WHEN** `opm instance apply --dry-run --create-namespace` runs for an instance in namespace `media`, the cluster has no namespace `media`, and the module renders no `Namespace` object
- **THEN** the command reports that namespace `media` would be created, skips every namespaced object in `media` with a warning naming `media`, counts them as skipped, and exits 0

#### Scenario: A dry run in an existing namespace sends everything

- **WHEN** the same dry run runs against a cluster where the namespace already exists
- **THEN** every object is sent to the server-side dry run and nothing is skipped for its namespace

#### Scenario: A real apply skips nothing

- **WHEN** `opm instance apply --create-namespace` runs without `--dry-run` for an instance whose namespace does not exist
- **THEN** the namespace is created and every object is applied
