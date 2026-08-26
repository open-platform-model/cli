# Capability: catalog-registry-check

## Purpose

Out-of-band verification of a published catalog (enhancement 0011, slice cli-catalog-gates, decision D7): `opm catalog registry check` runs the consumer's identity verification — and optionally the predecessor comparison — against a build already in a registry, on demand. It is an aid with the D35 contract stated in its own help text: nothing requires it to have been run, and enforcement exists only at publish; it does not make an unchecked catalog trustworthy, it makes it checkable.

## Requirements

### Requirement: Out-of-band verification of a published catalog

`opm catalog registry check <path@version>` SHALL pull the named published build and verify, out of band, what a consumer verifies at materialize: the declared identity is concrete and its `modulePath` and `version` agree with the coordinate the build was fetched by. It SHALL report the catalog's member inventory per kind and apiVersion. With `--compat`, it SHALL additionally run the predecessor comparison for the fetched build exactly as publish would have. The command's help text SHALL state that the check is an aid and not a guarantee — enforcement exists only at publish. Exit codes: 0 clean, 2 findings, 3 registry unreachable.

#### Scenario: Identity mismatch found out of band

- **WHEN** a published build's declared version disagrees with the tag it was fetched by
- **THEN** the check reports both values and exits 2

#### Scenario: Compat aid over a published build

- **WHEN** `--compat` runs against a build that broke a beta contract relative to its predecessor
- **THEN** the same violations publish would have refused are reported

#### Scenario: Aid, not guarantee

- **WHEN** the command's help is displayed
- **THEN** it states the check is an aid and that only publish enforces

### Requirement: `--compat` follows the publish gate's dev-build rules

`opm catalog registry check <path@version> --compat` SHALL apply the same dev-build rules as `opm catalog publish`: a fetched build whose version is a dev prerelease SHALL report `dev-exempt` on the `compat` row instead of a comparison, and the predecessor window for any fetched build SHALL exclude dev tags. The two commands SHALL agree on every build.

#### Scenario: Dev-tagged build reports exempt

- **WHEN** `--compat` runs against a published `-0.dev.*` build
- **THEN** the report's `compat` row reads `dev-exempt` and the exit code is 0 unless another finding exists

#### Scenario: Release build ignores dev history

- **WHEN** `--compat` runs against a release build whose history holds a newer dev tag than the latest release
- **THEN** the comparison runs against the release tag and never names a dev coordinate
