## Context

`config.Load` leaves `cfg.Registry` empty when no flag, `OPM_REGISTRY` or config value names one (`internal/config/loader.go`, `resolver.go`), and the kernel then resolves through `CUE_REGISTRY` from the process environment (`internal/config/kernel.go`, the library's `schema.OCILoader`). With nothing set, CUE asks its central registry, which does not hold `opmodel.dev/core`, and `module vet` and publish report "loading core schema" as exit 3.

`internal/publish/registry.go` wraps every `ModuleVersions` and `PutModule` error as `*ConnectivityError`, whose text is "registry unreachable". The cli already has a typed reader of the library's classification, `cuemod.IsConnectivityError` (`internal/cuemod/connectivity.go`), used by `instance init` and `instinit`.

## Goals / Non-Goals

**Goals:** name the missing registry and `opm config init`; call only "no response" unreachable in the publish lookup, push and schema fetch; name a refused credential and `opm registry login`; keep the cause in the text.

**Non-Goals:** a default registry; a check before the fetch; the compatibility walk (`compat.go`) and `catalog registry check` (`check.go`), whose answers `registryfailure_pin_test.go` and `registry_pin_test.go` pin; `module vet`'s classification of other schema failures; new exit codes for registry failures; the `--version` restore.

## Decisions

- **Report "no registry" only after the fetch failed.** MUST NOT refuse before the fetch: a run with nothing configured still succeeds today from a warm CUE module cache, and a run with only `CUE_REGISTRY` set resolves through it. The test is `registry == "" && CUE_REGISTRY == ""`, applied to the error of `SchemaCache().Get()`.
- **Exit 2 for the no-registry case.** The cli has no dedicated configuration exit code (`internal/exit/exit.go`: 1 general, 2 validation, 3 connectivity, 4 permission, 5 not found). `opm registry login` already refuses "no registry is configured" with exit 2 and the same `opm config init` action (`internal/cmd/registry/login.go`), so the same condition gets the same code.
- **One classifier in `internal/publish`.** `registryFailure(op, err)` returns `*ConnectivityError` when `cuemod.IsConnectivityError(err)`, else `*RegistryError{Unauthorized: cuemod.IsUnauthorized(err)}`. `cuemod.IsUnauthorized` is new and reads the same library classification (`FetchUnauthorized`). `ConnectivityError` keeps its type, text and every other construction site.
- **Exit 3 stays for every registry failure on publish.** A 401 could map to the existing code 4 (permission denied), but that is a change to the documented exit contract and is left to the owner. `publishError` maps `*RegistryError` to 3 like `*ConnectivityError`.
- **Hints at the command layer.** `internal/publish` returns typed errors; `cmdutil` adds the `opm registry login` line, as `operator install` adds its mirror hint.

```go
// internal/config
func RegistryConfigured(registry string) bool
type NoRegistryError struct{ Op, ConfigPath string; Err error }

// internal/cuemod
func IsUnauthorized(err error) bool

// internal/publish
type RegistryError struct{ Op string; Unauthorized bool; Err error }
func RegistryFailure(op string, err error) error

// internal/cmdutil
func CoreSchemaError(cfg *config.GlobalConfig, err error) (error, bool) // the no-registry exit error, when it applies
```

Messages:

```text
no registry is configured: loading core schema: <cause>
  Set one with --registry, OPM_REGISTRY or the registry field of <config path>.
  To write a config file with the default registry, run:  opm config init

registry refused the credentials (authentication or permission): pushing <repo>:<tag>: <cause>
  Log in to the registry, then retry:  opm registry login

registry operation failed: listing published versions of <path>: <cause>
registry unreachable: listing published versions of <path>: <cause>
```

## Research & Decisions

### Where the no-registry test runs

**Context**: every command builds its kernel with `config.NewKernel(cfg.Registry)`; only `module vet` and the publish commands load the core schema through `SchemaCache().Get()` (`internal/config/platform.go` holds a third call that is not on a command path changed here).
**Explored**: `internal/config/kernel.go`, the library's `kernel.New` and `schema.OCILoader`.
**Options considered**:
1. Refuse in `config.Load` or `NewKernel` when the registry is empty - one place, but it breaks runs that work today (warm cache, `CUE_REGISTRY` only) and commands that need no registry.
2. Decorate the failed schema fetch at the two call sites - narrow, changes no successful run.
**Decision**: option 2.
**Rationale**: the change must not alter a run that succeeds today, the `CUE_REGISTRY` path in particular.

### Exit code of a refused credential

**Options considered**: keep 3; move to 4 (`ExitPermissionDenied`); move to 1.
**Decision**: keep 3, ask the owner about 4.
**Rationale**: the label and the next step are the defect the task names; the exit table is a published contract.

## Risks / Trade-offs

- A push error reaches the classifier as flattened text ("cannot make scratch config: 401 Unauthorized: ..."), so it classifies through the library's one text fallback → tests drive a real registry answer for both 401 and 403 on the push.
- With nothing configured and a warm cache for core only, vet passes the schema fetch and later fails on a catalog fetch with CUE's own error → not covered here; reported as a follow-up.
- A script that matched the text "registry unreachable" for a 5xx answer now sees "registry operation failed" → the exit code is unchanged.
