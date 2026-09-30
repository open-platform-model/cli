## MODIFIED Requirements

### Requirement: Operator-version ceiling

Before applying, the CLI SHALL read the cluster-scoped singleton `Platform` and compare `status.operatorVersion` to its own version on MAJOR.MINOR only: the patch number, the prerelease suffix and any build metadata SHALL be ignored on both sides, because the CLI and the operator share MAJOR.MINOR and release patches and prerelease counters independently. If the Platform or the field is absent, the check SHALL be skipped (solo cluster semantics). If the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR, apply SHALL refuse with an error telling the user to upgrade the CLI; the error SHALL name both full versions as read. If the CLI's own version is not valid semver (dev build), the check SHALL be skipped with a warning. If the operator version is not valid semver, the check SHALL be skipped with a warning. If reading the Platform fails due to RBAC, the check SHALL be skipped with a warning (a namespace-scoped user must remain able to apply).

#### Scenario: Solo cluster skips the ceiling

- **WHEN** no `Platform` CR exists or `status.operatorVersion` is absent
- **THEN** the ceiling check SHALL be skipped and apply SHALL proceed

#### Scenario: Older CLI refused

- **WHEN** `status.operatorVersion` is `1.2.0` and the CLI version is `1.1.0`
- **THEN** apply SHALL exit non-zero with an upgrade-the-CLI error

#### Scenario: Dev build skips with warning

- **WHEN** the CLI version is `dev`
- **THEN** the ceiling check SHALL be skipped and a warning SHALL be printed

#### Scenario: Platform read denied degrades to warning

- **WHEN** reading the `Platform` returns a forbidden error
- **THEN** the ceiling check SHALL be skipped with a warning and apply SHALL proceed

#### Scenario: Newer prerelease counter on the same line passes

- **WHEN** `status.operatorVersion` is `1.0.0-beta.3` and the CLI version is `1.0.0-beta.2`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Beta operator against an alpha CLI of the same line passes

- **WHEN** `status.operatorVersion` is `1.0.0-beta.1` and the CLI version is `1.0.0-alpha.27`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Newer operator patch passes

- **WHEN** `status.operatorVersion` is `1.0.4` and the CLI version is `1.0.1`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Newer operator minor refused

- **WHEN** `status.operatorVersion` is `1.1.0` (or `1.1.0-beta.1`) and the CLI version is `1.0.0`
- **THEN** apply SHALL exit non-zero with an upgrade-the-CLI error naming both versions
