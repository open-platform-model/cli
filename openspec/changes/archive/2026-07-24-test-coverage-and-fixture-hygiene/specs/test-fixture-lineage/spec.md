# test-fixture-lineage

## Purpose

Keeps the repo's fixtures and examples on the current published schema line and its tests free of sibling-checkout dependencies. (Test-infrastructure capability; precedent: `validation-gates`, `kind-cluster-tasks`, `ci-workflow`.)

## ADDED Requirements

### Requirement: Maintained fixtures track the current schema line

Fixtures and examples consumed by tests or presented as current documentation SHALL import only the current published schema line (`opmodel.dev/core@v1` and current catalogs). Artifacts kept for the retired line SHALL live under an explicitly marked legacy location with a note naming the line they document, and SHALL NOT be consumed by any test.

#### Scenario: No retired-schema imports outside legacy

- **WHEN** the repo is grepped for retired-line imports (`core/v1alpha1`, `modulerelease@v1`, `opm/v1alpha1`) outside the marked legacy location
- **THEN** there SHALL be no matches in fixtures, examples, or test inputs

#### Scenario: Vet fixtures exercise current schema

- **WHEN** the module vet tests run
- **THEN** their fixtures SHALL be `core@v1`-line modules exercising the same behaviors (valid module, secrets discovery, debug values) as before the port

### Requirement: Tests depend only on repo-local fixtures

No test or integration program SHALL read fixtures from a sibling repository checkout. Vendored copies SHALL carry a provenance header naming their origin.

#### Scenario: render-parity is self-contained

- **WHEN** the render-parity program runs in a standalone clone of this repo (no sibling checkouts)
- **THEN** it SHALL locate its module fixture under this repo's `tests/fixtures/` and proceed to the registry-gated comparison
