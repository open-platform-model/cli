## Purpose

Define `opm platform pull`: the command that reproduces a cluster's platform package on the laptop from the Platform CR's effective inputs, so a local build renders what the cluster renders (enhancement 0015 D6).

## ADDED Requirements

### Requirement: opm platform pull writes the cluster's platform package to a directory

The CLI SHALL provide `opm platform pull <dir>`, registered under the `platform` group and cluster-facing (it accepts the kubeconfig and context flags the apply commands accept). It SHALL read the cluster Platform through the same resolution the render commands use, generate the platform module from the effective registry (or the spec when none is recorded), and write the module's files to `<dir>`, so that `opm module build --platform <dir>` and `opm instance build --platform <dir>` render against the package the cluster renders against. The written module SHALL be the generated module unchanged: reserved module path, pinned closure, one importing entry per catalog. The command SHALL NOT write to the cluster and SHALL NOT publish anything.

#### Scenario: A pulled package reproduces the cluster's render

- **WHEN** `opm platform pull ./cluster-platform` runs against a cluster whose Platform status records subscriptions and one registration-sourced catalog
- **THEN** `./cluster-platform` holds a platform module pinning every recorded catalog at its recorded version, and a local build against it resolves a provider contract exactly as the operator's render does

#### Scenario: The pulled module is the generated module

- **WHEN** the same CR is pulled twice
- **THEN** the two directories hold byte-identical files

### Requirement: pull reports what it reproduced

The command SHALL print the CR name and generation, the recorded package identity and the operator version from status (or that none is recorded), whether the effective registry or the spec was used, and every registry entry with its catalog, version, enabled flag and source. It SHALL print the stale-status and not-Ready warnings resolution raises. It SHALL end by naming the directory and the build command that consumes it.

#### Scenario: Provenance is printed

- **WHEN** the pull succeeds
- **THEN** the output names the generation, the package identity, the operator version, each entry's source, and the target directory

### Requirement: pull refuses to overwrite by accident

If `<dir>` exists and is not empty, the command SHALL refuse with the general error code unless `--force` is given, in which case it SHALL replace the directory's contents with the generated module. `--force` SHALL default to false.

#### Scenario: A non-empty target is refused

- **WHEN** `<dir>` holds files and `--force` is absent
- **THEN** the command fails naming the directory and the flag, and writes nothing

### Requirement: pull has no local fallback

A cluster without a readable Platform CR SHALL fail the command with the not-found error code naming the CR: there is no cluster package to reproduce, and the local default platform is not it. A Platform whose recorded registry cannot be generated (an unpublished pin, a legacy spec without versions) SHALL fail with the validation error code and the generation diagnostic. A cluster that cannot be reached SHALL fail with the connectivity error code.

#### Scenario: No Platform CR

- **WHEN** the cluster has no Platform named `cluster`
- **THEN** the command exits with the not-found code and the message names the CR and says there is no cluster package to pull
