## MODIFIED Requirements

<!-- enhancement 0002 D2/D6/D10. Restates only the requirements naming symbols that exist in code AND are renamed by X4: the InstanceSelectorFlags bundle, its LogName helper, the shared inventory-resolution helper, and the mod-command behavioral-equivalence flag references. The RenderFlags / RenderRelease / LoadReleasePackage requirements are pre-existing X1-gap residue (paused simplify-render-single-build names not present in code) and are out of scope per design D-X4.4 / the X1-gap exclusion. -->

### Requirement: InstanceSelectorFlags struct registers and validates instance identification flags

The `InstanceSelectorFlags` struct SHALL provide an `AddTo(*cobra.Command)` method that registers `--instance-name` (string), `--instance-id` (string), and `--namespace`/`-n` (string). It SHALL also provide a `Validate()` method that enforces mutual exclusivity between `--instance-name` and `--instance-id`, and requires exactly one to be present. <!-- Was: ReleaseSelectorFlags, --release-name/--release-id (0002 D10/D-X4.2) -->

#### Scenario: Both selectors set is rejected

- **WHEN** `InstanceSelectorFlags.Validate()` is called with both `InstanceName` and `InstanceID` set to non-empty strings
- **THEN** it SHALL return an error with message containing `"--instance-name and --instance-id are mutually exclusive"`

#### Scenario: Neither selector set is rejected

- **WHEN** `InstanceSelectorFlags.Validate()` is called with both `InstanceName` and `InstanceID` set to empty strings
- **THEN** it SHALL return an error with message containing `"either --instance-name or --instance-id is required"`

#### Scenario: Exactly one selector set is accepted

- **WHEN** `InstanceSelectorFlags.Validate()` is called with `InstanceName` set to `"my-app"` and `InstanceID` set to `""`
- **THEN** it SHALL return no error

### Requirement: InstanceSelectorFlags provides a LogName helper

The `InstanceSelectorFlags` struct SHALL provide a `LogName()` method that returns the instance name if set, or a truncated instance ID prefix (first 8 characters) formatted as `"instance:<prefix>"` otherwise. This is used for scoped logger creation. <!-- Was: ReleaseSelectorFlags.LogName, "release:<prefix>" (0002 D10) -->

#### Scenario: LogName prefers instance name

- **WHEN** `LogName()` is called with `InstanceName` set to `"my-app"`
- **THEN** it SHALL return `"my-app"`

#### Scenario: LogName falls back to truncated instance ID

- **WHEN** `LogName()` is called with `InstanceName` set to `""` and `InstanceID` set to `"a1b2c3d4-e5f6-7890-abcd"`
- **THEN** it SHALL return `"instance:a1b2c3d4"`

### Requirement: Shared inventory resolution helper in cmdutil

`cmdutil.ResolveInventory` SHALL resolve an inventory record from an `*InstanceSelectorFlags` (carrying instance name and/or instance ID). If `flags.InstanceID` is non-empty, it SHALL resolve via `inventory.GetInventory` using the instance ID; if `flags.InstanceName` is also set, that name SHALL be used as the display name. If only `flags.InstanceName` is non-empty, it SHALL resolve via `inventory.FindInventoryByInstanceName`. When no inventory Secret is found, it SHALL return an `InstanceNotFoundError`. <!-- Was: *ReleaseSelectorFlags, ReleaseID/ReleaseName, ReleaseNotFoundError (0002 D10) -->

#### Scenario: Resolve by instance name

- **WHEN** `InstanceSelectorFlags.InstanceName` is set and the inventory Secret exists
- **THEN** `ResolveInventory` SHALL return the matching inventory record

#### Scenario: Resolve by instance ID

- **WHEN** `InstanceSelectorFlags.InstanceID` is set and the inventory Secret exists
- **THEN** `ResolveInventory` SHALL return the matching inventory record

#### Scenario: Instance not found

- **WHEN** the underlying inventory lookup returns no Secret
- **THEN** `ResolveInventory` SHALL return an `InstanceNotFoundError`

### Requirement: Refactored mod commands preserve exact behavioral equivalence

The `mod` subcommands that consume `InstanceSelectorFlags` SHALL preserve the same observable behavior after the rename: identical resolution, output, and exit codes for equivalent inputs, with the flag names updated to `--instance-name`/`--instance-id`. <!-- Was: --release-name/--release-id (0002 D-X4.2) -->

#### Scenario: mod delete behavior preserved under renamed flags

- **WHEN** `opm mod delete --instance-name my-app -n production` is run
- **THEN** it SHALL produce the same resolution and deletion behavior the pre-rename `--release-name` form produced

#### Scenario: mod status behavior preserved under renamed flags

- **WHEN** `opm mod status --instance-name my-app -n production` is run
- **THEN** it SHALL produce the same status output the pre-rename `--release-name` form produced
