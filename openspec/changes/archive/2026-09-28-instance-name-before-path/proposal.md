## Why

The cluster-query commands (`opm instance status`, `tree`, `events`, `delete`) take one positional argument that is a path, a UUID or an instance name. The first detection rule treats the argument as a path whenever anything of that name exists in the working directory. `opm module init example.com/modules/hello@v0` creates `./hello`, so `opm instance status hello` next to it fails with "path …/hello is a module package, not an instance" instead of looking up the instance named `hello`. Worse, if `./hello` held an `instance.cue` for another instance, the command would act on that instance: `opm instance delete hello --force` would delete the wrong one. Found while testing the documentation quickstart, where the module directory and the instance share a name.

## What Changes

- The argument's shape alone decides whether it is a path: it ends in `.cue`, contains `/` or the OS path separator, or starts with `.` or `~`. What exists on disk no longer matters.
- A bare word is always an instance name or UUID. A user who means a directory writes `./hello`.
- `/` counts as a separator on every OS. The exists-on-disk rule used to cover `hello/instance.cue` on Windows, where the OS separator is `\`.

**BREAKING** for anyone who passed a bare directory name meaning a path, such as `opm instance status myapp` for `./myapp`: they now write `./myapp`. No help text or example used that form.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `inst-commands`: identifier auto-detection drops the exists-on-disk rule.

## Impact

- Code: `internal/cmdutil/instance_arg.go` (`isInstancePath`), with tests in `internal/cmdutil/instance_arg_test.go`.
- Commands: `opm instance status`, `tree`, `events`, `delete`.
