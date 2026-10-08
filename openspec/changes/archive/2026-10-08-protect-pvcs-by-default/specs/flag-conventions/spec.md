## ADDED Requirements

### Requirement: Delete-data is the only flag that allows deletion of PersistentVolumeClaims

`opm instance delete`, `opm instance apply` and `opm module apply` SHALL each accept `--delete-data`, a boolean flag with default false and no shorthand, and with it SHALL delete tracked PersistentVolumeClaims that they keep without it. No other flag SHALL have that effect: `--yes` only skips the confirmation prompt, and `--force` only allows the prune of an empty render. The help of each of the three commands SHALL say that PersistentVolumeClaims are kept by default and SHALL name `--delete-data`.

#### Scenario: The flag is offered with a safe default

- **WHEN** the flags of `opm instance delete`, `opm instance apply` and `opm module apply` are read
- **THEN** each SHALL offer `--delete-data`, default false, not deprecated, with no shorthand

#### Scenario: Yes alone keeps claims

- **WHEN** `opm instance delete jellyfin -n media --yes` is run for a CLI-owned instance that tracks a PersistentVolumeClaim
- **THEN** the claim SHALL be kept

#### Scenario: Force alone keeps claims

- **WHEN** `opm instance apply --force` prunes every tracked resource of an empty render
- **THEN** a tracked PersistentVolumeClaim SHALL be kept
