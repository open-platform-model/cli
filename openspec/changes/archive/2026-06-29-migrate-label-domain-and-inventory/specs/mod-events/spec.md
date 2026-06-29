## MODIFIED Requirements

<!-- enhancement 0002 D6/D10 — X3-deferred to X4 per D-X3.6 (single capability owner). Restates the requirement whose normative text changes under the rename: the shared selector-flags bundle (InstanceSelectorFlags, --instance-name/--instance-id). Per-scenario flag-string swaps in the other event requirements ride the archive spec-sync / hygiene pass. -->

### Requirement: Events command uses shared instance selector flags

The command SHALL use `InstanceSelectorFlags` for `--instance-name`/`--instance-id`/`-n` with the same mutual exclusivity validation as `mod status` and `mod delete`. Exactly one of `--instance-name` or `--instance-id` MUST be provided. <!-- Was: ReleaseSelectorFlags, --release-name/--release-id (0002 D10/D-X4.2) -->

#### Scenario: Both selectors provided

- **WHEN** the user provides both `--instance-name` and `--instance-id`
- **THEN** the command SHALL exit with error: `"--instance-name and --instance-id are mutually exclusive"`

#### Scenario: Neither selector provided

- **WHEN** the user provides neither `--instance-name` nor `--instance-id`
- **THEN** the command SHALL exit with error: `"either --instance-name or --instance-id is required"`

### Requirement: Events command discovers instance resources and their children

The `opm mod events` command SHALL discover OPM-managed resources via `cmdutil.ResolveInventory` (inventory-based discovery), then walk ownerReferences downward to find Kubernetes-owned children (ReplicaSets, Pods) of workload resources. The combined set of resource UIDs SHALL be used to filter events. <!-- Was: "discovers release resources" (0002 D9) -->

#### Scenario: Events include Pod-level events from Deployment children

- **WHEN** the user runs `opm mod events --instance-name my-app -n production`
- **AND** the instance contains a Deployment `my-app-web` which owns ReplicaSet `my-app-web-abc12` which owns Pod `my-app-web-abc12-x1`
- **THEN** the output SHALL include events for the Deployment, the ReplicaSet, and the Pod

#### Scenario: No OPM resources found

- **WHEN** no resources match the instance selector
- **THEN** the command SHALL exit with a non-zero exit code and display an error message
