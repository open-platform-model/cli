# Design: render-local-replacements (cli)

## Context

See `proposal.md` § Why. Today's pieces:

- `internal/workflow/render/render.go` `renderInstance` is the one `Kernel.Render` call. `RenderInput` carries instance, platform, `RuntimeName` and the skew policy.
- The D19 warning is one constant (`localReplacementWarning`) emitted by `warnLocalReplacement` when `loader.HasLocalModuleReplacement(moduleRoot)` is true: a file-presence test on the module context, never on the platform directory.
- `pkg/loader/provenance.go` parses `local-module.cue` with `CompileBytes` to answer "any local `replaceWith`?"; `internal/publish/gates.go` `readReplacements` parses it again, beside `module.cue`'s pins, for the publish refusal.
- The platform reaches a render as a directory: `--platform <dir>`, the local default platform module beside the config, or a module generated from the cluster CR (which never carries the file).
- The library change of the same name adds `RenderInput.LocalReplacements bool` and `RenderDiagnostics.Replacements []kernel.Replacement{Path, Target, By}` (path-sorted, `By` is `"platform"` or `"instance"`), and refuses an input carrying the file when the flag is off.

## Goals / Non-Goals

**Goals:**

- A developer's `local-module.cue` in a module, instance or platform directory changes what `opm` renders, and every warning states a fact the kernel reported.
- No new flag, no new exit code, no change to provenance.

**Non-Goals:**

- Deciding precedence (the library owns it).
- Making the publish gate share the render's reader (its reader also wants the published pin; left for a later cleanup).

## Research & Decisions

### Opt in unconditionally at the call site

**Context**: the kernel refuses an input carrying the file unless told to honour it.
**Explored**: a `--local-replacements` flag (off by default it would turn today's silent drop into a refusal for every developer who already has the file; on by default it is not a flag); an environment variable; unconditional opt-in.
**Options considered**:
1. New flag, default off - every existing local workflow starts refusing; the publish gates already guard the boundary that matters.
2. Unconditional opt-in - the file's presence is the developer's request; the render honours it and says so.
**Decision**: 2. `LocalReplacements: true` at the one call site.
**Rationale**: the file cannot exist in a fetched module and the operator never sets the flag, so the boundary the kernel's switch protects is "which frontend", not "which invocation".

### Warnings are built from rows, with the inert set computed locally

**Context**: the kernel reports what it honoured; it does not report what it ignored (an inert module-side replacement is not a kernel fact, it is a CLI one).
**Options considered**:
1. Keep the file-presence warning as is - stays wrong for inert entries and silent for platform replacements.
2. Rows only - a developer whose catalog redirect is inert gets no explanation.
3. Rows for honoured entries plus a local diff for inert ones.
**Decision**: 3.

```go
// internal/workflow/render/replacements.go
func replacementWarnings(rows []kernel.Replacement, moduleRoot string) []string
//   honoured: for each row ->
//     "local replacement in effect: <Path> served from <Target> (<By>); rendered bytes may not correspond to any published build"
//   inert: for each entry in loader.LocalReplacements(moduleRoot) whose Path no row carries ->
//     "local replacement of <Path> in <moduleRoot>/cue.mod/local-module.cue is ignored: the platform names that path; redirect it in the platform module's cue.mod/local-module.cue"
```

`moduleRoot` is the existing D19 module context (`loader.ModuleRootFrom` of the instance file's directory, or the module directory). Platform-side entries are only ever reported through rows: the platform is authoritative, nothing of its file can be inert.

### One reader in `pkg/loader`

**Decision**: `loader.LocalReplacements(moduleRoot string) ([]LocalReplacement, error)` returning `{Path, ReplaceWith}` per `replaceWith` entry, parsed with `cuelang.org/go/mod/modfile.ParseLocal` against the root's `module.cue` (the parser the kernel uses, so the CLI and the kernel agree on what counts as a replacement). `HasLocalModuleReplacement` becomes `len(entries) > 0 || parse error`, preserving its "malformed file counts as local" behaviour.
**Rationale**: `CompileBytes` on a modfile is an approximation; `ParseLocal` is the definition.

### Errors

- Kernel refusal naming `cue.mod/local-module.cue` (malformed file, target directory declaring another module path, version-less dependency with no replacement): already routed through `printValidationError`, exit `ExitValidationError`. No new handling.
- `loader.LocalReplacements` parse error while building warnings: the kernel has already refused by then, so the warning builder never sees one; it treats an error as "no entries" defensively.

Exit codes unchanged: 0 success, validation failures as today. Command syntax unchanged. No flags added.

Example, module render with a redirected shared module and an inert catalog redirect:

```
$ opm module build ./my-app
WARN local replacement in effect: example.com/my-lib@v0 served from /home/dev/my-lib (instance); rendered bytes may not correspond to any published build
WARN local replacement of opmodel.dev/catalogs/opm@v4 in /home/dev/my-app/cue.mod/local-module.cue is ignored: the platform names that path; redirect it in the platform module's cue.mod/local-module.cue
...rendered objects...
```

## Data flow

```
  module dir / instance file            platform dir
        |                                    |
   AcquireModuleFromDir + Synthesize    AcquirePlatformFromDir
   (developer dir is main module:       (ditto)
    local-module.cue applied)
        |                                    |
        +---------------- Kernel.Render(LocalReplacements: true)
                                |
                   Diagnostics.Replacements (rows)
                                |
                  replacementWarnings(rows, moduleRoot)
                                |
                   output.Warn x N   +   SourceLocal (unchanged)
```

## Risks / Trade-offs

- [A developer applies to a cluster with a replacement in effect] → already the D6 local-first story: the CR carries the declared reference, `source: local` is stamped, and operator handoff aborts on the digest (0006 D7.4). Unchanged.
- [The library bump lands before this change] → every render of a directory carrying the file refuses; the bump and the opt-in ship in one PR (Migration Plan).
- [Warning volume on a directory with many redirects] → one line per honoured or inert entry, sorted by path; acceptable for a developer-only condition.

## Migration Plan

One PR: `go.mod` bump to the library release carrying the opt-in, the call-site field, the warning builder, the reader, tests. `task check` green. Rollback is a revert of the whole PR (reverting only the bump would reintroduce the silent drop).

## Open Questions

None.
