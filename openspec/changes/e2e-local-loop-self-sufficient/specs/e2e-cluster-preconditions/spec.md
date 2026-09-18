## ADDED Requirements

### Requirement: The e2e suite resolves modules through the shipped default registry

The cluster-backed e2e suite SHALL resolve published modules through the registry mapping the CLI
ships as its default, and SHALL NOT depend on a registry process that no part of the suite starts.
The suite's stub home configuration SHALL NOT name a registry address, so that an invocation which
does not choose a registry for itself resolves exactly as an end user's would.

A test that needs to write to a registry — publishing a module, or scaffolding from a template it
publishes first — SHALL provide one it owns for the duration of that test, routing only the domains
it writes to at that registry and leaving `opmodel.dev` resolving from the shipped default. Such a
registry SHALL require no external process, port reservation or container, and SHALL be released
when the test ends.

A registry that a test does not itself provide SHALL NOT be a precondition of the suite: no test may
require a developer to start one by hand.

#### Scenario: A test that chooses no registry uses the shipped default

- **WHEN** an e2e test runs the CLI without selecting a registry of its own, and no local registry
  process is running
- **THEN** the CLI resolves `opmodel.dev` modules through the shipped default mapping
- **AND** the invocation SHALL NOT fail reporting that a published catalog has no published release

#### Scenario: The operator lifecycle test seeds a Platform

- **WHEN** the operator lifecycle e2e test installs the operator on a prepared cluster with no local
  registry running
- **THEN** the catalog version the install seeds the cluster `Platform` with resolves successfully
- **AND** the test proceeds to its assertions rather than failing on registry resolution

#### Scenario: A publishing test owns its registry

- **WHEN** an e2e test publishes a module
- **THEN** it publishes to a registry it started for that test, needing no external process
- **AND** that registry is released when the test ends, leaving nothing running

#### Scenario: No unreachable registry address is named

- **WHEN** the e2e suite's own configuration and helpers are inspected for a hardcoded local
  registry address
- **THEN** no such address is named as a default any test would fall through to

### Requirement: A destructive test that cannot restore the cluster fails its own run

An e2e test that tears down the shared dev cluster's operator SHALL restore it before the run ends,
and SHALL fail the run when it cannot. Reporting the failure without failing SHALL NOT satisfy this
requirement: a run that leaves the cluster unable to serve the operator-owned tests MUST NOT exit
successfully, because the next run's failure would otherwise appear as an unrelated product error
far from its cause.

The failure message SHALL name the restore step that did not complete, the underlying error, and
the command a developer runs to repair the cluster by hand.

#### Scenario: Restore fails

- **WHEN** a destructive e2e test completes its assertions but the step that rebuilds the dev
  operator fails
- **THEN** the test run SHALL fail
- **AND** the message names the failed restore, the underlying error, and the repair command

#### Scenario: Restore succeeds

- **WHEN** the same test completes and the rebuild step succeeds
- **THEN** the run reports no failure from the restore
- **AND** a subsequent operator-owned test finds a reconciling operator

#### Scenario: A poisoned cluster is never reported as success

- **WHEN** a destructive e2e run ends with the cluster's operator or CRDs absent because the restore
  did not complete
- **THEN** that run's exit status SHALL be non-zero
