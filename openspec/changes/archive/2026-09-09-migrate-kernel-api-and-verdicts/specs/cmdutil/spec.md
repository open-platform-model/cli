## MODIFIED Requirements

### Requirement: ShowRenderOutput checks for errors, shows transformer matches, and logs warnings

The render output function (`render.ShowOutput`) SHALL accept a render result and output options (verbose flag). It SHALL:

1. Check for render errors — if present, format and print render errors via `printValidationError`, then return an `*ExitError` with `ExitValidationError`.
2. Show transformer match output — verbose mode shows module metadata and match reasons; default mode shows compact match lines.
3. Log any warnings from the render result via the module-scoped logger. The warnings are worded by the CLI from the kernel's advisory diagnostic rows (unhandled optional traits, and resolved-versions rows marked newer); the render result carries no kernel-authored message strings to pass through.

#### Scenario: ShowRenderOutput returns error when render has errors

- **WHEN** `render.ShowOutput` is called with a result containing errors
- **THEN** it SHALL call `printValidationError` with the errors
- **AND** it SHALL return an `*ExitError` with `Code` equal to `ExitValidationError`

#### Scenario: ShowRenderOutput shows compact matches in default mode

- **WHEN** `render.ShowOutput` is called with verbose set to `false` and a result containing transformer matches
- **THEN** it SHALL print one line per match using `FormatTransformerMatch`
- **AND** it SHALL print a warning for each unmatched component using `FormatTransformerUnmatched`

#### Scenario: ShowRenderOutput shows detailed matches in verbose mode

- **WHEN** `render.ShowOutput` is called with verbose set to `true`
- **THEN** it SHALL print module metadata (name, namespace, version, components)
- **AND** it SHALL print match details including the match reason
- **AND** it SHALL print per-resource validation lines

#### Scenario: ShowRenderOutput logs warnings

- **WHEN** `render.ShowOutput` is called with a result that has warnings
- **THEN** each warning SHALL be logged via the module-scoped logger at warn level

### Requirement: PrintRenderErrors formats render errors with diagnostic detail

The render refusal printer (`printValidationError`, funnelling into `cmdutil.PrintValidationError`) SHALL accept a render error and print it with appropriate diagnostic information. For an unmatched-components refusal it SHALL print the component names; in verbose mode it SHALL additionally print, for each, the candidate transformers the render build evaluated with the required labels each was found to lack (or the primitive FQNs its bodies conflicted at), while the default output stays one line per component. For an over-subscription refusal it SHALL print every contract key and the catalogs competing for it from the single typed cause that carries them all. For transform errors, it SHALL print the component name, transformer FQN, and cause. Unresolved demands SHALL be rendered from the demand rows themselves, without reconstructing an aggregate error to reach the formatter.

#### Scenario: Unmatched components error with diagnostics

- **WHEN** `printValidationError` is called with an unmatched-components refusal
- **THEN** the output SHALL include the unmatched component names
- **AND** in verbose mode, for each, the candidate transformers evaluated for it with the labels the predicate found missing
- **AND** in default mode, one line per component and no candidate rows

#### Scenario: Over-subscribed contracts print once

- **WHEN** `printValidationError` is called with a refusal carrying two over-subscribed contract keys
- **THEN** both keys and their competing catalogs SHALL be printed, read from one typed cause rather than from two separately joined errors

#### Scenario: Transform error

- **WHEN** `printValidationError` is called with a transform error
- **THEN** the output SHALL include the component name, transformer FQN, and the cause error

#### Scenario: Unresolved demands format from rows

- **WHEN** the CLI formats a render refusal carrying unresolved demands
- **THEN** the formatter SHALL take the demand rows directly, and its output SHALL name each component, kind and contract key with the same-base alternatives the platform implements, or an explicit nothing-implements line
