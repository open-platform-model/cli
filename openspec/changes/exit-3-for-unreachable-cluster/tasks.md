## 1. The connectivity rule in the shared function

- [ ] 1.1 Add rows to `TestExitCodeFromK8sError` in `internal/cmdutil/exit_test.go`: a `*url.Error` with a refused dial (`*net.OpError`) gives 3, a wrapped `context.DeadlineExceeded` gives 3, a `*url.Error` that wraps `context.Canceled` gives 1, a `fmt.Errorf("...: %w", forbidden)` gives 4, and a ServerTimeout gives 3
- [ ] 1.2 Add the no-response branch to `cmdutil.ExitCodeFromK8sError` in `internal/cmdutil/k8s.go`, reading only the typed chain, and update its doc comment
- [ ] 1.3 Delete `exitCodeFromK8sError` from `internal/workflow/apply/apply.go` and call `cmdutil.ExitCodeFromK8sError` at each call site in `apply.go` and `thineditor.go`
- [ ] 1.4 `task fmt`, `task lint` and `task test:unit` green, then commit `fix(cmd): exit 3 when a cluster request gets no response`

## 2. The record read in ResolveInventory

- [ ] 2.1 Replace `TestResolveInventory_RecordReadErrorExitsGeneral` in `internal/workflow/query/inventory_test.go` with a table over the name path and the uuid path (reactor on `get` and on `list` of `moduleinstances`): Forbidden gives 4, a `*url.Error` with a refused dial gives 3, an internal server error gives 1, and every row has `Printed` set
- [ ] 2.2 Map the record-read error in `query.ResolveInventory` (`internal/workflow/query/status.go`) through `cmdutil.ExitCodeFromK8sError`, set `Printed: true`, and update its doc comment
- [ ] 2.3 Change the exit code list in the help of `opm instance status` (`internal/cmd/instance/status.go`) so that 1 no longer covers every failed record read, and run `task docs:bundle:check`
- [ ] 2.4 Add an e2e test in `tests/e2e/` that writes a kubeconfig whose server is `https://127.0.0.1:1`, runs `opm instance status my-app -n default` with the repo's hermetic config, and asserts exit 3 and that the error appears once on stderr; it needs no kind cluster
- [ ] 2.5 `task fmt`, `task lint`, `task test:unit`, the new e2e test and `task openspec:check` green, then commit `fix(cmd): exit 4 when the instance record read is denied and print the error once`
