## MODIFIED Requirements

### Requirement: PrintRenderErrors formats render errors with diagnostic detail

The render refusal printer (`printValidationError`, funnelling into `cmdutil.PrintValidationError`) SHALL accept a render error and print it with appropriate diagnostic information. For an unmatched-components refusal it SHALL print the component names; in verbose mode it SHALL additionally print, for each, the candidate transformers the render build evaluated with the required labels each was found to lack (or the primitive FQNs its bodies conflicted at), while the default output stays one line per component. For an over-subscription refusal it SHALL print every contract key and the catalogs competing for it from the single typed cause that carries them all. For transform errors, it SHALL print the component name, transformer FQN, and cause. Unresolved demands SHALL be rendered from the demand rows themselves, without reconstructing an aggregate error to reach the formatter, and each row SHALL name the catalog that defines the demanded contract when the row carries one.

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

#### Scenario: A defined but unimplemented demand names its catalog

- **WHEN** an unresolved-demand row carries the registry key of the catalog that defines the demanded contract
- **THEN** the line SHALL name that catalog, distinguishing "defined by this catalog and implemented by nothing" from a contract no enabled catalog defines at all

#### Scenario: A demand no catalog defines keeps the plain line

- **WHEN** an unresolved-demand row carries no defining catalog
- **THEN** the output SHALL keep the nothing-implements line unchanged, naming no catalog
