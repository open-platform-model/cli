## MODIFIED Requirements

### Requirement: Error types package location
All error types SHALL be defined in `pkg/errors/` (moved from `internal/errors/`). All existing types MUST be preserved: `DetailError`, `ExitError`, `TransformError`, `ValidationError`, `ValuesValidationError`, `FieldError`, `ConflictError`, sentinel errors (`ErrValidation`, `ErrConnectivity`, `ErrPermission`, `ErrNotFound`), and exit code constants.

#### Scenario: Error types importable from pkg/errors
- **WHEN** code imports `github.com/open-platform-model/cli/pkg/errors`
- **THEN** all error types, sentinels, and exit code constants are accessible

#### Scenario: Import alias convention
- **WHEN** code imports `pkg/errors` alongside stdlib `errors`
- **THEN** the convention `import oerrors "github.com/open-platform-model/cli/pkg/errors"` SHALL be used
