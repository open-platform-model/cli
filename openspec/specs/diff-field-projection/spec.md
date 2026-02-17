
## Purpose

Field projection filtering for `opm mod diff` that eliminates server-managed noise from diff output by projecting live Kubernetes objects to only the field paths present in rendered manifests.

## Requirements

### Requirement: Field projection strips server-managed metadata from live objects

Before comparing rendered and live objects, the diff system SHALL remove the following server-managed fields from the live object: `metadata.managedFields`, `metadata.uid`, `metadata.resourceVersion`, `metadata.creationTimestamp`, `metadata.generation`, and the top-level `status` block. These fields are never present in rendered output and MUST NOT appear in diff results.

#### Scenario: Apply then diff with no changes shows no differences

- **WHEN** a module is applied with `opm mod apply` and then immediately diffed with `opm mod diff` using identical values
- **THEN** the diff SHALL report "No differences found" for all resources

#### Scenario: Server-managed metadata does not appear in diff

- **WHEN** a live object contains `metadata.managedFields`, `metadata.uid`, `metadata.resourceVersion`, `metadata.creationTimestamp`, and `metadata.generation`
- **THEN** none of these fields SHALL appear in the diff output

#### Scenario: Status block does not appear in diff

- **WHEN** a live object contains a `status` block populated by a controller
- **THEN** the `status` block SHALL NOT appear in the diff output

### Requirement: Field projection retains only rendered field paths in live objects

After stripping server-managed metadata, the diff system SHALL project the live object to only contain field paths that exist in the rendered object. Any field present in the live object but absent from the rendered object (such as API-server defaults) SHALL be excluded from comparison.

#### Scenario: API-server defaults do not appear in diff

- **WHEN** a live StatefulSet contains server-defaulted fields such as `spec.podManagementPolicy`, `spec.revisionHistoryLimit`, and `spec.template.spec.dnsPolicy` that are not present in the rendered manifest
- **THEN** these defaulted fields SHALL NOT appear in the diff output

#### Scenario: Server-injected annotations do not appear in diff

- **WHEN** a live PersistentVolumeClaim contains annotations injected by the scheduler or controller-manager (e.g., `volume.kubernetes.io/selected-node`) that are not in the rendered manifest
- **THEN** these annotations SHALL NOT appear in the diff output

#### Scenario: Actual value changes still appear in diff

- **WHEN** a rendered resource has `spec.replicas: 3` but the live object has `spec.replicas: 1`
- **THEN** the diff SHALL show the replicas change

### Requirement: Field projection matches list elements by name

For lists of maps (e.g., containers, volumes, env vars), the projection SHALL match elements between rendered and live lists using the `name` field as the associative key. If no `name` field is present, elements SHALL be matched by index position.

#### Scenario: Container fields matched by name

- **WHEN** a rendered StatefulSet defines a container named "minecraft" with `image: itzg/minecraft-server:latest`
- **AND** the live object has the same container with a different image
- **THEN** the diff SHALL show the image change for the "minecraft" container

#### Scenario: List element with no name field uses index matching

- **WHEN** a rendered Service has a `spec.ports` list with entries that have no `name` field
- **AND** the live object has the same ports with additional server-defaulted fields per port entry
- **THEN** the projection SHALL match ports by index position and strip server-defaulted fields from each matched entry

### Requirement: Field projection removes empty maps after filtering

After projection, any map that becomes empty (all keys were server-managed and stripped) SHALL be removed from the live object to prevent spurious diffs caused by empty map vs absent key differences.

#### Scenario: Empty annotations map after filtering

- **WHEN** a live object has `metadata.annotations` containing only server-injected keys
- **AND** the rendered object has `metadata.annotations: {}`
- **THEN** the diff SHALL NOT show a difference for the annotations field
