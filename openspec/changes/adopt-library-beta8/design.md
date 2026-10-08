## Context

The cli reads every registry failure through the library's typed classification (`liberrors.Classify`, read by `internal/cuemod/connectivity.go`). It holds no text match of its own for a registry answer (0021:D8:R12). So a library release that types one more form moves the cli's answer with no code edit; only the tests that pin the old answer fail. Library v1.0.0-beta.8 carries three changes: library#222 (token endpoint refusal), library#223 (public surface, `feat!`) and library#227 (unset required values).

## Goals / Non-Goals

**Goals:**

- Land the pin bump with every test stating the answer the cli now gives.
- State, per beta.8 change, what a cli user saw before and sees now.

**Non-Goals:**

- No new hint or exit code where the cli has none today (`opm instance init`, `opm catalog registry check`, the compatibility walk).
- No fix for a 403 that the registry client reports as not found.

## Decisions

The bump MUST change `go.mod` and `go.sum` only. The cli MUST NOT add a text match to tell a token endpoint refusal apart; the library's kind is the only input.

### Command syntax, flags, exit codes

No command, flag or output format changes. Exit codes move only for runs that already failed:

| Command | Case | Old exit | New exit |
| --- | --- | --- | --- |
| `opm module publish <path>`, `opm catalog publish <path>` | the token endpoint answers 403 to the push | 3 | 4 |
| `opm platform check [dir]` | the token endpoint answers 401 while the platform's imports resolve | 2 | 4 |
| `opm instance init <module>` | the token endpoint answers 401 while the staged package's dependencies resolve | 3 | 1 |

Example, the push. The first line is the error of `TestPush_TokenEndpointRefusal`; the second is the hint `publishError` appends for that error type (`TestPublishError_LoginHint`):

```text
registry refused the credentials (authentication or permission): pushing example.com/modules/demo:v1.2.0: cannot make scratch config: cannot do HTTP request: Post "http://127.0.0.1:41747/v2/example.com/modules/demo/blobs/uploads/": 403 Forbidden
Log in to the registry, then retry:  opm registry login 127.0.0.1:41747+insecure
```

Example, a render with two unset required values (from `TestFromModule_UnsetRequiredValuesAreNamed`):

```text
ERRO render failed: 2 issues

incomplete value string
  values.note
    > module.cue:20:12

incomplete value int
  values.other
    > module.cue:21:12
```

The same module on library v1.0.0-beta.7 printed one issue, at the component field that reads `note`, and did not name `other`:

```text
ERRO render failed: 1 issue

incomplete value string
  values.components.foo.metadata.annotations.note
    > module.cue:20:12
    > module_instance.cue:103:12
```

## Research & Decisions

### How the pin moves

**Context**: The repo's task for a library bump is `task -x deps:cascade`. It also resolves the catalog, core and operator module pins against GHCR.
**Explored**: `git diff origin/main...origin/deps/cascade` in the main checkout: `go.mod` and `go.sum` only, library beta.7 to beta.8.
**Options considered**:
1. `task -x deps:cascade` - the repo's own path, but it reads GHCR and can move other pins that were published since.
2. `go get github.com/open-platform-model/library@v1.0.0-beta.8` and `go mod tidy` - the library lane only.
**Decision**: Option 2, then compare the result with the cascade branch.
**Rationale**: The work is limited to Go module downloads. The result is byte-identical to the cascade branch (`git diff origin/deps/cascade -- go.mod go.sum` is empty), so the cascade will find nothing left to do for the library.

### Why `Unit Tests` fails on cli#355

**Explored**: `task test:unit` on the bumped tree with no other edit.
**Finding**: two tests fail, both gap pins that say so in their failure message:
- `TestPush_TokenEndpointRefusal_Pinned` (`internal/publish`): expects `*ConnectivityError`, gets `*RegistryError` marked `Unauthorized` ("the gap closed: assert *RegistryError with Unauthorized instead").
- `TestPlatformBuildHint_RefusalNotTypedAsOne_Pinned/token_endpoint_answers_401` (`internal/config`): expects the validation cause and the pin hint, gets the permission cause ("the gap closed: move the row to TestPlatformBuildHint_RefusedCredential").

Every other package passes. `go build ./...` and `go vet ./...` pass on the bumped tree.

### Which token endpoint cases change

**Explored**: `cuemodtest.TokenRegistry` with 401 and 403, driven through each cli path on beta.7 (`go test -modfile` with the old `go.mod`) and beta.8.

| Path | Token answer | beta.7 | beta.8 |
| --- | --- | --- | --- |
| push | 403 | unreachable | unauthorized |
| directory load with a dependency (platform build) | 401 | unreachable | unauthorized |
| tidy (`opm instance init` write) | 401 | unreachable | unauthorized |
| direct fetch (schema fetch of vet and publish, module acquire) | 401 | unauthorized | unauthorized |
| any fetch | 403 | not found | not found |

**Decision**: pin all five rows. Where the cli has a login hint (publish, vet, platform build) the new answer is the hint and exit 4. `opm instance init` has no branch for a refused credential: a registry that answered exits 1 by its own spec ("Only a registry that gave no response counts as unreachable"), so the token 401 joins that rule. A hint there is a new behaviour and not part of a pin bump.

### The 403 that stays

**Context**: The brief for this work asks for the login hint on a 401 or 403.
**Finding**: a 403 from the token endpoint on a fetch reaches the cli as `FetchNotFound`, because the registry client reports it as `modregistry.ErrNotFound`. The library decided to keep that (library ADR-014: "a fetch whose token endpoint answers 403 stays not found").
**Decision**: keep those rows as gap pins (`TestPlatformBuildHint_RefusalNotTypedAsOne_Pinned`, the vet and publish schema-fetch rows) and say so in the specs and the publish page. The cli MUST NOT guess a refusal from a not-found answer.

### library#223, the public surface

**Explored**: every name the release removes or deprecates, searched in all Go files of the cli: `opm/helper/objectset`, `schema.ModuleMetadata`, `schema.InstanceMetadata`, `schema.PlatformMetadata`, `schema.CatalogMetadata`, the twelve moved `schema` paths, `schema.CollisionsSince`, `platform.Source` (library), `catalog.Source` (deprecated), the value forms of `errors.IdentityError` (deprecated).
**Finding**: no use. `task lint` (staticcheck SA1019 on) reports 0 issues with no new exclusion. `Kernel.ValidateConfigDetailed` now returns `*errors.ConfigValidationError` around the unchanged CUE tree; the cli wraps it with `%w` and reads the CUE tree through `errors.As`, and its tests pass unchanged.
**Decision**: no edit.

### library#227, unset required values

**Explored**: the cli's tests and docs for a quoted refusal: `tests/e2e/vet_output_test.go` (substrings `incomplete value int`, `values.replicas`, `incomplete value`), `internal/publish/identity_test.go` (`incomplete value`). No doc page quotes the refusal.
**Finding**: all hold. `opm module vet` and a render with `-f` files check the values against `#config` first (`ValidateConfigDetailed`), which always named the field; that path does not change. The kernel refusal is what a render prints when the values come from `debugValues` or from an instance package (`opm instance build`, `vet`, `apply`, `diff`).
**Decision**: add one test that pins the printed findings on the module path. The instance commands call the same kernel check through `AcquireInstanceFromDir` and print through the same function (`printValidationError`); the repo has no published fixture module with a required value to drive them offline.

## Risks / Trade-offs

- A script that reads exit 3 from `opm instance init` for a refused token now gets 1. → The old code was wrong by the command's own spec; the registry's answer is in the message.
- The 403 limit stays visible to users as "not found" or the pin hint. → Stated in the specs and the publish page; the fix is the registry client's.
