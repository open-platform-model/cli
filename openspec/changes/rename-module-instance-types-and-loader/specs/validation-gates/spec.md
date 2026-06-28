## MODIFIED Requirements

### Requirement: Module Gate validates consumer values against #module.#config
The loader SHALL validate consumer-provided values against the module's `#config` schema before any further processing. This is called the Module Gate.

#### Scenario: Valid values pass the Module Gate
- **WHEN** `LoadModuleInstanceFromValue()` is called with values that satisfy `#module.#config`
- **THEN** loading proceeds to finalization and metadata extraction

#### Scenario: Type mismatch caught by Module Gate
- **WHEN** consumer values contain a field with the wrong type (e.g., string where int is expected)
- **THEN** the Module Gate returns a `*ConfigError` with `Context: "module"` and the raw CUE unification error

#### Scenario: Missing required field caught by Module Gate
- **WHEN** consumer values omit a field that has no default in `#config`
- **THEN** the Module Gate returns a `*ConfigError` with `Context: "module"` and the raw CUE concreteness error
