## MODIFIED Requirements

### Requirement: Status supports table output format

The default output format SHALL be a table showing: resource kind, name, namespace, component, health status, and age. The COMPONENT column SHALL be populated from the `component.opmodel.dev/name` label on each resource. Resources without this label SHALL display `-` in the COMPONENT column. Columns SHALL be aligned and human-readable.

The `--output`/`-o` flag SHALL accept `wide` as a valid value in addition to `table`, `yaml`, and `json`. When `-o wide` is specified, the command SHALL render a table format with additional columns.

#### Scenario: Default table output includes component column

- **WHEN** the user runs `opm mod status --release-name my-app -n production` without `--output`
- **THEN** the output SHALL be a formatted table with KIND, NAME, COMPONENT, STATUS, and AGE columns
- **AND** the COMPONENT column SHALL show the value of the `component.opmodel.dev/name` label for each resource

#### Scenario: Resource without component label

- **WHEN** a resource does not have the `component.opmodel.dev/name` label
- **THEN** the COMPONENT column SHALL display `-` for that resource

#### Scenario: Wide format accepted as output value

- **WHEN** the user runs `opm mod status --release-name my-app -n production -o wide`
- **THEN** the command SHALL render a table with additional columns beyond the default format

### Requirement: Status displays metadata header

The status output SHALL display a metadata header above the resource table containing: release name, module version, namespace, aggregate health status, and a resource summary. All metadata SHALL be sourced from labels already present on the discovered resources — the command MUST NOT require module source or re-rendering.

The header SHALL include:
- **Release**: from the `module-release.opmodel.dev/name` label
- **Version**: from the `module-release.opmodel.dev/version` label (omitted if not present)
- **Namespace**: from the resource metadata
- **Status**: the aggregate health status (Ready, NotReady, Unknown)
- **Resources**: total count with breakdown (e.g., "6 total (6 ready)" or "6 total (5 ready, 1 not ready)")

#### Scenario: Header shows release metadata

- **WHEN** the user runs `opm mod status --release-name jellyfin -n media`
- **AND** the discovered resources have the labels `module-release.opmodel.dev/name: jellyfin` and `module-release.opmodel.dev/version: 1.2.0`
- **THEN** the output SHALL begin with a header showing `Release: jellyfin`, `Version: 1.2.0`, `Namespace: media`, the aggregate status, and a resource count summary

#### Scenario: Header omits version when label is absent

- **WHEN** the discovered resources do not have the `module-release.opmodel.dev/version` label
- **THEN** the header SHALL omit the Version line entirely

#### Scenario: Header shows not ready count

- **WHEN** 2 out of 6 resources have a health status of NotReady
- **THEN** the Resources line SHALL display "6 total (4 ready, 2 not ready)"

### Requirement: Status output uses color

The status table and header SHALL use color-coded output when stdout is a TTY. Color SHALL be disabled when stdout is not a TTY or when the `NO_COLOR` environment variable is set.

The color mapping SHALL be:
- Health status `Ready` and `Complete`: green
- Health status `NotReady`: red
- Health status `Unknown`: yellow
- Component names: cyan
- Resource names: cyan
- Structural elements (borders, separators): dim gray

#### Scenario: Color output on TTY

- **WHEN** the user runs `opm mod status` with stdout connected to a TTY
- **AND** the `NO_COLOR` environment variable is not set
- **THEN** the STATUS column SHALL render `Ready` in green and `NotReady` in red
- **AND** the COMPONENT column SHALL render component names in cyan

#### Scenario: Color disabled on pipe

- **WHEN** the user pipes the output (e.g., `opm mod status ... | cat`)
- **THEN** all color/ANSI escape codes SHALL be stripped from the output

#### Scenario: Color disabled by NO_COLOR

- **WHEN** the `NO_COLOR` environment variable is set
- **THEN** all color/ANSI escape codes SHALL be stripped from the output
