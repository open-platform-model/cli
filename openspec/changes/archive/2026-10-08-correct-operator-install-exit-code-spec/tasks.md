## 1. Correct the spec

- [x] 1.1 Confirm that code and tests agree on the exit codes: `go test ./internal/operator ./internal/cmd/operator -run 'TestPlanInstall_UnreadableObjectRefuses|TestInstallErrorMapping'` passes, and `installError` in `internal/cmd/operator/install.go` maps a `GuardError` to 2 and the two earlier read errors through the Kubernetes error mapper
- [x] 1.2 Search `openspec/specs/` and `docs/` for every statement of the install exit code for an unreadable object and cover each in the delta specs; record the search result in the change
- [x] 1.3 `task openspec:check` and `task docs:bundle:check` green, then commit docs(openspec): correct the operator install exit code for an unreadable object
