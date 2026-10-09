## MODIFIED Requirements

### Requirement: mod vet exit codes

The `opm mod vet` command SHALL signal its verdict through the process exit code: 0 when validation and the render pass, 1 on a usage error or when the platform cannot be generated or acquired, 2 on any validation failure or render refusal and when the core schema cannot be loaded because no registry is configured, 3 when the core-schema fetch fails against a configured registry, and 4 when that registry refuses the credentials, as the table below details. The mapping of a failed core-schema fetch SHALL be the one `opm module publish` uses, and a refusal SHALL print the `opm registry login <host>` hint.

| Code | Meaning |
|------|---------|
| 0 | Validation and render passed |
| 1 | Usage error (invalid flags, missing arguments), or the platform could not be generated or acquired (an unpublished pin, a bad `--platform` directory) |
| 2 | Validation error (CUE errors, invalid values, missing `debugValues`, identity/coordinate check failures), render refusal (unmatched components, unresolved demands, a failed transformer), or no registry configured (core-schema fetch) |
| 3 | The core-schema fetch failed against a configured registry (unreachable, or a failure answer other than a refused credential) |
| 4 | The registry refused the credentials on the core-schema fetch (a 401, or a 403 that reaches the cli as a refusal) |

#### Scenario: Exit code 0 on success

- **WHEN** `opm mod vet .` succeeds
- **THEN** the exit code SHALL be 0

#### Scenario: Exit code 2 on validation failure

- **WHEN** `opm mod vet .` fails due to CUE errors
- **THEN** the exit code SHALL be 2

#### Scenario: Exit code 2 on render refusal

- **WHEN** `opm mod vet .` passes the `#config` check but a component matches no transformer in the module's catalogs
- **THEN** the command SHALL print the kernel's refusal with its diagnostics
- **AND** the exit code SHALL be 2

#### Scenario: Exit code 2 when no registry is configured

- **WHEN** `opm mod vet .` runs with no `--registry`, no `OPM_REGISTRY`, no `registry` in the config file and no `CUE_REGISTRY`, and the core schema cannot be loaded
- **THEN** the message SHALL say no registry is configured, keep the cause, and point to `opm config init`
- **AND** the exit code SHALL be 2

#### Scenario: Exit code 4 when the registry refuses the credentials

- **WHEN** `opm mod vet .` runs against a configured registry that answers the core-schema fetch with 401
- **THEN** the message SHALL say the registry refused the credentials, keep the cause, and end with `opm registry login <host>`
- **AND** the exit code SHALL be 4

#### Scenario: Exit code 3 when the registry is unreachable or fails

- **WHEN** the configured registry refuses the connection, or answers the core-schema fetch with 503
- **THEN** the exit code SHALL be 3 and the message SHALL NOT name the login

#### Scenario: A 403 on the tag lookup reads as a failed registry operation

- **WHEN** the configured registry answers the core-schema fetch with 403, which CUE's registry client reports as "not found"
- **THEN** the exit code SHALL be 3, as it is for `opm module publish`
