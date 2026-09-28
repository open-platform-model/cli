## 1. Detect a path by its shape only

- [x] 1.1 Remove the exists-on-disk rule from `isInstancePath` in `internal/cmdutil/instance_arg.go`, and count `/` as a separator alongside the OS path separator
- [x] 1.2 Add tests in `internal/cmdutil/instance_arg_test.go`: a forward-slash path is a path, and a bare word matching a directory in the working directory resolves as a name while `./<word>` resolves as a path; verify they pass
- [x] 1.3 `task check` green, then commit `fix(instance): read a bare argument as an instance name`
