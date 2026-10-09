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
2. Name the host when the configured registry mapping holds exactly one host; otherwise print
   the bare `opm registry login`. First choice, dropped after review: CUE adds its central
   registry as the catch-all of every prefix mapping, so the cli's default mapping
   (`config.DefaultRegistry`) and every `prefix=host` mapping hold two hosts, and the default
   setup would never get a host.
3. Route each dependency the platform's `cue.mod/module.cue` declares to its host through the
   mapping and name the host when all agree; otherwise print the bare command.

**Decision**: option 3.
**Rationale**: a tidy CUE module file lists every module of the build, the indirect ones
included, so when all of them route to one host the refusal came from that host. A platform
on `opmodel.dev` modules under the default mapping names `ghcr.io`. Remaining limit: a module
file that is not tidy can miss the dependency that was refused; the hint then names the host
of the declared ones. With dependencies on several hosts, none declared, or a module file or
mapping that does not parse, the hint is the bare command, which resolves the mapping and
lists each host as a runnable `opm registry login <host>` line.

```go
// internal/config
func RegistryLoginHint(host string) string // "Log in to the registry, then retry:  opm registry login[ <host>]"
func platformRegistryHost(dir, registry string) string // the one host every declared dependency routes to, else ""; "+insecure" for plain HTTP
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

### How `opm platform check` prints the refusal

**Context**: found while implementing. The command prints a build failure through the
validation funnel. A failed import carries a CUE error with a source position, so the funnel
takes its grouped form and prints only `import failed` at `platform.cue:<line>:<col>`. The
registry's answer and the hint of the `DetailError` are not printed. That holds for every
failed import today, the unpublished pin included.
**Options considered**:

1. Print the hint after the grouped block for every cause. It changes the output of causes
   this change does not name.
2. Print a refused credential whole (the `DetailError`: location, the registry's answer, the
   hint) and leave every other cause on the funnel.

**Decision**: option 2. The command MUST print a refused credential with `output.Error` and
every other cause as before.
**Rationale**: the position of the import is not where the user fixes a refused credential;
the answer and the login command are. The hidden pin hint is recorded as a question for the
owner, and the command test records what an unpublished pin prints today.

## Error output

```text
ERRO platform module does not build
  error=
  | Error: platform module error
  |   Location: /path/to/platform
  |
  |   loading platform package from /path/to/platform (.): import failed: ...: cannot fetch example.com/dep@v0.2.0: module example.com/dep@v0.2.0: 401 Unauthorized: unauthorized: Unauthorized
  |
  | Hint: Log in to the registry, then retry:  opm registry login ghcr.io
```

Exit codes of `opm platform check` for a build failure: 4 for a refused registry credential,
2 for every other cause.

## Risks / Trade-offs

- A script that read exit 2 as "bad credentials" sees 4. → Stated in the proposal and the PR
  body; the help names the code.
- A platform whose declared dependencies route to several hosts gets the bare command, one
  step more for the user. → The bare command lists the hosts as runnable lines.
- The exit code now follows the library's classification, its text fallback included: a
  platform whose own CUE error text holds a registry refusal form would read as a refusal. →
  Low likelihood; the reading of error text is the library's alone.
- The three known limits still print the pin hint. → Pinned by tests that fail when the
  library types those answers as refusals.
