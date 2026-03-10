# pkg vs internal reorganization checklist

## Planning

- [x] Write the detailed reorganization plan
- [x] Write an execution checklist

## Package moves

- [x] Move `pkg/releaseprocess` to `internal/releaseprocess`
- [x] Move `pkg/match` to `internal/match`
- [x] Move `pkg/engine` to `internal/engine`
- [x] Move `pkg/modulerelease` to `internal/runtime/modulerelease`
- [x] Move `pkg/bundlerelease` to `internal/runtime/bundlerelease`

## API cleanup

- [x] Move CLI exit codes and `ExitError` from `pkg/errors` to `internal/exit`
- [x] Move resource apply-order weights from `pkg/core` to `internal/resourceorder`
- [x] Update imports and package references across the repository

## Verification

- [x] Run formatting
- [x] Run targeted tests for moved packages and affected workflows
- [x] Update this checklist with final completion status
