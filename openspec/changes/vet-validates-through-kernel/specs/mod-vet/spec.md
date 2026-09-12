## MODIFIED Requirements

### Requirement: mod vet does not use the render pipeline

The `opm mod vet` command SHALL NOT call the release render pipeline used by `mod build`, `mod apply`, or `opm rel vet`.

It SHALL:

1. Load the module package directly
2. Resolve values as kernel values sources, the same way `mod build` does: each supplied `-f` file as a file-backed source attributed to that file, else the module's `debugValues` as one source attributed to the module's `debugValues`
3. Validate the sources against the module's `#config` through the kernel's layered validation, which reports schema violations and merge conflicts at their source positions and refuses a merged value that is not concrete
4. Print validation output and exit without rendering resources

#### Scenario: mod vet loads module directly

- **WHEN** `opm mod vet .` is run
- **THEN** the command SHALL load the module package directly
- **AND** it SHALL NOT resolve a provider
- **AND** it SHALL NOT compute transformer matches
- **AND** it SHALL NOT render resources

#### Scenario: vet and build agree on a verdict

- **WHEN** `opm mod vet . -f values.cue` and `opm mod build . -f values.cue` are run on the same module and values file
- **THEN** both SHALL accept or both SHALL refuse the values, with the same `#config` violations reported

#### Scenario: Non-concrete values are refused as a #config violation

- **WHEN** `opm mod vet .` is run and the resolved values leave a required `#config` field incomplete
- **THEN** the command SHALL print the standard grouped validation block under "values do not satisfy #config", naming the incomplete field and the source position
- **AND** the exit code SHALL be 2
