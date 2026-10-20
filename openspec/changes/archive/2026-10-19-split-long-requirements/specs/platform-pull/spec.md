## REMOVED Requirements

### Requirement: opm platform pull writes the cluster's platform package to a directory
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm platform pull <dir>` is a cluster-facing command of the `platform` group", "`opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`", "The module that `opm platform pull` writes is the generated module unchanged", "`opm platform pull` writes nothing to the cluster and publishes nothing".

## ADDED Requirements

### Requirement: `opm platform pull <dir>` is a cluster-facing command of the `platform` group

The CLI SHALL provide `opm platform pull <dir>`, registered under the `platform` group and cluster-facing (it accepts the kubeconfig and context flags the apply commands accept).

#### Scenario: The pull command accepts the cluster flags

- **WHEN** `opm platform pull <dir>` runs
- **THEN** the command SHALL be registered under the `platform` group and SHALL accept the kubeconfig and context flags the apply commands accept

### Requirement: `opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`

`opm platform pull <dir>` SHALL read the cluster Platform through the same resolution the render commands use, generate the platform module from the effective registry (or the spec when none is recorded), and write the module's files to `<dir>`, so that `opm module build --platform <dir>` and `opm instance build --platform <dir>` render against the package the cluster renders against.

#### Scenario: A pulled package reproduces the cluster's render

- **WHEN** `opm platform pull ./cluster-platform` runs against a cluster whose Platform status records subscriptions and one registration-sourced catalog
- **THEN** `./cluster-platform` holds a platform module pinning every recorded catalog at its recorded version, and a local build against it resolves a provider contract exactly as the operator's render does

### Requirement: The module that `opm platform pull` writes is the generated module unchanged

The written module SHALL be the generated module unchanged: reserved module path, pinned closure, one importing entry per catalog.
(Requirement: `opm platform pull <dir>` writes the platform module that the cluster renders against to `<dir>`)

#### Scenario: The pulled module is the generated module

- **WHEN** the same CR is pulled twice
- **THEN** the two directories hold byte-identical files

### Requirement: `opm platform pull` writes nothing to the cluster and publishes nothing

`opm platform pull <dir>` SHALL NOT write to the cluster and SHALL NOT publish anything.

#### Scenario: The pull command writes nothing to the cluster

- **WHEN** `opm platform pull <dir>` runs
- **THEN** it SHALL NOT write to the cluster and SHALL NOT publish anything
