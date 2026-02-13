## 1. Warning Handler Testability

- [x] 1.1 Define unexported `warningLogger` interface in `internal/kubernetes/warnings.go` with `Warn(msg string, keyvals ...interface{})` and `Debug(msg string, keyvals ...interface{})` methods
- [x] 1.2 Add `logger warningLogger` field to `opmWarningHandler` struct
- [x] 1.3 Replace direct `output.Warn(msg)` / `output.Debug(msg)` calls with `h.logger.Warn(msg)` / `h.logger.Debug(msg)` in `HandleWarningHeader()`
- [x] 1.4 Create `outputWarningLogger` struct that implements `warningLogger` by delegating to `output.Warn` / `output.Debug`
- [x] 1.5 Update `opmWarningHandler` construction in `internal/kubernetes/client.go` to pass `&outputWarningLogger{}` as the logger

## 2. Warning Handler Tests

- [x] 2.1 Create `mockWarningLogger` in `internal/kubernetes/warnings_test.go` that records method name and message
- [x] 2.2 Update `TestOpmWarningHandler` to inject `mockWarningLogger` into the handler
- [x] 2.3 Assert `wantLevel` field: `"WARN"` → mock's `Warn()` called, `"DEBU"` → mock's `Debug()` called, `"none"` → neither called

## 3. Discovery Group Failure Logging

- [x] 3.1 Add `output.Warn()` call in `discoverAPIResources()` at `discovery.go` after the `IsGroupDiscoveryFailedError` guard, including the error for diagnostic context
- [x] 3.2 Add `"github.com/opmodel/cli/internal/output"` import to `discovery.go`

## 4. Config Init UX

- [x] 4.1 Add `cue mod tidy` hint line to success output in `internal/cmd/config_init.go` before the `Validate with: opm config vet` line

## 5. Validation

- [x] 5.1 Run `task check` (fmt + vet + test) to verify all changes pass
