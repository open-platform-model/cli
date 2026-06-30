## MODIFIED Requirements

<!-- enhancement 0002 D4/D9/D10. Restates only the deploy requirements whose normative text changes under the rename: the identity-label key (FR-D-065), the persisted "instance inventory record" noun, and the `--instance-id`/`--instance-name` selector flags on `mod delete`. The bulk of the FR-D-0xx render/apply/prune behavior is unchanged and rides the archive spec-sync. -->

### Requirement: Legacy CLI-managed instances remain mutable by the CLI

`opm mod apply` and `opm mod delete` SHALL continue to manage instances created by an earlier CLI. Identity-label injection SHALL use the instance label domain. <!-- Was: "Legacy CLI-managed releases", module-release.opmodel.dev/uuid (0002 D4) -->

| Requirement | Detail |
| --- | --- |
| FR-D-065 | All resources MUST have `module-instance.opmodel.dev/uuid: <instance-uuid>` when the instance identity is available. <!-- Was: module-release.opmodel.dev/uuid --> |

#### Scenario: Applied resources carry the instance UUID label

- **WHEN** `opm mod apply` applies resources and the instance identity is available
- **THEN** every applied resource SHALL carry `module-instance.opmodel.dev/uuid: <instance-uuid>`

### Requirement: mod delete selector flags use the instance domain

`opm mod delete` SHALL require at least one of `--instance-name` or `--instance-id` for identification; the `--namespace`/`-n` flag remains required in all cases. `--instance-id` SHALL support discovery by instance identity UUID. <!-- Was: --name/--release-id (0002 D-X4.2) -->

#### Scenario: Delete requires a selector

- **WHEN** `opm mod delete -n production` is run with neither `--instance-name` nor `--instance-id`
- **THEN** the command SHALL exit with an error requiring one of the selectors

#### Scenario: Delete by instance ID

- **WHEN** `opm mod delete --instance-id <uuid> -n production` is run
- **THEN** the command SHALL discover resources by instance identity UUID via the persisted instance inventory record

### Requirement: mod apply writes current instance inventory record after successful apply

After all resources are successfully applied, `opm mod apply` SHALL persist the current instance inventory record for the instance. The persisted form SHALL store top-level `createdBy`, `instanceMetadata`, `moduleMetadata`, and the current owned resource set directly instead of a history-bearing inventory shape. <!-- Was: "release inventory record", releaseMetadata (0002 D8/D9) -->

#### Scenario: Successful apply persists current instance inventory record

- **WHEN** `opm mod apply ./my-module` succeeds against a cluster
- **THEN** the persisted instance inventory record SHALL record the current owned resource entries for that instance
