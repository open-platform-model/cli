# Errors Domain

## Purpose

Defines the error types domain for OPM. All error types are exported from `pkg/errors/` for use by both internal packages and external tools. Includes the grouped-error helpers (`GroupedError`, `GroupedErrorsFromError`) the CLI's validation output is built on.

---

## Requirements

### Requirement: Error types package location

All error types SHALL be defined in `pkg/errors/`. The package SHALL export `DetailError` (with `NewValidationError` and `Wrap`), `ValidationError`, `GroupedError` and `ErrorLocation` (with `GroupedErrorsFromError`), and the sentinel errors `ErrValidation`, `ErrConnectivity`, `ErrPermission` and `ErrNotFound`. Types that no code in the tree or its consumers constructs (`ConfigError`, `TransformError`, `FieldError`) are removed rather than preserved. Every exported type SHALL keep its `.Error()` string and `Unwrap()` behavior stable across moves within the package.

#### Scenario: Error types importable from pkg/errors
- **WHEN** code imports `github.com/open-platform-model/cli/pkg/errors`
- **THEN** all error types, sentinels, and grouping helpers are accessible

#### Scenario: Import alias convention
- **WHEN** code imports `pkg/errors` alongside stdlib `errors`
- **THEN** the convention `import oerrors "github.com/open-platform-model/cli/pkg/errors"` SHALL be used

#### Scenario: ValidationError wrapping is unchanged
- **WHEN** `errors.As` or `errors.Unwrap` is used with a `ValidationError`
- **THEN** the cause error is correctly unwrapped as before
