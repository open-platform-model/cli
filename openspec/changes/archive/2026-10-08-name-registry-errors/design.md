## Context

`config.Load` leaves `cfg.Registry` empty when no flag, `OPM_REGISTRY` or config value names one (`internal/config/loader.go`, `resolver.go`), and the kernel then resolves through `CUE_REGISTRY` from the process environment (`internal/config/kernel.go`, the library's `schema.OCILoader`). With nothing set, CUE asks its central registry, which does not hold `opmodel.dev/core`, and `module vet` and publish report "loading core schema" as exit 3.

`internal/publish/registry.go` wraps every `ModuleVersions` and `PutModule` error as `*ConnectivityError`, whose text is "registry unreachable". The cli already has a typed reader of the library's classification, `cuemod.IsConnectivityError` (`internal/cuemod/connectivity.go`), used by `instance init` and `instinit`.

## Goals / Non-Goals

**Goals:** name the missing registry and `opm config init`; call only "no response" unreachable in the publish lookup, push and schema fetch; name a refused credential and `opm registry login`; keep the cause in the text.

**Non-Goals:** a default registry; a check before the fetch; the compatibility walk (`compat.go`) and `catalog registry check` (`check.go`), whose answers `registryfailure_pin_test.go` and `registry_pin_test.go` pin; `module vet`'s classification of other schema failures; new exit codes for registry failures; the `--version` restore.

## Decisions

- **Report "no registry" only after the fetch failed.** MUST NOT refuse before the fetch: a run with nothing configured still succeeds today from a warm CUE module cache, and a run with only `CUE_REGISTRY` set resolves through it. The test is `registry == "" && CUE_REGISTRY == ""`, applied to the error of `SchemaCache().Get()`.
- **Exit 2 for the no-registry case.** The cli has no dedicated configuration exit code (`internal/exit/exit.go`: 1 general, 2 validation, 3 connectivity, 4 permission, 5 not found). `opm registry login` already refuses "no registry is configured" with exit 2 and the same `opm config init` action (`internal/cmd/registry/login.go`), so the same condition gets the same code.
- **One classifier in `internal/publish`.** `RegistryFailure(op, host, err)` returns `*ConnectivityError` when `cuemod.IsConnectivityError(err)`, else `*RegistryError{Unauthorized: cuemod.IsUnauthorized(err), Host: host}`. `cuemod.IsUnauthorized` is new and reads the same library classification (`FetchUnauthorized`). `ConnectivityError` keeps its type, text and every other construction site.
- **A refused credential exits 4; other registry failures keep 3.** The first build kept 3 for every class and left the code to the owner, who chose the existing code 4 (permission denied) for a 401 or 403. `publishError` maps a `*RegistryError` marked `Unauthorized` to 4, and any other `*RegistryError` to 3 like `*ConnectivityError`. The token-endpoint gap below still exits 3, because it is not recognised as a refusal.
- **Hints at the command layer.** `internal/publish` returns typed errors; `cmdutil` adds the `opm registry login <host>` line, as `operator install` adds its mirror hint. The host is the one the registry mapping routes the module to (`RegistryHost`), because the bare command refuses when the mapping names several hosts, which the default mapping does.

```go
// internal/config
func RegistryConfigured(registry string) bool
type NoRegistryError struct{ Op, ConfigPath string; ConfigExists bool; Err error }

// internal/cuemod
func IsUnauthorized(err error) bool

// internal/publish
type RegistryError struct{ Op string; Unauthorized bool; Host string; Err error }
func RegistryFailure(op, host string, err error) error
func RegistryHost(registry, modulePath string) string // host[+insecure] the mapping routes the module to, or ""

// internal/cmdutil
func NoRegistryError(cfg *config.GlobalConfig, op string, err error) error // the exit-2 error, or nil when a registry is configured
```

Messages:

```text
no registry is configured: loading core schema: <cause>
  Set one with --registry, OPM_REGISTRY or the registry field of <config path>.
  To write a config file with the default registry, run:  opm config init
  (when the config file exists: add a registry field to it, or replace the whole file with  opm config init --force)

registry refused the credentials (authentication or permission): pushing <repo>:<tag>: <cause>
  Log in to the registry, then retry:  opm registry login <host>

registry operation failed: listing published versions of <path>: <cause>
registry unreachable: listing published versions of <path>: <cause>
```

## Research & Decisions

### Where the no-registry test runs

**Context**: every command builds its kernel with `config.NewKernel(cfg.Registry)`; only `module vet` and the publish commands load the core schema through `SchemaCache().Get()`.
**Explored**: `internal/config/kernel.go`, the library's `kernel.New` and `schema.OCILoader`.
**Options considered**:
1. Refuse in `config.Load` or `NewKernel` when the registry is empty - one place, but it breaks runs that work today (warm cache, `CUE_REGISTRY` only) and commands that need no registry.
2. Decorate the failed schema fetch at the two call sites - narrow, changes no successful run.
**Decision**: option 2.
**Rationale**: the change must not alter a run that succeeds today, the `CUE_REGISTRY` path in particular.

### Exit code of a refused credential

**Options considered**: keep 3; move to 4 (`ExitPermissionDenied`); move to 1.
**Decision**: 4, by owner decision (the first build kept 3 and asked).
**Rationale**: 4 is the cli's code for a permission failure, and a script can then tell a credential problem from a network problem.

## Risks / Trade-offs

- **Known gap: a refusing token endpoint.** A registry that hands out bearer tokens and whose token endpoint answers 403 lets the lookup pass (CUE reads a 403 there as not found) and fails the push with "cannot do HTTP request: ...: 403 Forbidden". The library's text classification reads the prefix as no response, so the push is still "registry unreachable". The cli MUST NOT read registry error text itself (the library owns the one text fallback), so the fix is a library change and a pin bump. `TestPush_TokenEndpointRefusal_Pinned` holds the present answer and fails when the library closes the gap. A token endpoint that answers 401 is named correctly.
- A 403 on the already-published lookup returns no error (not found), so the refused-credentials message shows on the push, not on a dry run.

- A push error reaches the classifier as flattened text ("cannot make scratch config: 401 Unauthorized: ..."), so it classifies through the library's one text fallback → tests drive a real registry answer for both 401 and 403 on the push.
- With nothing configured and a warm cache for core only, vet passes the schema fetch and later fails on a catalog fetch with CUE's own error → not covered here; reported as a follow-up.
- A script that matched the text "registry unreachable" for a 5xx answer now sees "registry operation failed" → the exit code is unchanged.
