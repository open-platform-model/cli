## Purpose

Define `opm platform check`: the offline pre-flight that answers whether a platform is usable before any module is deployed against it, reading the contract inventory core derives and library exposes. Covers what it reports, how it picks the platform, and why an unfulfilled contract ends in a different exit code from the two conditions platform-package generation refuses on — an over-subscribed contract and a comparable transformer pair (enhancement 0015 D1, D2, D5, D18).

## Requirements

### Requirement: opm platform check reports a platform's contract inventory

The CLI SHALL provide `opm platform check`, registered under a `platform` command group. It SHALL build the resolved platform through the kernel and read its contract inventory, then report: every contract the enabled catalogs define with the catalog that defines it; for each, the implementations that require it; the provider-fulfilled contracts nothing requires; every contract key more than one enabled registry entry defines, each with the registry keys (catalog path plus major) defining it; the provider-fulfilled contracts required by transformers of more than one enabled registry entry, each with the registry keys (catalog path plus major) that provide it, whether or not an enabled catalog defines the contract; every pair of enabled transformers whose match predicates are comparable over a shared catalog-fulfilled contract, naming the broader transformer, the narrower transformer and the shared contracts; and a verdict line for each of the three booleans the inventory carries (`fulfilled`, `routable`, `discriminated`). The report SHALL word the verdicts as vacuous only when no enabled catalog defines a contract, no enabled transformer provides one and no contract key collides; a verdict line SHALL never say yes when the inventory's boolean is false. The command SHALL NOT apply, render, or contact a cluster.

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

#### Scenario: A platform whose only contracts collide is not vacuous

- **WHEN** the platform enables two majors of one catalog whose contract maps list the same keys, and no other entry defines or provides a contract
- **THEN** the report does not word any verdict as vacuous, does not claim the catalogs define no contracts, lists every shared key under the colliding heading with both registry keys named, and reads `routable: no`

### Requirement: An unfulfilled contract is reported and does not fail the command

A provider-fulfilled contract that no enabled transformer implements SHALL be listed in the report and SHALL NOT change the exit status. Enhancement 0015 D18 makes this a report and never a gate: a platform may legitimately define a contract ahead of the provider that implements it, and the refusal for an unmet demand belongs to the render that demands it.

#### Scenario: Unfulfilled contracts exit zero

- **WHEN** the platform defines a provider-fulfilled contract that nothing implements
- **THEN** the report names the contract and its defining catalog, and the command exits 0

#### Scenario: The report distinguishes the two lists

- **WHEN** a platform is both unfulfilled on one contract and over-subscribed on another
- **THEN** the two appear under separate headings, and only the over-subscription decides the exit status

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

### Requirement: The checked platform is resolved like every other command's

`opm platform check` SHALL resolve which platform to check using the CLI's existing precedence and SHALL report the resolved location and how it was chosen, so a report can never be misread as describing a different platform. It MAY accept a platform directory as a positional argument, which takes precedence over the flag and the configured default.

#### Scenario: The report states its provenance

- **WHEN** the command runs with no argument and no flag, against the configured default platform
- **THEN** the report's first line names the resolved directory and that it came from the configured default

#### Scenario: A directory argument wins

- **WHEN** the command is given a platform directory argument while a different platform is configured
- **THEN** the argument's directory is checked and the report names it

#### Scenario: A directory that is not a platform module is refused

- **WHEN** the resolved location is a directory with no CUE module manifest
- **THEN** the command fails with a not-found style error naming the directory, before any build is attempted

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

### Requirement: A colliding contract key fails the command

A contract key that more than one enabled registry entry's catalog lists SHALL be reported under its own heading with every defining registry key named (catalog path plus major, sorted), ahead of the over-subscribed contracts, and the command SHALL exit with the validation error code. The heading SHALL state that a colliding key is left out of the defined, required, unfulfilled and comparable sections, and SHALL tell the reader to keep one of its defining entries enabled (the entries may be two majors of one catalog or two different catalogs listing the same key). The routable verdict SHALL name the colliding count beside the over-subscribed count, and the command's error SHALL name the colliding, over-subscribed and comparable counts. A disabled entry never defines a key. The command reads the collisions the platform's core derives and the render refuses on, and SHALL NOT compute its own. Because a colliding key is absent from the defined contracts, the `fulfilled` and `discriminated` verdicts MAY read yes on a colliding platform; the colliding heading and the exit status carry the refusal. Source: 0026 OQ17 (interim safety net until side-by-side majors).

#### Scenario: Two majors sharing keys exit non-zero and name the entries

- **WHEN** a platform enables `base@v1` and `base@v2` of one catalog and both list the same resource and trait
- **THEN** the report lists both keys under `colliding contracts: 2`, each followed by both registry keys, the routable verdict reads no naming 2 colliding and 0 over-subscribed contracts, and the command exits with the validation error code, its error naming 2 colliding contracts, 0 over-subscribed contracts and 0 comparable pairs

#### Scenario: A collision and an over-subscription are reported together

- **WHEN** two majors of one catalog share one resource key, only the first major defines a provider-fulfilled trait, and transformers of two other enabled entries require that trait
- **THEN** the report lists the resource under the colliding heading and the trait under the over-subscribed heading with both providers, the routable verdict names 1 colliding and 1 over-subscribed contract, and the command exits with the validation error code once

#### Scenario: A disabled second major is not a collision

- **WHEN** the second major of the catalog is present in the registry with `enable: false`
- **THEN** the report has no colliding heading, reads exactly as the platform without that entry, and the exit status is decided as if the entry were absent
