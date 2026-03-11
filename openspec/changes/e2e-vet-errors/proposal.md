## Why

The CLI error rendering for validation failures recently regressed, causing grouped, structured CUE validation errors to be flattened into noisy, repeated log lines in the terminal output. While unit tests for the error formatting layer passed, the integration between the validation engine and the CLI output layer was broken. We need end-to-end tests that execute the compiled `opm` binary and assert against the actual `stderr` output to prevent this regression from happening again.

## What Changes

- Add isolated, minimal CUE fixtures for testing validation errors (`field not allowed` and `conflicting values`).
- Add an E2E test for `opm rel vet` that verifies the output maintains the grouped format.
- Add an E2E test for `opm mod vet` that verifies the output maintains the grouped format.
- This is a PATCH level change (test additions only).

## Capabilities

### New Capabilities
None

### Modified Capabilities
None

## Impact

- **Affected code:** `tests/e2e/` (adding new test files and fixtures).
- **Dependencies:** None.
- **Systems:** E2E test suite.
