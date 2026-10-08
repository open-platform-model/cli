## Context

See proposal.md for the motivation. The facts that shape the approach, read at the base
commit 7b6962f2:

- `--force` is defined on five commands. `instance/delete.go` uses it to skip the prompt.
  `instance/apply.go` and `module/apply.go` (empty render prunes everything),
  `platform/pull.go` (replace a non-empty directory) and `config/init.go` (overwrite the
  config) use it to override a refusal. `module/init.go` already has `--yes`/`-y`.
- `module vet` reads `--instance-name` from the shared `cmdutil.RenderFlags`.
  `module build` and `module apply` register the same shared flag, hide it and ignore it, and
  read `--name` in its place.
- `cmd/opm/main.go` exits with `ExitError.Code`, or 1 for any other error. Cobra's own
  errors (unknown command, unknown flag, argument count) are not `ExitError`, so they exit 1.
- The main specs state 1 for a usage error in five places: `deploy` ("Exit Codes" table),
  `mod-vet` ("mod vet exit codes"), `inst-commands` ("instance build without file
  argument"), `cmd-structure` ("Flag rejected by instance build") and `status-exit-codes`
  ("Status exits with code 1 when the check cannot run"). Code 2 is taken: it is "validation
  error or refusal", and for `opm instance status` it is "resources not ready".
- The command reference is generated from the cobra tree when the docs bundle is built;
  nothing generated is committed.

## Goals / Non-Goals

**Goals:**

- One meaning per shared flag, with every old spelling still accepted.
- The exit code of a usage error readable by a user and pinned at the process boundary.
- `-o`/`--output` where a read command prints a fixed, small data shape.

**Non-Goals:**

- Moving the usage-error exit code. See "Research & Decisions".
- `-o` on `platform check`, `catalog registry check`, `module eval`, `instance diff` and the
  `vet` commands. See "Research & Decisions".
- A TTY check on the delete prompt, a usage hint under an argument-count error, the `-f`
  shorthand of `config init --force`, colour, signals: none is named by the task.
- Any change under `internal/kubernetes` or `internal/workflow`.

## Decisions

### Deprecated aliases use cobra's own mechanism

A deprecated flag is a second flag registered on the command and marked with
`Flags().MarkDeprecated(name, "use --new")`. Cobra then hides it from help and from
completion, and prints `Flag --old has been deprecated, use --new` on standard error when it
is used. No home-made warning code.

```go
// instance delete
c.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip the confirmation prompt")
c.Flags().BoolVar(&forceFlag, "force", false, "Skip the confirmation prompt")
_ = c.Flags().MarkDeprecated("force", "use --yes")
// RunE: skipPrompt := yesFlag || forceFlag
```

```go
// module vet
c.Flags().StringVar(&nameFlag, "name", "", "Synthetic instance name (default: <module name>-debug)")
_ = c.Flags().MarkDeprecated("instance-name", "use --name")
c.MarkFlagsMutuallyExclusive("name", "instance-name")
// RunE: if nameFlag != "" { rf.InstanceName = nameFlag }
```

`cmdutil.RenderFlags` is not changed: `module build` and `module apply` keep hiding and
ignoring `--instance-name`, which the task does not name.

### Output formats are parsed in the command package

`opm version` and `opm module template list` each validate `--output` against their own
list and marshal a small struct with `encoding/json` (indented) or `sigs.k8s.io/yaml`,
both already dependencies. The error text follows the existing commands:
`invalid output format "<v>" (valid: ...)`, exit 1 (`ExitGeneralError`), as
`cmd-structure` and `status-exit-codes` specify for the same mistake.

### The entry point returns its exit code

`main` becomes `os.Exit(run(os.Args[1:], os.Stderr))`. `run` builds the root command,
executes it and maps the error exactly as today. A test in `cmd/opm` calls `run` and
asserts the code, which is the only place the code of a non-`ExitError` is decided.

### Help layout is tested over the whole tree

One test walks `NewRootCmd()` and fails on any `Long` line that starts with a tab. The fix
for each hit is whitespace only.

## Research & Decisions

### Exit code of a usage error

**Context**: The task says "usage errors on the documented exit code", and its intent line
says 2 is documented.
**Explored**: `internal/exit/exit.go`, `cmd/opm/main.go`, every main spec that mentions an
exit code, `internal/cmd/instance/instance_test.go:84-96`.
**Options considered**:
1. Move usage errors to 2. Matches the common Unix habit. But 2 already means "validation
   error or refusal" everywhere and "resources not ready" on `opm instance status`, so a
   pipeline that polls `status` and treats 2 as "not ready yet" would retry forever on a
   typo. It also changes five main specs and a contract, which this change cannot decide.
2. Keep 1, document it where a user reads it, pin it at the process boundary.
**Decision**: Option 2.
**Rationale**: The specs name 1, the code exits 1, so "the docs and the code agree" holds
once the table is in `opm --help` and a test pins it. Whether to move it is a question for
the owner, recorded in the swarm report.

### Which read commands gain `-o`

**Context**: The audit lists `instance diff`, `instance vet`, `module vet`, `platform check`,
`module eval` and `version` as lacking `-o`.
**Options considered**:
1. Add `-o` to all of them.
2. Add it where the data shape is small, local and stable; leave the rest with a reason.
**Decision**: Option 2. `version` and `module template list` gain it.
**Rationale**:
- `platform check` and `catalog registry check`: the report carries library row types and
  three verdicts. A JSON form is a new public schema over library types; it needs its own
  design.
- `module eval`: prints CUE source, and a value that is not concrete has no JSON form.
- `instance diff`: the diff is built and printed under `internal/kubernetes`, out of scope.
- `instance vet`, `module vet`, `config vet`: they print a verdict as log lines; the exit
  code is the machine contract.

## Risks / Trade-offs

- [A script parses standard error of `instance delete --force` or `module vet
  --instance-name`] → one extra line appears there. Standard output and the exit code do not
  change.
- [`-y` collides with a later flag on `instance delete`] → `module init` already owns
  `-y` for the same meaning, so the shorthand is taken on purpose.
- [The JSON field names of `version` become a contract at v1.0.0] → they follow the
  camelCase of the existing JSON outputs (`instanceName`, `lastSeen`).
