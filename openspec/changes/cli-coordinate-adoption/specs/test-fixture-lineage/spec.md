# test-fixture-lineage — Delta

## MODIFIED Requirements

### Requirement: Maintained fixtures track the current schema line

Fixtures and examples consumed by tests or presented as current documentation SHALL import only the current published schema line (`opmodel.dev/core@v2` and the versioned `opmodel.dev/catalogs/opm@v2` packages). Artifacts kept for a retired line SHALL live under an explicitly marked legacy location with a note naming the line they document, and SHALL NOT be consumed by any test.

#### Scenario: No retired-schema imports outside legacy

- **WHEN** the repo is grepped for retired-line imports (`opmodel.dev/core@v1`, `core/v1alpha1`, `modulerelease@v1`, `opm/v1alpha1`) outside the marked legacy location
- **THEN** there SHALL be no matches in fixtures, examples, or test inputs

#### Scenario: Vet fixtures exercise current schema

- **WHEN** the module vet tests run
- **THEN** their fixtures SHALL be `core@v2`-line modules exercising the same behaviors (valid module, secrets discovery, debug values) as before the port
