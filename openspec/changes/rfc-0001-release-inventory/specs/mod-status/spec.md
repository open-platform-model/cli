## MODIFIED Requirements

### Requirement: Status discovers resources via OPM labels

The `opm mod status` command SHALL first attempt to read the inventory Secret for the release. If an inventory exists, it SHALL use targeted GET calls for each tracked resource instead of scanning all API types. If no inventory exists, it SHALL fall back to discovering deployed resources by querying the cluster using the OPM label selector (`module.opmodel.dev/name`) within the target namespace. It MUST NOT require module source or re-rendering.

#### Scenario: Status shows deployed resources via inventory

- **WHEN** the user runs `opm mod status -n my-namespace --name my-module`
- **AND** an inventory Secret exists for the release
- **THEN** the command SHALL fetch each tracked resource via targeted GET

#### Scenario: Status falls back to label scan

- **WHEN** the user runs `opm mod status -n my-namespace --name my-module`
- **AND** no inventory Secret exists for the release
- **THEN** the command SHALL discover resources via label-scan
- **AND** a debug log message SHALL indicate "No inventory found, falling back to label-based discovery"

#### Scenario: No resources found

- **WHEN** no resources match the given name and namespace labels
- **THEN** the command SHALL print "No resources found for module <name> in namespace <namespace>" and exit with code 0

## ADDED Requirements

### Requirement: Status groups resources by component from inventory

When an inventory Secret is available, the `opm mod status` command SHALL group resources by the `component` field from inventory entries. This eliminates the need to read `component.opmodel.dev/name` labels from the cluster.

#### Scenario: Resources grouped by component

- **WHEN** the user runs `opm mod status` and an inventory exists
- **AND** the inventory tracks 3 resources under component `app` and 2 under component `cache`
- **THEN** the output SHALL group the resources by component name

#### Scenario: Missing resource shown in status

- **WHEN** the inventory tracks a resource that no longer exists on the cluster
- **THEN** the status output SHALL show the resource with status "Missing" or equivalent
