## ADDED Requirements

### Requirement: A colliding platform is refused naming the entries to disable

When the kernel refuses a render because more than one enabled registry entry of the platform defines a contract key, every render-bearing command SHALL fail as a validation failure printing the kernel's message under the render-failed header, followed by one row per colliding contract, printed before every other refusing row, naming the contract key and the registry keys (catalog path plus major) of the entries defining it, as the kernel's diagnostics carry them. The refusal applies whatever the instance and whether or not `--skip-unprovided` is passed. An unresolved demand on a colliding key SHALL be worded as defined by more than one enabled registry entry, never as implemented by nothing on the platform, and the render SHALL add no platform-source hint beside a collision. When the kernel refuses a platform as not routable with no row explaining it, the command SHALL fail as a validation failure printing the kernel's message. The CLI SHALL NOT compute collisions of its own. Source: 0026 OQ17 (interim safety net until side-by-side majors).

#### Scenario: A colliding --platform directory is refused with its rows

- **WHEN** `opm instance build` runs with `--platform <dir>` whose platform enables two majors of one catalog listing the same contract keys
- **THEN** the command exits as a validation failure, prints `render failed` with the kernel's message, prints one `defined by more than one enabled registry entry` row per colliding key naming both registry keys before any other row, and prints no manifest

#### Scenario: Skipping unprovided demands does not skip a collision

- **WHEN** the same render runs with `--skip-unprovided`
- **THEN** the render is refused with the same collision rows

#### Scenario: An unresolved demand on a colliding key names the collision

- **WHEN** a refused render carries an unresolved demand whose contract key collides
- **THEN** the demand's row says the key is defined by more than one enabled registry entry and names them, and no hint about the platform source is printed
