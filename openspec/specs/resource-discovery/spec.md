## Purpose

Defines how OPM CLI commands discover and select resources in a Kubernetes cluster using label selectors. This covers the `delete` and `status` commands that operate on existing deployed resources.

## Requirements

### Requirement: Selector mutual exclusivity

Commands that discover resources (`delete`, `status`) MUST accept exactly one selector type per invocation.

#### Scenario: Both --name and --release-id provided

- **WHEN** user provides both `--name` and `--release-id` flags
- **THEN** command exits with error: `"--name and --release-id are mutually exclusive"`

#### Scenario: Neither --name nor --release-id provided

- **WHEN** user provides neither `--name` nor `--release-id` flag
- **THEN** command exits with error: `"either --name or --release-id is required"`

#### Scenario: Only --name provided

- **WHEN** user provides `--name` flag (and `--namespace`)
- **THEN** command uses name+namespace label selector

#### Scenario: Only --release-id provided

- **WHEN** user provides `--release-id` flag (and `--namespace`)
- **THEN** command uses release-id label selector

---

### Requirement: Namespace always required

The `--namespace` flag MUST be required regardless of selector type.

#### Scenario: Missing namespace with --name

- **WHEN** user provides `--name` without `--namespace`
- **THEN** command exits with error indicating namespace is required

#### Scenario: Missing namespace with --release-id

- **WHEN** user provides `--release-id` without `--namespace`
- **THEN** command exits with error indicating namespace is required

---

### Requirement: Fail on no resources found

Commands MUST fail with non-zero exit code when no resources match the selector.

#### Scenario: No resources match --name selector

- **WHEN** user runs `delete` or `status` with `--name foo --namespace bar`
- **AND** no resources have labels `module.opmodel.dev/name=foo` and `module.opmodel.dev/namespace=bar`
- **THEN** command exits with error: `"no resources found for module \"foo\" in namespace \"bar\""`

#### Scenario: No resources match --release-id selector

- **WHEN** user runs `delete` or `status` with `--release-id <uuid> --namespace bar`
- **AND** no resources have label `module.opmodel.dev/release-id=<uuid>`
- **THEN** command exits with error: `"no resources found for release-id \"<uuid>\" in namespace \"bar\""`

---

### Requirement: Label selector construction

Each selector type MUST query with specific labels.

#### Scenario: Name selector labels

- **WHEN** using `--name` selector
- **THEN** query includes labels:
  - `app.kubernetes.io/managed-by=open-platform-model`
  - `module.opmodel.dev/name=<name>`
  - `module.opmodel.dev/namespace=<namespace>`

#### Scenario: Release-id selector labels

- **WHEN** using `--release-id` selector
- **THEN** query includes labels:
  - `app.kubernetes.io/managed-by=open-platform-model`
  - `module.opmodel.dev/release-id=<uuid>`

---

### Requirement: Status command supports --release-id

The `status` command MUST support `--release-id` flag with same semantics as `delete`.

#### Scenario: Status with --release-id

- **WHEN** user runs `opm mod status --release-id <uuid> --namespace bar`
- **THEN** status displays resources matching the release-id label selector
