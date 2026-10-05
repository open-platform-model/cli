## MODIFIED Requirements

### Requirement: opm platform pull writes the cluster's platform package to a directory

The CLI SHALL provide `opm platform pull <dir>`, registered under the `platform` group and cluster-facing (it accepts the kubeconfig and context flags the apply commands accept). It SHALL read the cluster Platform through the same resolution the render commands use, generate the platform module from the effective registry (or the spec when none is recorded), and write the module's files to `<dir>`, so that `opm module build --platform <dir>` and `opm instance build --platform <dir>` render against the package the cluster renders against. The written module SHALL be the generated module unchanged: reserved module path, pinned closure, one importing entry per catalog. The command SHALL NOT write to the cluster and SHALL NOT publish anything.

#### Scenario: A pulled package reproduces the cluster's render

- **WHEN** `opm platform pull ./cluster-platform` runs against a cluster whose Platform status records subscriptions and one registration-sourced catalog
- **THEN** `./cluster-platform` holds a platform module pinning every recorded catalog at its recorded version, and a local build against it resolves a provider contract exactly as the controller's render does

#### Scenario: The pulled module is the generated module

- **WHEN** the same CR is pulled twice
- **THEN** the two directories hold byte-identical files

### Requirement: pull reports what it reproduced

The command SHALL print the CR name and generation, the recorded package identity and the controller version from status (or that none is recorded), whether the effective registry or the spec was used, and every registry entry with its catalog, version, enabled flag and source. It SHALL print the stale-status and not-Ready warnings resolution raises. It SHALL end by naming the directory and the build command that consumes it.

#### Scenario: Provenance is printed

- **WHEN** the pull succeeds
- **THEN** the output names the generation, the package identity, the controller version, each entry's source, and the target directory
