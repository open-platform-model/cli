## MODIFIED Requirements

### Requirement: mod build --verbose shows per-resource validation lines

The `--verbose` output of `opm mod build` SHALL include per-resource validation lines in the "Generated Resources" section. Each resource SHALL be rendered using `FormatResourceLine` with `"valid"` status, matching the `r:<Kind/namespace/name>  <status>` format used by `mod apply`.

This replaces the current plain-text resource listing in verbose output.

#### Scenario: Verbose output renders resources with FormatResourceLine

- **WHEN** `opm mod build . --verbose` is run on a valid module that generates 3 resources
- **THEN** the "Generated Resources" section SHALL contain 3 lines
- **THEN** each line SHALL use `FormatResourceLine(kind, namespace, name, "valid")` format
- **THEN** the `r:` prefix SHALL be dim, resource path SHALL be cyan, and `"valid"` SHALL be green

#### Scenario: Verbose output aligns with mod apply resource output

- **WHEN** `opm mod build . --verbose` generates a `StatefulSet/default/jellyfin` resource
- **THEN** the verbose output SHALL render: `r:StatefulSet/default/jellyfin          valid`
- **THEN** the format SHALL be visually consistent with `mod apply`'s `r:StatefulSet/default/jellyfin  created`
