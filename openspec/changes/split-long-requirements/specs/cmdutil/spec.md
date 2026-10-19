## REMOVED Requirements

### Requirement: ShowRenderOutput checks for errors, shows transformer matches, and logs warnings
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "ShowRenderOutput returns an ExitError with ExitValidationError when the render has errors", "ShowRenderOutput shows compact transformer matches by default and detailed matches in verbose mode", "ShowRenderOutput logs the render warnings via the module-scoped logger".

### Requirement: PrintRenderErrors formats render errors with diagnostic detail
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The render refusal printer accepts a render error and prints the component name, transformer FQN and cause of a transform error", "The render refusal printer prints the component names of an unmatched-components refusal and the candidate transformers in verbose mode", "The render refusal printer prints every contract key and the competing catalogs of an over-subscription refusal", "Unresolved demands are rendered from the demand rows and name the defining catalog".

### Requirement: Shared inventory resolution helper in the query workflow
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "ResolveInventory resolves the record by instance ID or by instance name", "ResolveInventory returns an InstanceNotFoundError with exit code 5 when no record is found", "ResolveInventory returns the live resources and the missing entries on success".

## MODIFIED Requirements

### Requirement: InstanceSelectorFlags carries and validates the resolved instance selector

The `InstanceSelectorFlags` struct SHALL carry the resolved instance selector (`InstanceName` or `InstanceID`, plus `Namespace`). It SHALL provide a `Validate()` method that enforces mutual exclusivity between `InstanceName` and `InstanceID`, and requires exactly one to be present.

#### Scenario: Both selectors set is rejected

- **WHEN** `InstanceSelectorFlags.Validate()` is called with both `InstanceName` and `InstanceID` set to non-empty strings
- **THEN** it SHALL return an error with message containing `"--instance-name and --instance-id are mutually exclusive"`

#### Scenario: Neither selector set is rejected

- **WHEN** `InstanceSelectorFlags.Validate()` is called with both `InstanceName` and `InstanceID` set to empty strings
- **THEN** it SHALL return an error with message containing `"either --instance-name or --instance-id is required"`

#### Scenario: Exactly one selector set is accepted

- **WHEN** `InstanceSelectorFlags.Validate()` is called with `InstanceName` set to `"my-app"` and `InstanceID` set to `""`
- **THEN** it SHALL return no error


### Requirement: PrintValidationError groups CUE positions on any error

When the error chain carries CUE errors with at least one valid source position, `PrintValidationError` SHALL print a summary line counting the distinct issues followed by the grouped CUE details (each distinct message once, with every source position that reports it). Typed kernel refusals (unresolved demands) and `ValidationError` values with details keep their dedicated formats. For other errors, `PrintValidationError` SHALL use the standard key-value log format.

#### Scenario: Kernel validation error with CUE positions

- **WHEN** `PrintValidationError` is called with the error returned by the kernel's values validation for a values file that violates `#config`
- **THEN** the output SHALL include the summary message with the issue count
- **AND** the output SHALL include the grouped CUE details, each with the values file position, as plain text on stderr

#### Scenario: Generic error without CUE details

- **WHEN** `PrintValidationError` is called with a plain `error`
- **THEN** the output SHALL use the standard error log format with the message and error key-value pair


## ADDED Requirements

### Requirement: Commands build the instance selector from their positional argument

Commands SHALL build the resolved instance selector from their positional `<file|name|uuid>` argument through `InstanceArg.ToSelectorFlags`; no command registers `--instance-name`/`--instance-id` flags for cluster queries. <!-- Was: ReleaseSelectorFlags, --release-name/--release-id (enhancement 0002 ("Every `release`-named file and directory is renamed on disk (extends D8 to the filesystem)")) -->

#### Scenario: A command builds its selector from the positional argument
- **WHEN** a command builds the resolved instance selector
- **THEN** it builds it from its positional `<file|name|uuid>` argument through `InstanceArg.ToSelectorFlags`, and registers no `--instance-name`/`--instance-id` flags for cluster queries


### Requirement: ShowRenderOutput returns an ExitError with ExitValidationError when the render has errors

The render output function (`render.ShowOutput`) SHALL accept a render result and output options (verbose flag). It SHALL:

1. Check for render errors: if present, format and print render errors via `printValidationError`, then return an `*ExitError` with `ExitValidationError`.

#### Scenario: ShowRenderOutput returns error when render has errors

- **WHEN** `render.ShowOutput` is called with a result containing errors
- **THEN** it SHALL call `printValidationError` with the errors
- **AND** it SHALL return an `*ExitError` with `Code` equal to `ExitValidationError`


### Requirement: ShowRenderOutput shows compact transformer matches by default and detailed matches in verbose mode

The render output function (`render.ShowOutput`) SHALL accept a render result and output options (verbose flag). It SHALL:

2. Show transformer match output: verbose mode shows module metadata and match reasons; default mode shows compact match lines.

#### Scenario: ShowRenderOutput shows compact matches in default mode

- **WHEN** `render.ShowOutput` is called with verbose set to `false` and a result containing transformer matches
- **THEN** it SHALL print one line per match using `FormatTransformerMatch`
- **AND** it SHALL print a warning for each unmatched component using `FormatTransformerUnmatched`

#### Scenario: ShowRenderOutput shows detailed matches in verbose mode

- **WHEN** `render.ShowOutput` is called with verbose set to `true`
- **THEN** it SHALL print module metadata (name, namespace, version, components)
- **AND** it SHALL print match details including the match reason
- **AND** it SHALL print per-resource validation lines


### Requirement: ShowRenderOutput logs the render warnings via the module-scoped logger

The render output function (`render.ShowOutput`) SHALL accept a render result and output options (verbose flag). It SHALL:

3. Log any warnings from the render result via the module-scoped logger. The warnings are worded by the CLI from the kernel's advisory diagnostic rows (unhandled optional traits, and resolved-versions rows marked newer); the render result carries no kernel-authored message strings to pass through.

#### Scenario: ShowRenderOutput logs warnings

- **WHEN** `render.ShowOutput` is called with a result that has warnings
- **THEN** each warning SHALL be logged via the module-scoped logger at warn level


### Requirement: PrintValidationError accepts a message string and an error

The `PrintValidationError` function SHALL accept a message string and an error.

#### Scenario: PrintValidationError is called with a message and an error
- **WHEN** `PrintValidationError` is called
- **THEN** it accepts a message string and an error


### Requirement: The render refusal printer accepts a render error and prints the component name, transformer FQN and cause of a transform error

The render refusal printer (`printValidationError`, funnelling into `cmdutil.PrintValidationError`) SHALL accept a render error and print it with appropriate diagnostic information. For transform errors, it SHALL print the component name, transformer FQN, and cause.

#### Scenario: Transform error

- **WHEN** `printValidationError` is called with a transform error
- **THEN** the output SHALL include the component name, transformer FQN, and the cause error


### Requirement: The render refusal printer prints the component names of an unmatched-components refusal and the candidate transformers in verbose mode

For an unmatched-components refusal the render refusal printer SHALL print the component names; in verbose mode it SHALL additionally print, for each, the candidate transformers the render build evaluated with the required labels each was found to lack (or the primitive FQNs its bodies conflicted at), while the default output stays one line per component.

#### Scenario: Unmatched components error with diagnostics

- **WHEN** `printValidationError` is called with an unmatched-components refusal
- **THEN** the output SHALL include the unmatched component names
- **AND** in verbose mode, for each, the candidate transformers evaluated for it with the labels the predicate found missing
- **AND** in default mode, one line per component and no candidate rows


### Requirement: The render refusal printer prints every contract key and the competing catalogs of an over-subscription refusal

For an over-subscription refusal the render refusal printer SHALL print every contract key and the catalogs competing for it from the single typed cause that carries them all.

#### Scenario: Over-subscribed contracts print once

- **WHEN** `printValidationError` is called with a refusal carrying two over-subscribed contract keys
- **THEN** both keys and their competing catalogs SHALL be printed, read from one typed cause rather than from two separately joined errors


### Requirement: Unresolved demands are rendered from the demand rows and name the defining catalog

Unresolved demands SHALL be rendered from the demand rows themselves, without reconstructing an aggregate error to reach the formatter, and each row SHALL name the catalog that defines the demanded contract when the row carries one.

#### Scenario: Unresolved demands format from rows

- **WHEN** the CLI formats a render refusal carrying unresolved demands
- **THEN** the formatter SHALL take the demand rows directly, and its output SHALL name each component, kind and contract key with the same-base alternatives the platform implements, or an explicit nothing-implements line

#### Scenario: A defined but unimplemented demand names its catalog

- **WHEN** an unresolved-demand row carries the registry key of the catalog that defines the demanded contract
- **THEN** the line SHALL name that catalog, distinguishing "defined by this catalog and implemented by nothing" from a contract no enabled catalog defines at all

#### Scenario: A demand no catalog defines keeps the plain line

- **WHEN** an unresolved-demand row carries no defining catalog
- **THEN** the output SHALL keep the nothing-implements line unchanged, naming no catalog


### Requirement: ResolveInventory resolves the record by instance ID or by instance name

`query.ResolveInventory` (in `internal/workflow/query`) SHALL resolve a `ModuleInstance` record from an `*InstanceSelectorFlags`. If `InstanceID` is non-empty, it SHALL resolve via `inventory.FindRecordByInstanceUUID` (list the namespace's CRs, match `status.instanceUUID`); otherwise it SHALL resolve via `inventory.GetRecord` (direct GET by `InstanceName`).

#### Scenario: Resolve by instance name

- **WHEN** `InstanceSelectorFlags.InstanceName` is set and the `ModuleInstance` record exists
- **THEN** `ResolveInventory` SHALL return the matching inventory record with its live resources

#### Scenario: Resolve by instance UUID

- **WHEN** `InstanceSelectorFlags.InstanceID` is set and a `ModuleInstance` in the namespace carries that `status.instanceUUID`
- **THEN** `ResolveInventory` SHALL return the matching inventory record with its live resources


### Requirement: ResolveInventory returns an InstanceNotFoundError with exit code 5 when no record is found

When no record is found, `query.ResolveInventory` SHALL return an `InstanceNotFoundError` with exit code 5. <!-- Was: cmdutil.ResolveInventory over *ReleaseSelectorFlags, ReleaseID/ReleaseName, ReleaseNotFoundError (enhancement 0002, "Every `release`-named file and directory is renamed on disk (extends D8 to the filesystem)") -->

#### Scenario: Instance not found

- **WHEN** the underlying lookup finds no `ModuleInstance` record
- **THEN** `ResolveInventory` SHALL return an `InstanceNotFoundError` carrying exit code 5


### Requirement: ResolveInventory returns the live resources and the missing entries on success

On success `query.ResolveInventory` SHALL also return the live resources and the missing entries from `inventory.DiscoverResourcesFromInventory`.

#### Scenario: A successful resolution returns the live resources and the missing entries
- **WHEN** `query.ResolveInventory` resolves a record
- **THEN** it also returns the live resources and the missing entries from `inventory.DiscoverResourcesFromInventory`
