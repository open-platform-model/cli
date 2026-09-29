## MODIFIED Requirements

### Requirement: opm platform check reports a platform's contract inventory

The CLI SHALL provide `opm platform check`, registered under a `platform` command group. It SHALL build the resolved platform through the kernel and read its contract inventory, then report: every contract the enabled catalogs define with the catalog that defines it; for each, the implementations that require it; the provider-fulfilled contracts nothing requires; the provider-fulfilled contracts required by transformers of more than one enabled registry entry, each with the registry keys (catalog path plus major) that provide it, whether or not an enabled catalog defines the contract; every pair of enabled transformers whose match predicates are comparable over a shared catalog-fulfilled contract, naming the broader transformer, the narrower transformer and the shared contracts; and a verdict line for each of the three booleans the inventory carries (`fulfilled`, `routable`, `discriminated`). The report SHALL word the verdicts as vacuous only when no enabled catalog defines a contract and no enabled transformer provides one; a verdict line SHALL never say yes when the inventory's boolean is false. The command SHALL NOT apply, render, or contact a cluster.

#### Scenario: A healthy platform reports clean

- **WHEN** `opm platform check` runs against a platform whose enabled catalogs define contracts that are each implemented exactly once and whose transformers sharing a contract are discriminated
- **THEN** the report lists the defined contracts with their defining catalogs, states that nothing is unfulfilled, nothing over-subscribed and nothing comparable, and the command exits 0

#### Scenario: A platform whose catalogs list nothing reports an empty inventory

- **WHEN** the platform's catalogs populate no contract maps and no enabled transformer requires a provider-fulfilled contract
- **THEN** the report states that the catalogs define no contracts, rather than reporting a clean platform, and words all three verdicts as vacuous, so a vacuous answer is distinguishable from a verified one

#### Scenario: The report names the defining catalog

- **WHEN** a contract is defined by an enabled catalog
- **THEN** the report shows that catalog's registry key beside the contract key

#### Scenario: The report names both transformers of a comparable pair

- **WHEN** two enabled transformers have comparable predicates over a catalog-fulfilled contract
- **THEN** the report lists the pair under its own heading, identifying which transformer is broader and which narrower, and names the contracts they share

#### Scenario: Providers of a contract no enabled catalog defines are not vacuous

- **WHEN** the platform's enabled catalogs define no contract, and transformers of two enabled registry entries require one provider-fulfilled contract (its defining catalog disabled or absent from the registry)
- **THEN** the report does not word any verdict as vacuous, lists the contract under the over-subscribed heading marked as defined by no enabled catalog with both registry keys named, and reads `routable: no`

### Requirement: An over-subscribed contract fails the command

A provider-fulfilled contract required by transformers of more than one enabled registry entry SHALL be reported with every providing registry key named (catalog path plus major, sorted), and the command SHALL exit with the validation error code. Two majors of one catalog are two registry entries and so two providers; two transformers of one entry are one provider; the count does not depend on whether an enabled catalog defines the contract. The command reads the count the platform's core derives and the render build enforces, and SHALL NOT compute its own. This mirrors where the refusal lives in the system: `Routable: false` is the condition a platform-package generation step refuses on, so a pre-flight that exits 0 on it would contradict the tool that will reject the platform later.

#### Scenario: Over-subscription exits non-zero

- **WHEN** two enabled catalogs each supply a transformer requiring one provider-fulfilled contract
- **THEN** the report names the contract and both catalogs' registry keys as its providers, and the command exits with the validation error code

#### Scenario: Catalog-fulfilled plurality is not over-subscription

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the report shows them as implementations and the command exits 0

#### Scenario: Two majors of one provider catalog are two providers

- **WHEN** a platform enables two majors of one provider catalog (for example `k8up@v2` and `k8up@v3`) whose transformers each require one provider-fulfilled contract, and the contract's defining catalog is enabled
- **THEN** the report lists the contract as over-subscribed with both registry keys named as its providers, reads `routable: no`, and the command exits with the validation error code

#### Scenario: A disabled defining catalog does not hide over-subscription

- **WHEN** the catalog defining a provider-fulfilled contract is present in the registry with `enable: false`, and transformers of two other enabled entries require the contract
- **THEN** the report lists the contract as over-subscribed, defined by no enabled catalog, with both registry keys named, and the command exits with the validation error code

### Requirement: A platform that cannot build fails with its build diagnostic

If the resolved platform does not build, the command SHALL report the build failure with the CLI's grouped CUE diagnostics and exit with the validation error code, rather than reporting an empty inventory. If the platform builds but its inventory cannot be read, the command SHALL fail naming the missing field and the core release that introduced it, rather than reporting a partial inventory. When the inventory cannot be read because the platform module pins a core release older than one the kernel reads, the command SHALL also name the platform directory and the `cue mod get` command that re-pins core there to the release the kernel was verified against.

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
