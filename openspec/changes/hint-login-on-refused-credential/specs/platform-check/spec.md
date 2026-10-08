## MODIFIED Requirements

### Requirement: A platform that cannot build fails with its build diagnostic

If the resolved platform does not build, the command SHALL report the build failure with the CLI's grouped CUE diagnostics and exit with the validation error code, rather than reporting an empty inventory. When the platform does not build because the registry refused the credentials, the command SHALL exit with the permission error code 4 in place of the validation error code, and SHALL print the registry's answer and the `opm registry login` hint in place of the grouped CUE diagnostics. If the platform builds but its inventory cannot be read, the command SHALL fail naming the missing field and the core release that introduced it, rather than reporting a partial inventory. When the inventory cannot be read because the platform module pins a core release older than one the kernel reads, the command SHALL also name the platform directory and the `cue mod get` command that re-pins core there to the release the kernel was verified against.

#### Scenario: A broken platform module reports the build error

- **WHEN** the platform module's registry entry names a catalog it cannot resolve
- **THEN** the command prints the build diagnostic and exits with the validation error code, and prints no contract report

#### Scenario: A platform built against a core without the inventory

- **WHEN** the resolved platform pins a core release that derives no contract inventory
- **THEN** the command fails with an error naming the missing field and the core release required, rather than reporting an empty inventory

#### Scenario: A platform built against a core without the comparable-predicate report

- **WHEN** the resolved platform pins a core release that derives the inventory but not its `comparable` report
- **THEN** the command fails with an error naming the `comparable` field and the core release required, and prints no partial report

#### Scenario: A platform built against a core without the provider count

- **WHEN** the resolved platform pins core `2.0.0-alpha.11`, which derives the inventory and its `comparable` report but not `providedBy`
- **THEN** the command fails with the validation error code, naming the `providedBy` field, the core release `2.0.0-alpha.12`, the platform directory, and `cue mod get opmodel.dev/core@v2.0.0-alpha.12` as the command to run in it, and prints no partial report

#### Scenario: A registry that refuses the credentials

- **WHEN** the platform module's imports resolve against a registry that answers 401
- **THEN** the command prints the registry's 401 answer and the hint `Log in to the registry, then retry:  opm registry login <host>`, exits with the permission error code 4, and prints no contract report
