# core/project-structure — Delta

## MODIFIED Requirements

### Requirement: Template variants

The official template set SHALL be `minimal`, `standard`, and `advanced` (renaming `simple` to `minimal`), sourced as real published modules under `cli/templates/` rather than embedded text templates. `internal/templates/` and the embed machinery SHALL NOT exist. The variants' authoritative definition is the `template-modules` capability; this requirement retains only the set's names and their cli-repo home.

#### Scenario: No embedded templates remain

- **WHEN** the cli binary is built
- **THEN** it embeds no template file trees — templates reach users only as published modules

#### Scenario: Three variants published

- **WHEN** a cli release completes
- **THEN** `minimal`, `standard`, and `advanced` are resolvable in the reserved segment at their declared versions
