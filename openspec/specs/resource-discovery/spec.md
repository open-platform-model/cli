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


### Requirement: Label selector construction

Each selector type MUST query with specific labels.

#### Scenario: Name selector labels

- **WHEN** using `--name` selector
- **THEN** query includes labels:
  - `app.kubernetes.io/managed-by=open-platform-model`
  - `module.opmodel.dev/name=<name>`
- **AND** namespace scoping is handled by the Kubernetes API (namespaced list calls)

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

---

### Requirement: Inventory Secret excluded from workload discovery

`DiscoverResources()` SHALL exclude Secrets that carry the label `opmodel.dev/component: inventory` from its results. This prevents the inventory Secret from appearing in workload resource queries used by diff, status, and the label-based delete fallback.

#### Scenario: Inventory Secret not returned by DiscoverResources

- **WHEN** `DiscoverResources()` scans a namespace containing an inventory Secret with label `opmodel.dev/component: inventory`
- **AND** the inventory Secret also has `app.kubernetes.io/managed-by: open-platform-model`
- **THEN** the inventory Secret SHALL NOT appear in the returned resource list

#### Scenario: Non-inventory Secrets still returned

- **WHEN** `DiscoverResources()` scans a namespace containing a regular Secret with OPM labels but without `opmodel.dev/component: inventory`
- **THEN** the Secret SHALL appear in the returned resource list

---

### Requirement: Inventory-based resource discovery

A new `DiscoverResourcesFromInventory` function SHALL accept an `InventorySecret` and perform targeted GET calls for each tracked resource. It SHALL return both the live resources found and a list of inventory entries for resources that no longer exist on the cluster (missing). This provides fast, precise discovery (N API calls for N resources) compared to the label-scan approach (hundreds of API calls).

#### Scenario: All tracked resources exist

- **WHEN** the inventory contains 5 entries and all 5 resources exist on the cluster
- **THEN** 5 live resources SHALL be returned and the missing list SHALL be empty

#### Scenario: Some tracked resources missing

- **WHEN** the inventory contains 5 entries and 2 resources have been manually deleted
- **THEN** 3 live resources SHALL be returned and 2 entries SHALL be in the missing list

#### Scenario: Empty inventory

- **WHEN** the inventory contains 0 entries
- **THEN** both the live list and missing list SHALL be empty
