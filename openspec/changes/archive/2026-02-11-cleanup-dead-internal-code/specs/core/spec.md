## REMOVED Requirements

### Requirement: testutil package

The `internal/testutil` package provided test helper utilities (`TempDir`, `FixturePath`, `WriteFile`, `CopyFixture`).

**Reason**: The package has zero imports across the entire codebase. No test file uses it.
**Migration**: None required — no consumers exist.

#### Scenario: Package deletion has no effect

- **WHEN** the `internal/testutil/` directory is deleted
- **THEN** `go build ./...` and `task test` SHALL pass without modification to any other file

### Requirement: identity package

The `internal/identity` package provided a single constant `OPMNamespaceUUID`.

**Reason**: The constant is duplicated as a string literal in `build/release_builder.go`. Only one test file imports the package; no production code references it.
**Migration**: The UUID constant SHALL be inlined as an unexported constant in the `build` package. The test SHALL reference the constant directly.

#### Scenario: UUID constant remains available to build package

- **WHEN** the `internal/identity/` directory is deleted
- **THEN** the `build` package SHALL define the UUID as an unexported constant
- **THEN** `build/release_builder_identity_test.go` SHALL continue to validate the UUID value

### Requirement: mod_stubs command

The `internal/cmd/mod_stubs.go` file defined `NewModBuildStubCmd()`, a placeholder for `mod build`.

**Reason**: Superseded by the real `NewModBuildCmd()` in `mod_build.go`. The stub is never registered or called.
**Migration**: None required — no consumers exist.

#### Scenario: Stub deletion has no effect

- **WHEN** `internal/cmd/mod_stubs.go` is deleted
- **THEN** `go build ./...` SHALL pass without modification to any other file

### Requirement: Dead exported functions

Exported functions that are never called outside their own package SHALL be removed or unexported.

**Reason**: Dead exported functions mislead contributors about the package API and add maintenance burden (Principle VII).
**Migration**: None required — no external consumers exist (these are `internal/` packages).

#### Scenario: Dead functions are removed

- **WHEN** dead exported functions are removed from `config/`, `output/`, `errors/`, `kubernetes/`, and `cmd/`
- **THEN** `go build ./...` and `task test` SHALL pass
- **THEN** no remaining exported function in `internal/` SHALL be uncalled from outside its package (excluding test helpers and interface implementations)
