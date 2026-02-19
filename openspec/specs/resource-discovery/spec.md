## Purpose

Defines how OPM CLI commands discover and select resources in a Kubernetes cluster. Primary discovery uses the inventory Secret (targeted GET calls per tracked resource). Label-based discovery via `DiscoverResources()` is retained for commands that still require it (e.g., delete fallback). This covers the `delete` and `status` commands that operate on existing deployed resources.

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

### Requirement: Namespace defaults to config

The `--namespace`/`-n` flag SHALL be optional for commands that discover resources (`delete`, `status`). When omitted, namespace SHALL be resolved using the precedence: `--namespace` flag → `OPM_NAMESPACE` environment variable → `kubernetes.namespace` in `~/.opm/config.cue` → `"default"`.

#### Scenario: Namespace omitted uses config default

- **WHEN** the user runs `opm mod delete --release-name my-app` without `-n`
- **AND** the config file sets `kubernetes: namespace: "staging"`
- **THEN** the command SHALL operate in the `staging` namespace

#### Scenario: Namespace omitted falls back to default

- **WHEN** the user runs `opm mod status --release-name my-app` without `-n`
- **AND** no config or env sets a namespace
- **THEN** the command SHALL operate in the `default` namespace

---

### Requirement: Status command supports --release-id

The `status` command MUST support `--release-id` flag with same semantics as `delete`.

#### Scenario: Status with --release-id

- **WHEN** user runs `opm mod status --release-id <uuid> --namespace bar`
- **THEN** status displays resources matching the release-id label selector

---
