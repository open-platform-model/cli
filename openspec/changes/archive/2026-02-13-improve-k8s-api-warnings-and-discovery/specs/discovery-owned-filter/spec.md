## ADDED Requirements

### Requirement: Discovery uses preferred API resources

The CLI SHALL use `ServerPreferredResources()` instead of `ServerGroupsAndResources()` when discovering API resources on the cluster.

#### Scenario: Only preferred version discovered for each resource type
- **WHEN** discovery enumerates API resources
- **THEN** only the preferred version of each resource type SHALL be queried
- **AND** deprecated API versions (like `v1/Endpoints`) SHALL NOT be enumerated when a preferred alternative exists

#### Scenario: Discovery handles unavailable API groups gracefully
- **WHEN** some API groups fail to respond during discovery
- **THEN** the CLI SHALL continue with available groups
- **AND** log a warning about the unavailable groups

### Requirement: ExcludeOwned option filters controller-managed resources

The CLI SHALL provide an `ExcludeOwned` option in `DiscoveryOptions` that excludes resources with `ownerReferences` from discovery results.

#### Scenario: Resources with ownerReferences excluded when option set
- **WHEN** discovery runs with `ExcludeOwned: true`
- **AND** a resource has non-empty `metadata.ownerReferences`
- **THEN** that resource SHALL NOT appear in discovery results

#### Scenario: Resources without ownerReferences included
- **WHEN** discovery runs with `ExcludeOwned: true`
- **AND** a resource has empty or missing `metadata.ownerReferences`
- **THEN** that resource SHALL appear in discovery results

#### Scenario: All resources included when option not set
- **WHEN** discovery runs with `ExcludeOwned: false` (or unset)
- **THEN** all resources matching the label selector SHALL appear in discovery results regardless of ownerReferences

### Requirement: Delete command excludes owned resources

The `opm mod delete` command SHALL discover resources with `ExcludeOwned: true` to prevent attempting to delete controller-managed children.

#### Scenario: Auto-managed Endpoints not included in delete
- **WHEN** a user runs `opm mod delete --name mymodule`
- **AND** the module created a Service that K8s auto-created Endpoints for
- **THEN** the Endpoints resource SHALL NOT appear in the delete list
- **AND** the delete SHALL complete without 404 errors for Endpoints

#### Scenario: Auto-managed EndpointSlice not included in delete
- **WHEN** a user runs `opm mod delete --name mymodule`
- **AND** the module created a Service that K8s auto-created EndpointSlice for
- **THEN** the EndpointSlice resource SHALL NOT appear in the delete list

#### Scenario: Parent resources deleted successfully
- **WHEN** a user runs `opm mod delete --name mymodule`
- **THEN** resources created by OPM (Service, Deployment, etc.) SHALL be deleted
- **AND** K8s garbage collection SHALL handle their children automatically

### Requirement: Diff command excludes owned resources

The `opm mod diff` command SHALL discover resources with `ExcludeOwned: true` to prevent showing diffs for controller-managed children.

#### Scenario: Auto-managed resources not included in diff
- **WHEN** a user runs `opm mod diff`
- **AND** the cluster has auto-managed Endpoints or EndpointSlice resources
- **THEN** those resources SHALL NOT appear in the diff output
