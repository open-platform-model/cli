# Delta: resource-discovery (cli-cr-inventory-backend)

Selector resolution reads the `ModuleInstance` CR instead of Secret-era label selectors. Namespace precedence, `--instance-id` support on `status`, and child traversal are unchanged.

## MODIFIED Requirements

### Requirement: Selector mutual exclusivity

Commands that discover resources (`delete`, `status`) MUST accept exactly one selector type per invocation. Name selectors resolve by a direct `ModuleInstance` GET; instance-id selectors resolve by listing `ModuleInstance` CRs in the namespace and matching `status.instanceUUID`.

#### Scenario: Both --name and --instance-id provided

- **WHEN** user provides both `--name` and `--instance-id` flags
- **THEN** command exits with error: `"--name and --instance-id are mutually exclusive"`

#### Scenario: Neither --name nor --instance-id provided

- **WHEN** user provides neither `--name` nor `--instance-id` flag
- **THEN** command exits with error: `"either --name or --instance-id is required"`

#### Scenario: Only --name provided

- **WHEN** user provides `--name` flag (and `--namespace`)
- **THEN** command resolves the instance by a direct `ModuleInstance` GET by name in the namespace

#### Scenario: Only --instance-id provided

- **WHEN** user provides `--instance-id` flag (and `--namespace`)
- **THEN** command resolves the instance by listing `ModuleInstance` CRs in the namespace and matching `status.instanceUUID`
