## Purpose

Define `opm platform check`: the offline pre-flight that answers whether a platform is usable before any module is deployed against it, reading the contract inventory core derives and library exposes. Covers what it reports, how it picks the platform, and why an unfulfilled contract ends in a different exit code from the two conditions platform-package generation refuses on — an over-subscribed contract and a comparable transformer pair (enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One provider per contract stands; provider classes are rejected in this entry", "Comparable predicates within one catalog-fulfilled bucket are refused, not arbitrated" and "An unfulfilled contract is reported, never refused; only over-subscription refuses platform-package generation").

## Requirements

### Requirement: An unfulfilled contract is reported and does not fail the command

A provider-fulfilled contract that no enabled transformer implements SHALL be listed in the report and SHALL NOT change the exit status. Enhancement 0015 ("An unfulfilled contract is reported, never refused; only over-subscription refuses platform-package generation") makes this a report and never a gate: a platform may legitimately define a contract ahead of the provider that implements it, and the refusal for an unmet demand belongs to the render that demands it.

#### Scenario: Unfulfilled contracts exit zero

- **WHEN** the platform defines a provider-fulfilled contract that nothing implements
- **THEN** the report names the contract and its defining catalog, and the command exits 0

#### Scenario: The report distinguishes the two lists

- **WHEN** a platform is both unfulfilled on one contract and over-subscribed on another
- **THEN** the two appear under separate headings, and only the over-subscription decides the exit status

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

### Requirement: opm platform check is registered under a platform command group and never applies, renders or contacts a cluster

The CLI SHALL provide `opm platform check`, registered under a `platform` command group. The command SHALL NOT apply, render, or contact a cluster.

#### Scenario: The check applies, renders and contacts no cluster
- **WHEN** `opm platform check` runs
- **THEN** the command does not apply, render, or contact a cluster

### Requirement: opm platform check reports the defined contracts, their implementations and the provider-fulfilled contracts nothing requires

`opm platform check` SHALL build the resolved platform through the kernel and read its contract inventory, then report: every contract the enabled catalogs define with the catalog that defines it; for each, the implementations that require it; the provider-fulfilled contracts nothing requires.

#### Scenario: A healthy platform reports clean

- **WHEN** `opm platform check` runs against a platform whose enabled catalogs define contracts that are each implemented exactly once and whose transformers sharing a contract are discriminated
- **THEN** the report lists the defined contracts with their defining catalogs, states that nothing is unfulfilled, nothing over-subscribed and nothing comparable, and the command exits 0

#### Scenario: The report names the defining catalog

- **WHEN** a contract is defined by an enabled catalog
- **THEN** the report shows that catalog's registry key beside the contract key

### Requirement: opm platform check reports every contract key that more than one enabled registry entry defines

`opm platform check` SHALL build the resolved platform through the kernel and read its contract inventory, then report: every contract key more than one enabled registry entry defines, each with the registry keys (catalog path plus major) defining it.

#### Scenario: A platform whose only contracts collide is not vacuous

- **WHEN** the platform enables two majors of one catalog whose contract maps list the same keys, and no other entry defines or provides a contract
- **THEN** the report does not word any verdict as vacuous, does not claim the catalogs define no contracts, lists every shared key under the colliding heading with both registry keys named, and reads `routable: no`

### Requirement: opm platform check reports the provider-fulfilled contracts required by transformers of more than one enabled registry entry

`opm platform check` SHALL build the resolved platform through the kernel and read its contract inventory, then report: the provider-fulfilled contracts required by transformers of more than one enabled registry entry, each with the registry keys (catalog path plus major) that provide it, whether or not an enabled catalog defines the contract.

#### Scenario: Providers of a contract no enabled catalog defines are not vacuous

- **WHEN** the platform's enabled catalogs define no contract, and transformers of two enabled registry entries require one provider-fulfilled contract (its defining catalog disabled or absent from the registry)
- **THEN** the report does not word any verdict as vacuous, lists the contract under the over-subscribed heading marked as defined by no enabled catalog with both registry keys named, and reads `routable: no`

### Requirement: opm platform check reports every comparable pair of enabled transformers

`opm platform check` SHALL build the resolved platform through the kernel and read its contract inventory, then report: every pair of enabled transformers whose match predicates are comparable over a shared catalog-fulfilled contract, naming the broader transformer, the narrower transformer and the shared contracts.

#### Scenario: The report names both transformers of a comparable pair

- **WHEN** two enabled transformers have comparable predicates over a catalog-fulfilled contract
- **THEN** the report lists the pair under its own heading, identifying which transformer is broader and which narrower, and names the contracts they share

### Requirement: opm platform check reports a verdict line for each inventory boolean and words verdicts as vacuous only when nothing is defined, provided or colliding

`opm platform check` SHALL build the resolved platform through the kernel and read its contract inventory, then report: a verdict line for each of the three booleans the inventory carries (`fulfilled`, `routable`, `discriminated`). The report SHALL word the verdicts as vacuous only when no enabled catalog defines a contract, no enabled transformer provides one and no contract key collides; a verdict line SHALL never say yes when the inventory's boolean is false.

#### Scenario: A platform whose catalogs list nothing reports an empty inventory

- **WHEN** the platform's catalogs populate no contract maps and no enabled transformer requires a provider-fulfilled contract
- **THEN** the report states that the catalogs define no contracts, rather than reporting a clean platform, and words all three verdicts as vacuous, so a vacuous answer is distinguishable from a verified one

### Requirement: An over-subscribed contract is reported with its providing registry keys and fails the command

A provider-fulfilled contract required by transformers of more than one enabled registry entry SHALL be reported with every providing registry key named (catalog path plus major, sorted), and `opm platform check` SHALL exit with the validation error code. This mirrors where the refusal lives in the system: `Routable: false` is the condition a platform-package generation step refuses on, so a pre-flight that exits 0 on it would contradict the tool that will reject the platform later.

#### Scenario: Over-subscription exits non-zero

- **WHEN** two enabled catalogs each supply a transformer requiring one provider-fulfilled contract
- **THEN** the report names the contract and both catalogs' registry keys as its providers, and the command exits with the validation error code

#### Scenario: Catalog-fulfilled plurality is not over-subscription

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the report shows them as implementations and the command exits 0

### Requirement: Providers are counted per enabled registry entry and the command computes no count of its own

Two majors of one catalog are two registry entries and so two providers; two transformers of one entry are one provider; the count does not depend on whether an enabled catalog defines the contract. `opm platform check` reads the count the platform's core derives and the render build enforces, and SHALL NOT compute its own.

#### Scenario: Two majors of one provider catalog are two providers

- **WHEN** a platform enables two majors of one provider catalog (for example `k8up@v2` and `k8up@v3`) whose transformers each require one provider-fulfilled contract, and the contract's defining catalog is enabled
- **THEN** the report lists the contract as over-subscribed with both registry keys named as its providers, reads `routable: no`, and the command exits with the validation error code

#### Scenario: A disabled defining catalog does not hide over-subscription

- **WHEN** the catalog defining a provider-fulfilled contract is present in the registry with `enable: false`, and transformers of two other enabled entries require the contract
- **THEN** the report lists the contract as over-subscribed, defined by no enabled catalog, with both registry keys named, and the command exits with the validation error code

### Requirement: A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command

A pair of enabled transformers whose match predicates are comparable over a shared catalog-fulfilled contract SHALL be reported with both transformers and the shared contracts named, and `opm platform check` SHALL exit with the validation error code (enhancement 0015, "Comparable predicates within one catalog-fulfilled bucket are refused, not arbitrated"). No arbitration between the two transformers SHALL be applied or suggested.

#### Scenario: A comparable pair exits non-zero

- **WHEN** one enabled transformer requires a catalog-fulfilled resource alone and another enabled transformer requires the same resource plus a trait
- **THEN** the report names the first as broader, the second as narrower and the resource as shared, states the platform is not discriminated, and the command exits with the validation error code

#### Scenario: Both refusals are reported together

- **WHEN** a platform is over-subscribed on one contract and undiscriminated on another
- **THEN** both appear under their own headings, both verdict lines say no, the error names both counts, and the command exits with the validation error code once

### Requirement: The undiscriminated exit mirrors where platform-package generation refuses

This SHALL mirror where the refusal lives in the system: `Discriminated: false` is a condition platform-package generation refuses on, exactly as `Routable: false` is, so a pre-flight that exits 0 on it would bless a platform the controller rejects. (Requirement: A comparable pair of enabled transformers is reported with both transformers and the shared contracts and fails the command)

#### Scenario: An undiscriminated platform is refused where platform-package generation refuses it
- **WHEN** a platform is not discriminated (`Discriminated: false`)
- **THEN** `opm platform check` exits with the validation error code, as platform-package generation refuses on it

### Requirement: Transformers with incomparable predicates are reported as implementations and do not change the exit status

Transformers with incomparable predicates over a shared contract (a differing required label value, or a required trait the other lacks) SHALL be reported as implementations and SHALL NOT change the exit status.

#### Scenario: Discriminated plurality exits zero

- **WHEN** several enabled transformers require the same catalog-fulfilled contract and each adds a required label value or trait the others lack
- **THEN** the report shows them as implementations, states the platform is discriminated, and the command exits 0

### Requirement: A platform that does not build fails with the grouped CUE diagnostics and the validation error code

If the resolved platform does not build, `opm platform check` SHALL report the build failure with the CLI's grouped CUE diagnostics and exit with the validation error code, rather than reporting an empty inventory.

#### Scenario: A broken platform module reports the build error

- **WHEN** the platform module's registry entry names a catalog it cannot resolve
- **THEN** the command prints the build diagnostic and exits with the validation error code, and prints no contract report

### Requirement: A registry that refuses the credentials fails with the permission error code 4 and the registry login hint

When the platform does not build because the registry refused the credentials, `opm platform check` SHALL exit with the permission error code 4 in place of the validation error code, and SHALL print the registry's answer and the `opm registry login` hint in place of the grouped CUE diagnostics. The hint SHALL name the registry host when every dependency the platform declares routes to one host, and SHALL be the bare command otherwise.

#### Scenario: A registry that refuses the credentials

- **WHEN** the platform module's imports resolve against a registry that answers 401, and every dependency the platform declares routes to that one host
- **THEN** the command prints the registry's 401 answer and the hint `Log in to the registry, then retry:  opm registry login <host>`, exits with the permission error code 4, and prints no contract report

#### Scenario: A refused credential with dependencies on several registry hosts

- **WHEN** a registry answers 401 and the dependencies the platform declares route to more than one host
- **THEN** the command prints the registry's 401 answer and the hint `Log in to the registry, then retry:  opm registry login` without a host, and exits with the permission error code 4

### Requirement: A platform whose inventory cannot be read fails naming the missing field and the core release

If the platform builds but its inventory cannot be read, `opm platform check` SHALL fail naming the missing field and the core release that introduced it, rather than reporting a partial inventory.

#### Scenario: A platform built against a core without the inventory

- **WHEN** the resolved platform pins a core release that derives no contract inventory
- **THEN** the command fails with an error naming the missing field and the core release required, rather than reporting an empty inventory

#### Scenario: A platform built against a core without the comparable-predicate report

- **WHEN** the resolved platform pins a core release that derives the inventory but not its `comparable` report
- **THEN** the command fails with an error naming the `comparable` field and the core release required, and prints no partial report

### Requirement: A platform module that pins an older core also names the platform directory and the cue mod get command

When the inventory cannot be read because the platform module pins a core release older than one the kernel reads, `opm platform check` SHALL also name the platform directory and the `cue mod get` command that re-pins core there to the release the kernel was verified against. (Requirement: A platform whose inventory cannot be read fails naming the missing field and the core release)

#### Scenario: A platform built against a core without the provider count

- **WHEN** the resolved platform pins core `2.0.0-alpha.11`, which derives the inventory and its `comparable` report but not `providedBy`
- **THEN** the command fails with the validation error code, naming the `providedBy` field, the core release `2.0.0-alpha.12`, the platform directory, and `cue mod get opmodel.dev/core@v2.0.0-alpha.12` as the command to run in it, and prints no partial report

### Requirement: A colliding contract key is reported under its own heading and fails the command

A contract key that more than one enabled registry entry's catalog lists SHALL be reported under its own heading with every defining registry key named (catalog path plus major, sorted), ahead of the over-subscribed contracts, and `opm platform check` SHALL exit with the validation error code. The routable verdict SHALL name the colliding count beside the over-subscribed count, and `opm platform check`'s error SHALL name the colliding, over-subscribed and comparable counts.

**Source**: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: Two majors sharing keys exit non-zero and name the entries

- **WHEN** a platform enables `base@v1` and `base@v2` of one catalog and both list the same resource and trait
- **THEN** the report lists both keys under `colliding contracts: 2`, each followed by both registry keys, the routable verdict reads no naming 2 colliding and 0 over-subscribed contracts, and the command exits with the validation error code, its error naming 2 colliding contracts, 0 over-subscribed contracts and 0 comparable pairs

#### Scenario: A collision and an over-subscription are reported together

- **WHEN** two majors of one catalog share one resource key, only the first major defines a provider-fulfilled trait, and transformers of two other enabled entries require that trait
- **THEN** the report lists the resource under the colliding heading and the trait under the over-subscribed heading with both providers, the routable verdict names 1 colliding and 1 over-subscribed contract, and the command exits with the validation error code once

### Requirement: The colliding heading states what is left out and tells the reader to keep one defining entry enabled

The heading SHALL state that a colliding key is left out of the defined, required, unfulfilled and comparable sections, and SHALL tell the reader to keep one of its defining entries enabled (the entries may be two majors of one catalog or two different catalogs listing the same key). A disabled entry never defines a key. (Requirement: A colliding contract key is reported under its own heading and fails the command)

**Source**: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: A disabled second major is not a collision

- **WHEN** the second major of the catalog is present in the registry with `enable: false`
- **THEN** the report has no colliding heading, reads exactly as the platform without that entry, and the exit status is decided as if the entry were absent

### Requirement: The command computes no collision of its own and the fulfilled and discriminated verdicts may read yes on a colliding platform

`opm platform check` reads the collisions the platform's core derives and the render refuses on, and SHALL NOT compute its own. Because a colliding key is absent from the defined contracts, the `fulfilled` and `discriminated` verdicts MAY read yes on a colliding platform; the colliding heading and the exit status carry the refusal.

**Source**: enhancement 0026 (interim safety net until side-by-side majors).

#### Scenario: The fulfilled and discriminated verdicts may read yes on a colliding platform
- **WHEN** a colliding key is absent from the defined contracts
- **THEN** the `fulfilled` and `discriminated` verdicts MAY read yes on a colliding platform, and the colliding heading and the exit status carry the refusal
