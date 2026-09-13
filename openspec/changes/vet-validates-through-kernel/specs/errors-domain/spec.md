## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: ConfigError type for gate validation

**Reason**: The only producer of `ConfigError` was the CLI's copy of the kernel validator (`pkg/validate`), which is deleted; values validation now returns the library kernel's raw CUE error tree, and the grouped presentation reads positions from that tree directly.

**Migration**: Callers that matched `*ConfigError` with `errors.As` walk the CUE error tree instead (`cuelang.org/go/cue/errors.Errors`, or the CLI's `GroupedErrorsFromError`, which stays in `pkg/errors`). Callers that wanted the gate context and name frame the kernel error with `fmt.Errorf` at the call site.

### Requirement: Error behavior is preserved after move

**Reason**: Its `TransformError` scenario names a type nothing constructs, deleted with this change. The surviving `ValidationError` unwrapping scenario moves under "Error types package location", which now also states the stability rule.

**Migration**: None.
