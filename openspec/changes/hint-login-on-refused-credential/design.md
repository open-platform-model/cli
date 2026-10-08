## Context

`config.BuildPlatformModule` wraps a failed platform acquire in a `DetailError` whose hint
`platformBuildHint` picks from the cause. The library's `Classify` types a failed registry
interaction as a `*FetchError` with a kind. Today every `*FetchError`, whatever its kind, and
every `*ResolutionError` gets the pin hint. `opm platform check` is the only caller and maps
every build failure to exit 2.

Publish and `module vet` map a refused credential to exit 4 with
`Log in to the registry, then retry:  opm registry login <host>`
(`cmdutil.CoreSchemaError`, `cmdutil.publishError`).

## Goals / Non-Goals

**Goals:**

- A refused credential on a platform build names the login command, with the host when it is
  certain, and exits 4 from `opm platform check`.
- No decision on the text of an error.
- Every other cause keeps its hint and exit code.

**Non-Goals:**

- A new hint for a registry that gives no response (it keeps the pin hint).
- The render path's platform build (`internal/workflow/render`), which has no hint.
- A cli-side reading of the token endpoint answers the library does not type as refusals.

## Research & Decisions

### How the refusal reaches the cli

**Context**: the hint must come from a typed cause.
**Explored**: a spike drove `BuildPlatformModule` against local registries with a cold cache
and read the library's classification (library v1.0.0-beta.7):

| Registry answer | Library kind | Status |
| --- | --- | --- |
| every request answers 401 | unauthorized | 401 |
| the archive blob answers 401 | unauthorized | 401 |
| the archive blob answers 403 | unauthorized | 403 |
| every request answers 403 (the tag lookup) | not found | 0 |
| token endpoint answers 403 | not found | 0 |
| token endpoint answers 401 | unreachable | 0 |

**Decision**: the cli MUST branch on `cuemod.IsUnauthorized(err)`, the existing reader of the
kind `FetchUnauthorized`. The last three rows keep the pin hint and exit 2, and a test pins
each as a known limit that fails when the library closes it.
**Rationale**: the reading of registry error text is the library's alone (0021:D8:R12).

### Which host the hint names

**Context**: publish knows the module it pushes, so it routes that one path to a host. A
platform build fails somewhere in a dependency graph; the library's `FetchError.Coordinate`
is empty for it (observed in the spike), and the host is only in the message text.
**Options considered**:

1. Read the host from the message text. Refused: a text match.
2. Route each dependency the platform's `cue.mod/module.cue` declares to its host and name
   the host when all agree. Not certain: the refused fetch may be a transitive dependency
   that routes elsewhere.
3. Name the host when the configured registry mapping holds exactly one host; otherwise print
   the bare `opm registry login`.

**Decision**: option 3.
**Rationale**: with one host the name is certain. The bare command resolves the same mapping
and, with several hosts, refuses and lists each as a runnable `opm registry login <host>`
line, so the user still reaches the right command and the cli never names a host the refusal
did not come from.

```go
// internal/config
func RegistryLoginHint(host string) string // "Log in to the registry, then retry:  opm registry login[ <host>]"
func soleRegistryHost(registry string) string // "" unless the mapping holds exactly one host; "+insecure" for plain HTTP
```

### Where the hint text lives

**Context**: `internal/cmdutil` imports `internal/config`, so `config` cannot call
`cmdutil`'s unexported `registryLoginHint`.
**Options considered**:

1. Repeat the text in `config`. Two texts for one thing drift.
2. Build the hint in `internal/cmd/platform` through `cmdutil`. The hint switch then lives in
   two packages.
3. Move the text to `config.RegistryLoginHint` and call it from `cmdutil.publishError`.

**Decision**: option 3. `cmdutil.CoreSchemaError` itself does not fit: it builds a registry
error for one named module and an operation text, and prints through another path than the
`DetailError` the platform build uses.
**Rationale**: one text, one switch. The publish and vet tests that pin the text stay green
unchanged.

### The exit code

**Context**: `opm platform check` maps every build failure to exit 2. The `DetailError`
wraps `ErrValidation`.
**Decision**: for a refused credential the `DetailError` wraps `oerrors.ErrPermission` in
place of `ErrValidation`, and `opm platform check` maps `errors.Is(err, ErrPermission)` to
exit 4 (`ExitPermissionDenied`). Every other build failure keeps `ErrValidation` and exit 2.
**Rationale**: the decision is made once, where the cause is read; the command reads a
sentinel. Exit 4 is the documented code for a refused credential on publish and `module vet`.

## Error output

```text
ERRO platform module does not build
Error: platform module error
  Location: /path/to/platform

  loading platform package from /path/to/platform (.): import failed: ...: cannot fetch example.com/dep@v0.2.0: module example.com/dep@v0.2.0: 401 Unauthorized: unauthorized: Unauthorized

Hint: Log in to the registry, then retry:  opm registry login ghcr.io
```

Exit codes of `opm platform check` for a build failure: 4 for a refused registry credential,
2 for every other cause.

## Risks / Trade-offs

- A script that read exit 2 as "bad credentials" sees 4. → Stated in the proposal and the PR
  body; the help names the code.
- A mapping with several hosts gives the bare command, one step more for the user. → The bare
  command lists the hosts as runnable lines.
- The three known limits still print the pin hint. → Pinned by tests that fail when the
  library types those answers as refusals.
