# Validation Gates

## Purpose

Defines the gate-based validation system in `pkg/loader` that validates consumer-provided values against module and bundle `#config` schemas before loading proceeds. Replaces Go-side builder validation with CUE-native gate checks.

## Requirements

### Requirement: Module Gate validates consumer values against #module.#config
The loader SHALL validate consumer-provided values against the module's `#config` schema before any further processing. This is called the Module Gate.

#### Scenario: Valid values pass the Module Gate
- **WHEN** `LoadModuleReleaseFromValue()` is called with values that satisfy `#module.#config`
- **THEN** loading proceeds to finalization and metadata extraction

#### Scenario: Type mismatch caught by Module Gate
- **WHEN** consumer values contain a field with the wrong type (e.g., string where int is expected)
- **THEN** the Module Gate returns a `*ConfigError` with `Context: "module"` and the raw CUE unification error

#### Scenario: Missing required field caught by Module Gate
- **WHEN** consumer values omit a field that has no default in `#config`
- **THEN** the Module Gate returns a `*ConfigError` with `Context: "module"` and the raw CUE concreteness error

### Requirement: Bundle Gate validates consumer values against #bundle.#config
The loader SHALL validate bundle-level consumer values against the bundle's `#config` schema before processing individual releases. This is called the Bundle Gate.

#### Scenario: Valid bundle values pass the Bundle Gate
- **WHEN** `LoadBundleReleaseFromValue()` is called with values that satisfy `#bundle.#config`
- **THEN** loading proceeds to per-release Module Gate validation

#### Scenario: Bundle Gate runs before individual Module Gates
- **WHEN** bundle-level values fail the Bundle Gate
- **THEN** the error is returned immediately without running per-release Module Gates

### Requirement: ConfigError provides structured field errors
The `ConfigError` type SHALL carry the raw CUE error and provide a `FieldErrors()` method that parses the CUE error tree into `[]FieldError` with file, line, column, path, and message fields.

#### Scenario: FieldErrors parsing
- **WHEN** `configError.FieldErrors()` is called on a ConfigError
- **THEN** it returns a slice of `FieldError` structs with `File`, `Line`, `Column`, `Path`, and `Message` populated from the CUE error positions

#### Scenario: ConfigError.Error() produces human-readable summary
- **WHEN** `configError.Error()` is called
- **THEN** it returns a formatted string with one line per CUE error position, prefixed with the gate context and release name

#### Scenario: ConfigError is unwrappable
- **WHEN** `errors.Unwrap(configError)` is called
- **THEN** it returns the raw CUE error for `errors.Is`/`errors.As` compatibility

### Requirement: Post-gate concreteness check
After gates pass, the loader SHALL validate full concreteness of the release value. If this fails after gates pass, it indicates a bug in module/bundle wiring, not a user error.

#### Scenario: Concreteness failure after gate pass
- **WHEN** the Module Gate passes but the full release value is not concrete
- **THEN** the loader returns an error indicating a wiring issue (distinct from a user values error)
