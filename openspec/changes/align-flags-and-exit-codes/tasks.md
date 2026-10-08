## 1. Confirmation and name flags

- [x] 1.1 `internal/cmd/instance/delete.go`: add `--yes`/`-y`, keep `--force` as a deprecated alias through cobra's `MarkDeprecated`, update the help text and the example; a test in `delete_test.go` fails before the change (no `yes` flag, `force` not deprecated) and passes after
- [x] 1.2 `internal/cmd/module/vet.go`: add `--name`, mark `--instance-name` deprecated, refuse both together; tests in `vet_test.go` fail before and pass after (flag present, alias sets the same name, both together is a usage error)
- [x] 1.3 A test over the command tree pins the `--force` and `--yes` conventions: `--force` is offered, not deprecated, on `instance apply`, `module apply`, `platform pull` and `config init`; `--yes` with `-y` is offered on `instance delete` and `module init`
- [x] 1.4 `task fmt`, `task vet`, `task lint` and `task test:unit` green, then commit `feat(cmd): add --yes to instance delete and --name to module vet`

## 2. Exit code of a usage error and help layout

- [x] 2.1 Move the body of `main` into `cmd.Run(args, stderr) int` in `internal/cmd/run.go` with unchanged mapping, so the tests run in CI (`./internal/...`); `internal/cmd/run_test.go` asserts exit 1 for an unknown command, an unknown flag, a missing argument, a bad flag value and excluding flags, and that the error is on standard error
- [x] 2.2 `internal/cmd/root.go`: add the `Exit codes` table to the root long description; a test in `root_test.go` fails before and passes after
- [x] 2.3 Add a test in `internal/cmd` that walks the command tree and fails on a long description line that begins with a tab; fix every hit (the `catalog` and `registry` groups at least) with whitespace-only edits
- [x] 2.4 `task fmt`, `task vet`, `task lint` and `task test:unit` green, then commit `fix(cmd): document the exit codes in the root help and remove tab indentation from help text`

## 3. Output formats on version and template list

- [x] 3.1 `internal/cmd/version.go`: add `-o`/`--output` (`text`, `json`, `yaml`), writing to the command's standard output; tests in `version_test.go` fail before and pass after (JSON fields, default text unchanged, invalid value exits 1 with the message)
- [x] 3.2 `internal/cmd/module/template.go`: add `-o`/`--output` (`table`, `json`, `yaml`); tests fail before and pass after
- [x] 3.3 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `feat(cmd): add --output to version and module template list`
