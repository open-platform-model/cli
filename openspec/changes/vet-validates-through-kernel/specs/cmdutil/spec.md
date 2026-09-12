## REMOVED Requirements

### Requirement: PrintValidationError formats render validation errors consistently

**Reason**: The requirement was written around the `*errors.ConfigError` type, which is deleted with the CLI's copy of the kernel validator; the grouped presentation it described is now driven by CUE positions on any error, and the replacement requirement below states that contract without naming the type.

**Migration**: None for users. Code that constructed a `ConfigError` to get grouped output passes the kernel's error tree straight to `PrintValidationError`.

## ADDED Requirements

### Requirement: PrintValidationError groups CUE positions on any error

The `PrintValidationError` function SHALL accept a message string and an error. When the error chain carries CUE errors with at least one valid source position, it SHALL print a summary line counting the distinct issues followed by the grouped CUE details (each distinct message once, with every source position that reports it). Typed kernel refusals (unresolved demands) and `ValidationError` values with details keep their dedicated formats. For other errors, it SHALL use the standard key-value log format.

#### Scenario: Kernel validation error with CUE positions

- **WHEN** `PrintValidationError` is called with the error returned by the kernel's values validation for a values file that violates `#config`
- **THEN** the output SHALL include the summary message with the issue count
- **AND** the output SHALL include the grouped CUE details, each with the values file position, as plain text on stderr

#### Scenario: Generic error without CUE details

- **WHEN** `PrintValidationError` is called with a plain `error`
- **THEN** the output SHALL use the standard error log format with the message and error key-value pair
