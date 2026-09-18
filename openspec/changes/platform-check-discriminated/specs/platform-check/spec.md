## MODIFIED Requirements

### Requirement: opm platform check reports a platform's contract inventory

The CLI SHALL provide `opm platform check`, registered under a `platform` command group. It SHALL build the resolved platform through the kernel and read its contract inventory, then report: every contract the enabled catalogs define with the catalog that defines it; for each, the implementations that require it; the provider-fulfilled contracts nothing requires; the provider-fulfilled contracts required by transformers from more than one catalog; every pair of enabled transformers whose match predicates are comparable over a shared catalog-fulfilled contract, naming the broader transformer, the narrower transformer and the shared contracts; and a verdict line for each of the three booleans the inventory carries (`fulfilled`, `routable`, `discriminated`). The command SHALL NOT apply, render, or contact a cluster.

#### Scenario: A healthy platform reports clean

- **WHEN** `opm platform check` runs against a platform whose enabled catalogs define contracts that are each implemented exactly once and whose transformers sharing a contract are discriminated
- **THEN** the report lists the defined contracts with their defining catalogs, states that nothing is unfulfilled, nothing over-subscribed and nothing comparable, and the command exits 0

#### Scenario: A platform whose catalogs list nothing reports an empty inventory

- **WHEN** the platform's catalogs populate no contract maps
- **THEN** the report states that the catalogs define no contracts, rather than reporting a clean platform, and words all three verdicts as vacuous, so a vacuous answer is distinguishable from a verified one

#### Scenario: The report names the defining catalog

- **WHEN** a contract is defined by an enabled catalog
- **THEN** the report shows that catalog's registry key beside the contract key

#### Scenario: The report names both transformers of a comparable pair

- **WHEN** two enabled transformers have comparable predicates over a catalog-fulfilled contract
- **THEN** the report lists the pair under its own heading, identifying which transformer is broader and which narrower, and names the contracts they share

### Requirement: A platform that cannot build fails with its build diagnostic

If the resolved platform does not build, the command SHALL report the build failure with the CLI's grouped CUE diagnostics and exit with the validation error code, rather than reporting an empty inventory. If the platform builds but its inventory cannot be read, the command SHALL fail naming the missing field and the core release that introduced it, rather than reporting a partial inventory.

#### Scenario: A broken platform module reports the build error

- **WHEN** the platform module's registry entry names a catalog it cannot resolve
- **THEN** the command prints the build diagnostic and exits with the validation error code, and prints no contract report

#### Scenario: A platform built against a core without the inventory

- **WHEN** the resolved platform pins a core release that derives no contract inventory
- **THEN** the command fails with an error naming the missing field and the core release required, rather than reporting an empty inventory

#### Scenario: A platform built against a core without the comparable-predicate report

- **WHEN** the resolved platform pins a core release that derives the inventory but not its `comparable` report
- **THEN** the command fails with an error naming the `comparable` field and the core release required, and prints no partial report

## ADDED Requirements

### Requirement: An undiscriminated platform fails the command

A pair of enabled transformers whose match predicates are comparable over a shared catalog-fulfilled contract SHALL be reported with both transformers and the shared contracts named, and the command SHALL exit with the validation error code (enhancement 0015 D5). This mirrors where the refusal lives in the system: `Discriminated: false` is a condition platform-package generation refuses on, exactly as `Routable: false` is, so a pre-flight that exits 0 on it would bless a platform the operator rejects. No arbitration between the two transformers SHALL be applied or suggested. Transformers with incomparable predicates over a shared contract (a differing required label value, or a required trait the other lacks) SHALL be reported as implementations and SHALL NOT change the exit status.

#### Scenario: A comparable pair exits non-zero

- **WHEN** one enabled transformer requires a catalog-fulfilled resource alone and another enabled transformer requires the same resource plus a trait
- **THEN** the report names the first as broader, the second as narrower and the resource as shared, states the platform is not discriminated, and the command exits with the validation error code

#### Scenario: Discriminated plurality exits zero

- **WHEN** several enabled transformers require the same catalog-fulfilled contract and each adds a required label value or trait the others lack
- **THEN** the report shows them as implementations, states the platform is discriminated, and the command exits 0

#### Scenario: Both refusals are reported together

- **WHEN** a platform is over-subscribed on one contract and undiscriminated on another
- **THEN** both appear under their own headings, both verdict lines say no, the error names both counts, and the command exits with the validation error code once
