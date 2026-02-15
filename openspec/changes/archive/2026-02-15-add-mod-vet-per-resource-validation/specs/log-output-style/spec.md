## ADDED Requirements

### Requirement: StatusValid constant for validated resources

The output package SHALL define a `StatusValid` constant with value `"valid"` for resources that passed validation. The `StatusStyle` function SHALL return a green (`82`) foreground style for `"valid"`, matching the color used for `"created"`.

#### Scenario: StatusStyle returns green for valid status

- **WHEN** `StatusStyle("valid")` is called
- **THEN** it SHALL return a style with bright green (`82`) foreground
- **THEN** the style SHALL be identical to the style returned for `"created"`

#### Scenario: FormatResourceLine renders valid status in green

- **WHEN** `FormatResourceLine("StatefulSet", "default", "jellyfin", "valid")` is called
- **THEN** the output SHALL contain `valid` in bright green, right-aligned to the minimum column

### Requirement: Validation check line formatting

The output package SHALL provide a `FormatVetCheck(label, detail string)` function that renders a validation check result with a green checkmark, a label, and an optional right-aligned detail string.

The format SHALL be: `✔ <label>` followed by right-aligned dim detail text when provided.

The checkmark SHALL use the existing `colorGreenCheck` (`10`) constant. The detail text SHALL be rendered in dim/faint style, consistent with structural chrome in `FormatResourceLine`.

The alignment column for detail text SHALL be 34 characters from the start of the label, ensuring consistent alignment across multiple check lines.

When `detail` is empty, the function SHALL render only the checkmark and label with no trailing whitespace.

#### Scenario: Check line with detail renders aligned output

- **WHEN** `FormatVetCheck("Config file found", "~/.opm/config.cue")` is called
- **THEN** the output SHALL contain `✔` in green (color `10`)
- **THEN** the output SHALL contain `Config file found` in default style
- **THEN** the output SHALL contain `~/.opm/config.cue` in dim/faint style, right-aligned to column 34

#### Scenario: Check line without detail renders cleanly

- **WHEN** `FormatVetCheck("CUE evaluation passed", "")` is called
- **THEN** the output SHALL contain `✔` in green followed by `CUE evaluation passed`
- **THEN** the output SHALL NOT contain trailing whitespace or padding

#### Scenario: Multiple check lines align consistently

- **WHEN** multiple `FormatVetCheck` calls are rendered sequentially
- **THEN** all detail strings SHALL start at the same column position
- **THEN** the visual alignment SHALL match the pattern used by `FormatResourceLine` for status words

## MODIFIED Requirements

### Requirement: Centralized color palette for human log output

The `internal/output` package SHALL define a centralized set of named color constants using `lipgloss.Color` values. All human-readable styled output SHALL reference these constants rather than inline color codes.

The palette SHALL include:

- Cyan (`14`) for identifiable nouns (module paths, resource names, namespaces)
- Bright green (`82`) for `created` and `valid` status
- Yellow (`220`) for `configured` status and notice arrows
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
- **THEN** the hierarchy SHALL be: `created` > `valid` > `deleted` > `configured` > `failed` > `unchanged`

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

- **WHEN** `StatusStyle("valid")` is called
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
