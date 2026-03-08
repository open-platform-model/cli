## MODIFIED Requirements

### Requirement: Error types package location
All error types SHALL be defined in `pkg/errors/` (moved from `internal/errors/`). All existing types MUST be preserved: `DetailError`, `ExitError`, `TransformError`, `ValidationError`, `ValuesValidationError`, `FieldError`, `ConflictError`, sentinel errors (`ErrValidation`, `ErrConnectivity`, `ErrPermission`, `ErrNotFound`), and exit code constants.

#### Scenario: Error types importable from pkg/errors
- **WHEN** code imports `github.com/opmodel/cli/pkg/errors`
- **THEN** all error types, sentinels, and exit code constants are accessible

#### Scenario: Import alias convention
- **WHEN** code imports `pkg/errors` alongside stdlib `errors`
- **THEN** the convention `import oerrors "github.com/opmodel/cli/pkg/errors"` SHALL be used

## ADDED Requirements

### Requirement: ConfigError type for gate validation
The `pkg/errors` package (or `pkg/loader`) SHALL export a `ConfigError` type that carries the gate context ("module" or "bundle"), the release name, and the raw CUE error. It SHALL provide `FieldErrors() []FieldError` for structured parsing and `Unwrap() error` for error chain compatibility.

#### Scenario: ConfigError created by gate
- **WHEN** a validation gate detects invalid values
- **THEN** it returns a `*ConfigError` with `Context`, `Name`, and `RawError` populated

#### Scenario: ConfigError provides structured field errors
- **WHEN** `configError.FieldErrors()` is called
- **THEN** it returns parsed `[]FieldError` with file, line, column, path, and message
