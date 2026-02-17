## MODIFIED Requirements

### Requirement: Diff renders locally and compares against cluster

The `opm mod diff` command SHALL call `Pipeline.Render()` to produce the desired resource set, then fetch the live state of each resource from the cluster. Before comparison, the command SHALL project the live object to only contain field paths present in the rendered object, stripping server-managed metadata and API-server defaults. The projected objects SHALL then be compared using semantic YAML diff.

#### Scenario: Module with deployed changes shows diff

- **WHEN** a module has been previously applied and local definitions have changed
- **THEN** `opm mod diff` SHALL display a colorized semantic diff for each modified resource, showing only changes to fields defined in the rendered manifest

#### Scenario: Module not yet deployed shows all as additions

- **WHEN** a module has never been applied to the cluster
- **THEN** `opm mod diff` SHALL show every rendered resource as an addition
