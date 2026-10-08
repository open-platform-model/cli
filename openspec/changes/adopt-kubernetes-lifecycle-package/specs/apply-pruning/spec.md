## ADDED Requirements

### Requirement: Prune follows the shared deletion plan

The prune of `opm instance apply` and `opm module apply` SHALL take every action on a stale resource from the deletion plan the CLI shares with the operator, built from the stale entries the prune may delete and the instance identity stored in the instance's record. That set SHALL NOT hold a `Namespace`, a `CustomResourceDefinition`, or a `PersistentVolumeClaim` that is kept. The prune SHALL read a stale resource, delete it or leave it in place only when the plan names that action as the next one, in descending kind-weight order, and it SHALL send each delete with the propagation policy and the precondition the plan names. A resource whose read or delete fails SHALL NOT stop the plan, except that after a failed API discovery request the prune SHALL send no further request and SHALL report that resource and every resource not yet tried as failed with the discovery error. The prune SHALL NOT wait for a deleted resource to disappear. Which entries stay in the written inventory, the output lines and the exit codes SHALL stay as the other requirements of this capability state them. The dry-run prune preview SHALL NOT be affected. Source: 0012:D4:R1, 0012:D4:R6.

#### Scenario: Stale resources are deleted in the plan's order

- **WHEN** the stale set holds a ConfigMap and a Deployment
- **THEN** the delete request for the Deployment SHALL be sent before the one for the ConfigMap

#### Scenario: Delete carries what the plan names

- **WHEN** the plan names the delete of a stale ConfigMap the instance owns
- **THEN** the delete request SHALL carry Foreground propagation and a precondition on the UID of the object that was read

#### Scenario: One failed delete does not stop the others

- **WHEN** the stale set holds ConfigMaps `a` and `b` and the delete of `a` fails with Forbidden
- **THEN** ConfigMap `b` SHALL still be read and deleted
- **AND** the written inventory SHALL hold the entry of `a` and SHALL NOT hold the entry of `b`

#### Scenario: Failed discovery request stops the requests

- **WHEN** the stale set holds two resources and the API discovery request for the first one's group and version fails with Forbidden
- **THEN** no read and no delete SHALL be sent for the second resource
- **AND** both entries SHALL stay in the written inventory
- **AND** the command SHALL exit 4

#### Scenario: Stale resource held by a finalizer

- **WHEN** the delete of a stale ConfigMap is accepted and a finalizer keeps the ConfigMap from disappearing
- **THEN** the prune SHALL NOT wait for it
- **AND** the written inventory SHALL NOT hold its entry
- **AND** the command SHALL exit 0
