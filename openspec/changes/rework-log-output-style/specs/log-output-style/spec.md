## ADDED Requirements

### Requirement: Centralized color palette for human log output

The `internal/output` package SHALL define a centralized set of named color constants using `lipgloss.Color` values. All human-readable styled output SHALL reference these constants rather than inline color codes.

The palette SHALL include:

- Cyan (`14`) for identifiable nouns (module paths, resource names, namespaces)
- Bright green (`82`) for `created` status
- Yellow (`220`) for `configured` status
- Red (`196`) for `deleted` status
- Bold red (`204`) for `failed` status
- Green (`10`) for completion checkmarks

#### Scenario: Color constants are defined and accessible

- **WHEN** a command needs to style a resource name
- **THEN** it SHALL use the named `ColorCyan` constant from the output package
- **THEN** it SHALL NOT use inline color codes like `lipgloss.Color("14")`

#### Scenario: All status colors follow the visibility hierarchy

- **WHEN** resource lines are rendered in a terminal
- **THEN** `created` (bright green) SHALL be more visually prominent than `configured` (yellow)
- **THEN** `configured` (yellow) SHALL be more visually prominent than `unchanged` (dim/faint)
- **THEN** the hierarchy SHALL be: `created` > `deleted` > `configured` > `failed` > `unchanged`

### Requirement: Semantic style constructors

The output package SHALL provide semantic style functions that map domain concepts to visual styles.

The following semantic styles SHALL be defined:

- **Noun style**: Cyan foreground for module paths, resource identifiers, and namespace values
- **Action style**: Bold for action verbs (`applying`, `installing`, `upgrading`, `deleting`)
- **Dim style**: Faint rendering for structural chrome (scope prefixes, separators)
- **Summary style**: Bold for completion and summary lines

A `StatusStyle(status string) lipgloss.Style` function SHALL return the appropriate style for a given resource status string.

#### Scenario: StatusStyle returns correct style for each status

- **WHEN** `StatusStyle("created")` is called
- **THEN** it SHALL return a style with bright green (`82`) foreground

- **WHEN** `StatusStyle("configured")` is called
- **THEN** it SHALL return a style with yellow (`220`) foreground

- **WHEN** `StatusStyle("unchanged")` is called
- **THEN** it SHALL return a style with faint rendering

- **WHEN** `StatusStyle("deleted")` is called
- **THEN** it SHALL return a style with red (`196`) foreground

- **WHEN** `StatusStyle("failed")` is called
- **THEN** it SHALL return a style with bold and red (`204`) foreground

#### Scenario: Unknown status falls back to default style

- **WHEN** `StatusStyle("unknown-value")` is called
- **THEN** it SHALL return the default unstyled `lipgloss.Style`

### Requirement: Resource line formatting with right-aligned status

The output package SHALL provide a `FormatResourceLine` function that renders a resource identifier with a right-aligned, color-coded status suffix.

The format SHALL be: `r:<Kind/namespace/name>  <status>` where:

- `r:` prefix is rendered in dim/faint
- The resource path (`Kind/namespace/name`) is rendered in cyan
- The status word is right-aligned to a consistent column and styled per `StatusStyle`

For cluster-scoped resources (no namespace), the format SHALL be: `r:<Kind/name>`.

#### Scenario: Namespaced resource with created status

- **WHEN** `FormatResourceLine("Deployment", "production", "my-app", "created")` is called
- **THEN** the output SHALL contain `r:` in dim style
- **THEN** the output SHALL contain `Deployment/production/my-app` in cyan
- **THEN** the output SHALL contain `created` in bright green, right-aligned

#### Scenario: Cluster-scoped resource with no namespace

- **WHEN** `FormatResourceLine("ClusterRole", "", "admin-view", "unchanged")` is called
- **THEN** the output SHALL render as `r:ClusterRole/admin-view` (no double slash)
- **THEN** the status SHALL be `unchanged` in dim/faint style

#### Scenario: Status words align across a batch of resources

- **WHEN** multiple resource lines are rendered in the same operation
- **THEN** the status words SHALL start at the same column position
- **THEN** the minimum alignment column SHALL be 48 characters from the start of the resource path

### Requirement: Scoped module logger

The output package SHALL provide a `ModuleLogger(name string) *log.Logger` function that returns a `charmbracelet/log.Logger` with the `m:<name> >` prefix.

The prefix SHALL be styled with:

- `m:` in dim/faint
- The module name in cyan
- `>` separator in dim/faint

The scoped logger SHALL inherit the current global logger's level, timestamp, and writer settings.

#### Scenario: Module logger produces scoped output

- **WHEN** `modLog := output.ModuleLogger("my-app")` is created
- **WHEN** `modLog.Info("resources are ready")` is called
- **THEN** the output line SHALL contain the `m:my-app >` prefix before the message
- **THEN** the prefix SHALL use the dim/cyan styling described above

#### Scenario: Module logger inherits global settings

- **WHEN** the global logger is configured with `DEBUG` level and timestamps on
- **WHEN** a module logger is created
- **THEN** the module logger SHALL also be at `DEBUG` level with timestamps on

### Requirement: Log level labels use 4-character abbreviations for human output

The human-readable log output SHALL use 4-character uppercase level labels as rendered by charmbracelet/log's default `MaxWidth(4)`: `DEBU`, `INFO`, `WARN`, `ERRO`, `FATA`.

JSON output SHALL continue to use full lowercase level names: `debug`, `info`, `warn`, `error`, `fatal`.

#### Scenario: Human output shows abbreviated level labels

- **WHEN** an info-level message is logged in human output mode
- **THEN** the level label SHALL render as `INFO` (4 characters, bold, teal-green)

#### Scenario: Error level renders as ERRO in human output

- **WHEN** an error-level message is logged in human output mode
- **THEN** the level label SHALL render as `ERRO` (4 characters, bold, pink-red)

#### Scenario: JSON output uses full level names

- **WHEN** an error-level message is logged in JSON output mode
- **THEN** the level field SHALL be `"error"` (full lowercase word)

### Requirement: Human log output preserves two-channel architecture

All styled log messages SHALL be written to `os.Stderr`. Data output (manifests, tables, file trees, diffs, completion checkmarks) SHALL be written to `os.Stdout`.

#### Scenario: Log messages go to stderr

- **WHEN** `modLog.Info(output.FormatResourceLine(...))` is called
- **THEN** the output SHALL be written to `os.Stderr`

#### Scenario: Completion line goes to stdout

- **WHEN** a command prints `✔ Module applied`
- **THEN** the output SHALL be written to `os.Stdout` via `output.Println`

#### Scenario: Piping stdout captures only data

- **WHEN** the user runs `opm mod apply > output.txt`
- **THEN** `output.txt` SHALL contain only data output (completion messages)
- **THEN** styled log lines SHALL appear in the terminal via stderr
