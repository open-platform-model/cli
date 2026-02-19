
## Requirements

### Requirement: Diff renders locally and compares against cluster

The `opm mod diff` command SHALL call `Pipeline.Render()` to produce the desired resource set, then fetch the live state of each resource from the cluster. Before comparison, the command SHALL project the live object to only contain field paths present in the rendered object, stripping server-managed metadata and API-server defaults. The projected objects SHALL then be compared using semantic YAML diff.

#### Scenario: Module with deployed changes shows diff

- **WHEN** a module has been previously applied and local definitions have changed
- **THEN** `opm mod diff` SHALL display a colorized semantic diff for each modified resource, showing only changes to fields defined in the rendered manifest

#### Scenario: Module not yet deployed shows all as additions

- **WHEN** a module has never been applied to the cluster
- **THEN** `opm mod diff` SHALL show every rendered resource as an addition

### Requirement: Diff uses semantic YAML comparison via dyff

The diff output SHALL use `homeport/dyff` for semantic YAML comparison. Field reordering, whitespace differences, and other non-semantic changes MUST NOT appear as modifications.

#### Scenario: Field reordering does not produce diff

- **WHEN** a resource has identical content but fields are in different order locally vs cluster
- **THEN** `opm mod diff` SHALL report no differences for that resource

### Requirement: Diff categorizes resources into three states

The command SHALL categorize each resource into one of three states: modified (exists both locally and on cluster with differences), added (exists locally but not on cluster), or orphaned (exists on cluster per inventory but not in local render). When an inventory Secret exists, orphan detection SHALL use inventory set-difference: entries in the previous inventory not present in the current render, verified with targeted GETs. When no inventory exists, orphan detection SHALL return an empty set — all rendered resources SHALL be shown as additions. The command MUST NOT fall back to a cluster-wide label-scan at any point.

#### Scenario: Module not yet deployed shows all as additions

- **WHEN** a module has never been applied to the cluster
- **AND** therefore no inventory Secret exists
- **THEN** `opm mod diff` SHALL show every rendered resource as an addition (`[new resource]`)
- **AND** no orphans SHALL be reported

#### Scenario: Resource exists on cluster but not in local render

- **WHEN** a resource was previously applied (has OPM labels) but is no longer produced by the local render
- **THEN** `opm mod diff` SHALL display the resource as orphaned with a message indicating it will be removed on next apply

#### Scenario: New resource in local render

- **WHEN** a resource is produced by the local render but does not exist on the cluster
- **THEN** `opm mod diff` SHALL display the resource as a new addition

#### Scenario: Orphan detection with inventory

- **WHEN** an inventory Secret exists for the release
- **THEN** orphans SHALL be computed as inventory entries not present in the current rendered set
- **AND** each orphan candidate SHALL be verified via a targeted GET (missing resources on cluster are excluded from orphan list)

### Requirement: Diff displays a summary line

The command SHALL print a summary line showing the count of modified, added, and orphaned resources before the detailed output.

#### Scenario: Summary with mixed changes

- **WHEN** the diff contains 2 modified, 1 added, and 1 orphaned resource
- **THEN** the summary line SHALL read "2 modified, 1 added, 1 orphaned"

#### Scenario: No differences

- **WHEN** local render matches live cluster state exactly
- **THEN** the command SHALL print "No differences found" and exit with code 0

### Requirement: Diff supports partial render results

The command MAY process a partial `RenderResult` when some resources failed to render. Successfully rendered resources SHALL be compared; failed resources SHALL produce a warning in the output.

#### Scenario: Partial render with warnings

- **WHEN** the render pipeline produces 5 resources and 1 error
- **THEN** `opm mod diff` SHALL compare the 5 successful resources and print a warning about the 1 failed resource

### Requirement: Diff accepts standard module flags

The command SHALL accept `--values`/`-f` (repeatable), `--namespace`/`-n`, `--name`, `--kubeconfig`, and `--context` flags. The `path` positional argument SHALL default to the current directory.

#### Scenario: Values file override

- **WHEN** the user runs `opm mod diff -f custom-values.yaml`
- **THEN** the render pipeline SHALL merge the custom values before comparing

#### Scenario: Explicit kubeconfig and context

- **WHEN** the user runs `opm mod diff --kubeconfig ~/.kube/staging --context staging-cluster`
- **THEN** the command SHALL connect to the specified cluster context for live state comparison

### Requirement: Diff fails fast on connectivity errors

The command SHALL fail immediately with a clear error message if the Kubernetes cluster is unreachable, rather than timing out silently.

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with code 3 and display a connectivity error message
