## Context

The six `mod` commands (apply, build, vet, delete, diff, status) total 1,530 lines across their implementation files. Detailed analysis reveals ~337 lines of duplicated orchestration code distributed across 8 distinct patterns (flag declarations, module path resolution, K8s config resolution, OPM config retrieval, render pipeline execution, error formatting, client creation, and release selector validation). The duplication has already caused subtle inconsistencies — diff omits the `HasErrors()` check that the other render commands perform, and delete/status access the package-level `opmConfig` directly while build/vet/apply use `GetOPMConfig()`.

The existing code lives entirely in `package cmd` (`internal/cmd/`). Shared helpers (`writeTransformerMatches`, `printValidationError`, etc.) are defined in `mod_build.go` and used by other command files. `ExitError` is also defined in `mod_build.go` despite being used everywhere. Tests are package-internal and test via cobra's public API (create command, set args, execute), so internal restructuring is safe as long as behavior is preserved.

## Goals / Non-Goals

**Goals:**

- Eliminate duplicated orchestration code across mod commands
- Establish composable, struct-based flag groups that replace 46 package-level flag variables
- Create a reusable render pipeline helper that handles the 10-step preamble shared by render-based commands
- Standardize K8s client creation and error handling across cluster-interacting commands
- Make each command file contain only its unique logic, with shared patterns in `cmdutil`
- Preserve exact behavioral equivalence — same flags, exit codes, output, error messages

**Non-Goals:**

- Changing any user-facing CLI behavior (flags, output format, error messages, exit codes)
- Refactoring `config` commands (config init, config vet) — only `mod` commands
- Creating a generic command framework — this is targeted deduplication, not a framework
- Adding new features or flags during this refactoring
- Changing the `internal/build`, `internal/kubernetes`, or `internal/output` packages

## Decisions

### Decision 1: New `internal/cmdutil` package (not extending `internal/cmd`)

Create a new `internal/cmdutil` package rather than adding helper files within `internal/cmd`.

**Rationale:** Enforces a clean dependency direction — `cmd` imports `cmdutil`, never the reverse. The `cmdutil` package accepts its dependencies as parameters (OPMConfig, registry, verbose flag) rather than reaching into `cmd`'s package-level state. This makes helpers independently testable without cobra command scaffolding.

**Alternative considered:** Add `cmd/helpers.go` and `cmd/flag_groups.go` within the existing package. Rejected because it doesn't prevent helpers from coupling to package-level state (`opmConfig`, `verboseFlag`), and testing requires the full cobra root command setup.

### Decision 2: Struct-based flag groups with `AddTo(*cobra.Command)` method

Replace per-command flag variable blocks with composable structs:

```text
┌─────────────────────────────────────────────────────┐
│ Flag Group Structs                                  │
├──────────────────┬──────────────────────────────────┤
│ RenderFlags      │ Values, Namespace, ReleaseName,  │
│                  │ Provider                         │
│                  │ Used by: apply, build, vet, diff │
├──────────────────┼──────────────────────────────────┤
│ K8sFlags         │ Kubeconfig, Context              │
│                  │ Used by: apply, diff, delete,    │
│                  │          status                  │
├──────────────────┼──────────────────────────────────┤
│ ReleaseSelectorFlags │ ReleaseName, ReleaseID,      │
│                  │ Namespace                        │
│                  │ Used by: delete, status           │
│                  │ Has: Validate(), LogName()       │
└──────────────────┴──────────────────────────────────┘
```

Each struct has an `AddTo(cmd *cobra.Command)` method that registers flags with consistent names and descriptions. Commands compose by embedding the structs they need.

**Rationale:** Eliminates 46 package-level variables. Flag names and descriptions are defined once. Type-safe — compiler catches misspellings. Each command's `New*Cmd()` function instantiates only the flag groups it uses, and the `RunE` closure captures them.

**Alternative considered:** Continue using package-level vars with a naming convention. Rejected because it doesn't reduce line count and the `{cmd}` prefix pattern is error-prone.

### Decision 3: Two-phase render helper (RenderModule + ShowRenderOutput)

Split the render pipeline helper into two functions rather than one monolithic function with boolean options:

```text
RenderModule()                    ShowRenderOutput()
├── Resolve module path           ├── Check HasErrors() → printRenderErrors
├── Get/validate OPM config       ├── Show transformer matches (verbose/default)
├── Resolve K8s config            └── Log warnings
├── Build RenderOptions
├── Validate options
├── Create pipeline
├── Execute Render()
└── Handle render errors
    (printValidationError)
```

- `RenderModule(ctx, opts) → (*build.RenderResult, error)` — executes the pipeline, returns result or ExitError
- `ShowRenderOutput(result, opts)` — checks for errors, shows transformer matches, logs warnings

**Rationale:** The diff command needs `RenderModule` but handles post-render differently (it passes partial results to `DiffPartial` instead of failing on `HasErrors()`). Two functions let diff call only `RenderModule`, while build/vet/apply call both. This avoids boolean flags like `SkipErrorCheck` that obscure intent.

**Alternative considered:** Single `RenderModule` function with `PostProcess bool` option. Rejected because a boolean hides the semantic difference — diff isn't "skipping" post-processing, it's doing different post-processing.

### Decision 4: K8s client factory as a simple function

```text
NewK8sClient(opts) → (*kubernetes.Client, error)
├── Read API warnings from OPMConfig
├── Create kubernetes.NewClient
└── On error: return ExitError{Code: ExitConnectivityError}
```

No builder pattern — just a function that creates a client or returns an ExitError. The caller provides a logger for the error message (since delete/status use release-scoped loggers while apply/diff use module-scoped loggers).

**Rationale:** The 4 commands with K8s clients have character-identical creation code. A simple function eliminates this without adding abstraction. The logger parameter handles the only variation (different `modLog` sources).

### Decision 5: Move ExitError to `exit.go`, shared formatters to `cmdutil`

- `ExitError` struct moves from `mod_build.go` to `exit.go` (same package, alongside exit code constants).
- Orchestration helpers that combine output + error handling (`printValidationError`, `printRenderErrors`) move to `cmdutil` since they are part of the render pipeline orchestration.
- Pure output formatters (`writeTransformerMatches`, `writeVerboseMatchLog`, `writeBuildVerboseJSON`, `formatApplySummary`) stay in `internal/cmd` or move to `internal/output` since they are presentation-only.

**Rationale:** Each function moves to where its primary concern lives. `ExitError` is an error type used by all commands — it belongs with exit codes. `printValidationError` is orchestration glue (format + error handling) — it belongs in `cmdutil`. `writeTransformerMatches` is pure output — it belongs in the output layer.

### Decision 6: Commands compose flag groups, not inherit from a base command

Each command's `New*Cmd()` function declares which flag groups it uses as local variables. There is no `BaseModCommand` struct or inheritance hierarchy.

**Example composition for mod vet:**

```text
NewModVetCmd()
├── var rf cmdutil.RenderFlags      ← declares render flags
├── cmd := &cobra.Command{...}
├── rf.AddTo(cmd)                   ← registers flags on command
└── RunE closure captures rf        ← uses flags in execution
```

**Example composition for mod apply (more flags):**

```text
NewModApplyCmd()
├── var rf cmdutil.RenderFlags      ← render flags
├── var kf cmdutil.K8sFlags         ← k8s connection flags
├── var dryRun, wait bool           ← command-specific flags (stay local)
├── cmd := &cobra.Command{...}
├── rf.AddTo(cmd)
├── kf.AddTo(cmd)
├── cmd.Flags().BoolVar(&dryRun...) ← register command-specific flags normally
└── RunE closure captures rf, kf, dryRun, wait
```

**Rationale:** Composition over inheritance. Each command explicitly declares its dependencies. No hidden behavior from a base struct. New commands can compose any subset of flag groups.

## Risks / Trade-offs

**[Risk] Behavioral regression in refactored commands** → All existing tests must pass after refactoring. Run `task test` after each command migration. The tests use cobra's public API (create, set args, execute) so they validate end-to-end behavior regardless of internal structure.

**[Risk] Circular dependency between `cmd` and `cmdutil`** → Strict rule: `cmdutil` MUST NOT import `cmd`. All dependencies flow one direction: `cmd → cmdutil → {build, kubernetes, output, config}`. The `cmdutil` helpers accept `*config.OPMConfig` and `string` registry as parameters rather than calling `GetOPMConfig()` or `GetRegistry()`.

**[Risk] Package-level state in `cmd` becomes harder to manage** → The `opmConfig` and `verboseFlag` package-level vars remain in `cmd/root.go` (set by `PersistentPreRunE`). `cmdutil` functions accept these as explicit parameters. This is a trade-off: explicit parameter passing is verbose but eliminates hidden coupling.

**[Risk] `mod_build.go` shrinks significantly but still has build-specific helpers** → After extracting shared helpers, `mod_build.go` will still contain `writeVerboseOutput`, `writeBuildVerboseJSON`, `formatApplySummary`, and split/output logic. This is correct — those are build-specific concerns.

**[Trade-off] Two function calls instead of one for render commands** → build/vet/apply call both `cmdutil.RenderModule()` and `cmdutil.ShowRenderOutput()` instead of a single function. This adds one line per command but makes the diff exception clean and the code more readable.

**[Trade-off] `cmdutil` adds a new package to the dependency graph** → Justified by Principle VII: the package eliminates ~337 lines of duplication across 6 commands. The new package is ~200 lines total, a net reduction of ~137 lines with significantly better maintainability.

## Migration Plan

Migrate one command at a time in a single branch, running tests after each:

1. Create `internal/cmdutil/` with flag structs, render helpers, k8s factory, utility functions
2. Move `ExitError` from `mod_build.go` to `exit.go`
3. Extract shared formatters from `mod_build.go` to appropriate locations
4. Migrate `mod vet` first (simplest render command, good smoke test)
5. Migrate `mod build` (adds output-format complexity)
6. Migrate `mod apply` (adds K8s client + operation flags)
7. Migrate `mod diff` (tests the two-phase render split)
8. Migrate `mod delete` (tests ReleaseSelectorFlags)
9. Migrate `mod status` (tests ReleaseSelectorFlags + watch)
10. Remove dead code (orphaned package-level flag vars, unused helpers)
11. Run full test suite: `task check` (fmt + vet + test)

**Rollback:** Since this is a single PR with no user-facing changes, rollback is simply reverting the PR. No data migration, no config changes, no API versioning needed.
