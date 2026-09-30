## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: A colliding contract key fails the command

A contract key that more than one enabled registry entry's catalog lists SHALL be reported under its own heading with every defining registry key named (catalog path plus major, sorted), ahead of the over-subscribed contracts, and the command SHALL exit with the validation error code. The heading SHALL state that a colliding key is left out of the defined, required and unfulfilled sections, and that one major of the catalog must be enabled. The routable verdict SHALL name the colliding count beside the over-subscribed count, and the command's error SHALL name the colliding, over-subscribed and comparable counts. A disabled entry never defines a key. The command reads the collisions the platform's core derives and the render refuses on, and SHALL NOT compute its own. Because a colliding key is absent from the defined contracts, the `fulfilled` and `discriminated` verdicts MAY read yes on a colliding platform; the colliding heading and the exit status carry the refusal. Source: 0026 OQ17 (interim safety net until side-by-side majors).

#### Scenario: Two majors sharing keys exit non-zero and name the entries

- **WHEN** a platform enables `base@v1` and `base@v2` of one catalog and both list the same resource and trait
- **THEN** the report lists both keys under `colliding contracts: 2`, each followed by both registry keys, the routable verdict reads no naming 2 colliding and 0 over-subscribed contracts, and the command exits with the validation error code, its error naming 2 colliding contracts, 0 over-subscribed contracts and 0 comparable pairs

#### Scenario: A collision and an over-subscription are reported together

- **WHEN** two majors of one catalog share one resource key, only the first major defines a provider-fulfilled trait, and transformers of two other enabled entries require that trait
- **THEN** the report lists the resource under the colliding heading and the trait under the over-subscribed heading with both providers, the routable verdict names 1 colliding and 1 over-subscribed contract, and the command exits with the validation error code once

#### Scenario: A disabled second major is not a collision

- **WHEN** the second major of the catalog is present in the registry with `enable: false`
- **THEN** the report has no colliding heading, reads exactly as the platform without that entry, and the exit status is decided as if the entry were absent
