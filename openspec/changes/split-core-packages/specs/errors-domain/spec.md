## ADDED Requirements

### Requirement: Domain error types are consolidated in internal/errors
`TransformError` and `ValidationError` SHALL be defined in `internal/errors/domain.go` (package `errors`). They SHALL NOT remain in `internal/core`.

#### Scenario: Domain errors are accessible from internal/errors
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/errors`
- **THEN** both `TransformError` and `ValidationError` are accessible

#### Scenario: Domain errors are no longer in internal/core
- **WHEN** the codebase is built after this change
- **THEN** no file references `core.TransformError` or `core.ValidationError`

### Requirement: internal/errors is split into three files
The `internal/errors` package SHALL be organized into three files with clear, distinct responsibilities:
- `errors.go` — `DetailError`, `ExitError`, exit code constants, `Wrap()`
- `sentinel.go` — sentinel error variables (`ErrValidation`, `ErrConnectivity`, `ErrPermission`, `ErrNotFound`)
- `domain.go` — `TransformError`, `ValidationError`

#### Scenario: Each file has a single responsibility
- **WHEN** a developer opens `internal/errors/`
- **THEN** the purpose of each file is immediately clear from its name and contents

### Requirement: Error behavior is preserved after move
`TransformError` and `ValidationError` SHALL produce identical `.Error()` strings and `Unwrap()` behavior after the move.

#### Scenario: TransformError message is unchanged
- **WHEN** `TransformError.Error()` is called
- **THEN** the formatted string `component %q, transformer %q: %v` is returned identically

#### Scenario: ValidationError wrapping is unchanged
- **WHEN** `errors.As` or `errors.Unwrap` is used with a `ValidationError`
- **THEN** the cause error is correctly unwrapped as before
