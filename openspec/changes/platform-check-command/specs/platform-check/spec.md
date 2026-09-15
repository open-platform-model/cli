## Purpose

Define `opm platform check`: the offline pre-flight that answers whether a platform is usable before any module is deployed against it, reading the contract inventory core derives and library exposes. Covers what it reports, how it picks the platform, and why an unfulfilled contract and an over-subscribed one end in different exit codes (enhancement 0015 D1, D2, D18).

## ADDED Requirements

### Requirement: opm platform check reports a platform's contract inventory

The CLI SHALL provide `opm platform check`, registered under a `platform` command group. It SHALL build the resolved platform through the kernel and read its contract inventory, then report: every contract the enabled catalogs define with the catalog that defines it; for each, the implementations that require it; the provider-fulfilled contracts nothing requires; the provider-fulfilled contracts required by transformers from more than one catalog; and a verdict line for each of the two booleans the inventory carries. The command SHALL NOT apply, render, or contact a cluster.

#### Scenario: A healthy platform reports clean

- **WHEN** `opm platform check` runs against a platform whose enabled catalogs define contracts that are each implemented exactly once
- **THEN** the report lists the defined contracts with their defining catalogs, states that nothing is unfulfilled and nothing over-subscribed, and the command exits 0

#### Scenario: A platform whose catalogs list nothing reports an empty inventory

- **WHEN** the platform's catalogs populate no contract maps
- **THEN** the report states that the catalogs define no contracts, rather than reporting a clean platform, so a vacuous answer is distinguishable from a verified one

#### Scenario: The report names the defining catalog

- **WHEN** a contract is defined by an enabled catalog
- **THEN** the report shows that catalog's registry key beside the contract key

### Requirement: An unfulfilled contract is reported and does not fail the command

A provider-fulfilled contract that no enabled transformer implements SHALL be listed in the report and SHALL NOT change the exit status. Enhancement 0015 D18 makes this a report and never a gate: a platform may legitimately define a contract ahead of the provider that implements it, and the refusal for an unmet demand belongs to the render that demands it.

#### Scenario: Unfulfilled contracts exit zero

- **WHEN** the platform defines a provider-fulfilled contract that nothing implements
- **THEN** the report names the contract and its defining catalog, and the command exits 0

#### Scenario: The report distinguishes the two lists

- **WHEN** a platform is both unfulfilled on one contract and over-subscribed on another
- **THEN** the two appear under separate headings, and only the over-subscription decides the exit status

### Requirement: An over-subscribed contract fails the command

A provider-fulfilled contract required by transformers from more than one catalog SHALL be reported with every competing catalog named, and the command SHALL exit with the validation error code. This mirrors where the refusal lives in the system: `Routable: false` is the condition a platform-package generation step refuses on, so a pre-flight that exits 0 on it would contradict the tool that will reject the platform later.

#### Scenario: Over-subscription exits non-zero

- **WHEN** two enabled catalogs each supply a transformer requiring one provider-fulfilled contract
- **THEN** the report names the contract and both catalogs, and the command exits with the validation error code

#### Scenario: Catalog-fulfilled plurality is not over-subscription

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the report shows them as implementations and the command exits 0

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

If the resolved platform does not build, the command SHALL report the build failure with the CLI's grouped CUE diagnostics and exit with the validation error code, rather than reporting an empty inventory.

#### Scenario: A broken platform module reports the build error

- **WHEN** the platform module's registry entry names a catalog it cannot resolve
- **THEN** the command prints the build diagnostic and exits with the validation error code, and prints no contract report

#### Scenario: A platform built against a core without the inventory

- **WHEN** the resolved platform pins a core release that derives no contract inventory
- **THEN** the command fails with an error naming the missing field and the core release required, rather than reporting an empty inventory
