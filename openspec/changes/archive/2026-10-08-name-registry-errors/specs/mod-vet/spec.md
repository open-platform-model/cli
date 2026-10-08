## MODIFIED Requirements

### Requirement: mod vet exit codes

The `opm mod vet` command SHALL signal its verdict through the process exit code: 0 when validation and the render pass, 1 on a usage error or when the platform cannot be generated or acquired, 2 on any validation failure or render refusal and when the core schema cannot be loaded because no registry is configured, and 3 when the core-schema fetch fails against a configured registry, as the table below details.

| Code | Meaning |
|------|---------|
| 0 | Validation and render passed |
| 1 | Usage error (invalid flags, missing arguments), or the platform could not be generated or acquired (an unpublished pin, a bad `--platform` directory) |
| 2 | Validation error (CUE errors, invalid values, missing `debugValues`, identity/coordinate check failures), render refusal (unmatched components, unresolved demands, a failed transformer), or no registry configured (core-schema fetch) |
| 3 | The core-schema fetch failed against a configured registry (unreachable, or any failure answer) |

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

### Requirement: Registry-aware loading

Vet's module load SHALL resolve dependencies using the resolved registry (flag > env > config precedence), not the ambient process environment alone. Failing to reach a configured registry for the core-schema fetch SHALL exit 3 (connectivity), distinct from check failures (2). A core-schema fetch that fails while no registry is configured (no flag, no `OPM_REGISTRY`, no config value, no `CUE_REGISTRY`) SHALL be reported as a missing registry configuration, not as a connectivity failure.

#### Scenario: Registry flag respected

- **WHEN** vet runs with `--registry` pointing at a reachable mapping
- **THEN** the module's dependencies resolve through that mapping

#### Scenario: A run with only CUE_REGISTRY set is not a missing configuration

- **WHEN** vet runs with no `--registry`, no `OPM_REGISTRY` and no config value, with `CUE_REGISTRY` set, and the core-schema fetch fails
- **THEN** the failure SHALL exit 3 and SHALL NOT be reported as a missing registry configuration
