## Context

See proposal.md for motivation and specs/ for the behavior contract.

Current state that shapes the approach:

- `render.FromModule` (`internal/workflow/render/module.go`) acquires with `Kernel.AcquireModuleFromDir`, resolves values (`ResolveModuleValues`: `-f` files, else `debugValues`) and calls `Kernel.SynthesizeInstance`. The operator does the same with `AcquireModuleFromRegistry` (`opm-operator/internal/moduleacquire`), so the kernel already supports a registry-acquired module end to end.
- `opm instance build` dispatches on `os.Stat` (`internal/cmd/instance/build.go`): a directory goes to `FromModule`, a file to `FromInstanceFile`. `FromInstanceFile` already reduces a directory argument to itself (`cmdutil.InstanceDir`), but computes the module context from `filepath.Dir(arg)`, which is the parent when the argument is a directory.
- Input guards judge by file name: `ValidateModuleInputPath` refuses a directory holding `instance.cue`; `ValidateInstanceInputPath` refuses one holding `module.cue`. Enhancement 0016's experiment 03 hit the first one: `opm instance build .` in an instance package fails with "is an instance package, not a module".
- The kernel's shape gate classifies packages by `kind` and wraps `oerrors.ErrWrongKind` when the kind does not match the acquire verb (`library/opm/errors`).
- Version listing: `publish.NewRegistryClient(...).ModuleVersions` on a major-free path returns every major's tags (0016 experiment 01). A candidate's dependencies are readable from its module file alone (`modconfig.Registry.ModFile`), about 0.7 s cold per major against GHCR (experiment 02).
- Selection predicates exist in two forms: `scaffold`'s `highestStable` (prerelease fallback that can return a dev build, experiment 07) and `publish`'s `isDevTag`/`isReleasePrerelease`. The rule 0016 D5 fixes for init is the publish one.
- The CLI's core major is the major of the library's `schema.DefaultSchemaVersion()` (`v2` today).

## Goals / Non-Goals

**Goals:**

- One resolver for "published module path plus optional version", shared by `module build`, `module apply` and the planned `instance init`.
- `FromModule` gains a source, not a second pipeline: after acquisition, a registry module and a directory module take the identical synthesis and render path.
- `opm instance build` decides by package kind through the kernel, so file-name heuristics stop deciding what a directory is.

**Non-Goals:**

- Published-module input for `opm module vet` or any `instance` command. Nothing has asked for it; the resolver makes it cheap later.
- Changing synthetic-instance defaults (`<module>-debug`, namespace `default`) for published modules. They stay identical to the directory form.
- Writing or tidying any `cue.mod`. This change has no dependency on `add-dependency-tidy`.

## Decisions

### Command syntax and flags

```text
opm module build [path | module-path] [--version <vN | X.Y.Z>] [flags]
opm module apply [path | module-path] [--version <vN | X.Y.Z>] [flags]
opm instance build <instance.cue | instance-dir> [flags]
```

| Command | Flag | Type | Default | Description |
| --- | --- | --- | --- | --- |
| `module build`, `module apply` | `--version` | string | `""` | Version of a published module: `vN` floats within major N, `X.Y.Z` pins. Empty takes the highest core-compatible major. Refused with a local directory. |
| `instance build` | `--name` | removed | | Unknown flag from now on (cobra usage error, exit 1). |

All other flags are unchanged.

### Argument classification

Rule, applied before any registry access (spec: "`opm module build` ... accepts a module directory or a published module"):

1. `.`, a `./` or `../` prefix, or `filepath.IsAbs`: local directory.
2. First element contains a dot: published module path. If it also exists on disk as a directory: refuse as ambiguous.
3. Otherwise: local directory.

**Options considered**:
1. Existence first (on disk means local, else registry): silent, and a typo in a directory name turns into a registry lookup with a confusing "no published versions" error.
2. A separate `--from <module-path>` flag for the registry form: explicit, but two ways to name the thing `module build` builds, and 0016 D5 uses the positional for the module path.
3. Shape first with an ambiguity refusal (chosen): the Go-toolchain convention (paths with a `./` prefix are local), no behavior change for `.`, `./x` or plain `my-module`, and a loud refusal for the one ambiguous case.

**Decision**: Option 3. The only invocation that changes meaning is a bare dotted relative directory (`opm module build web.app`), which is now refused with the `./web.app` hint instead of silently guessed.

### Resolver package and signatures

```go
// internal/modref
package modref

// ParsePath validates a major-free published module path. It refuses an
// @-suffix (hint: --version) and a URL scheme. Pure, no I/O.
func ParsePath(arg string) (string, error)

// Selector is a parsed --version value: Major ("v1") floats, Exact
// ("1.0.4") pins, both empty means highest core-compatible major.
type Selector struct{ Major, Exact string }

func ParseSelector(s string) (Selector, error)

// Source is the registry surface resolution needs; modconfig.Registry
// satisfies ModFile, the modregistry client ModuleVersions.
type Source interface {
	ModuleVersions(ctx context.Context, path string) ([]string, error)
	ModFile(ctx context.Context, mv module.Version) (*modfile.File, error)
}

type Strategy int // Exact, FloatMajor, HighestCompatibleMajor

type Skipped struct{ Major, Reason string }

type Resolution struct {
	Path, Major, Version string // "opmodel.dev/modules/web_app", "v1", "v1.0.4"
	Strategy             Strategy
	CoreMajor            string // the CLI's core major, "v2"
	Skipped              []Skipped
}

// Import returns the major-qualified module path ("…/web_app@v1").
func (r *Resolution) Import() string

func Resolve(ctx context.Context, src Source, path string, sel Selector, coreMajor string) (*Resolution, error)

// NewSource builds a Source over the CLI's registry mapping.
func NewSource(registry string) (Source, error)
```

`Resolve` returns a `*publish.ConnectivityError` for transport failures (exit 3) and a refusal carried as `publish.Refusal` for everything the user must fix (exit 2), the funnels `opm module init` already uses. The selection predicate is `publish`'s dev-tag rule, moved into `modref` and called back from `publish` so there is one definition.

**Why a new package and not `internal/scaffold`**: `scaffold` owns template references, whose grammar expands bare words into `opmodel.dev/templates/<name>` and whose selector deliberately differs (experiment 07). Sharing it would couple deployment to template policy.

### Data flow

```text
opm module build <arg> [--version v]
  -> classify(arg)                                   local | published | refuse(2)
  published:
  -> modref.ParsePath, ParseSelector                 refuse(2), no I/O
  -> modref.Resolve(NewSource(cfg.Registry), ...)    ConnectivityError(3) | Refusal(2)
       ModuleVersions(path) -> group by major
       exact: tag must exist
       vN:    newest selectable in N
       none:  majors high->low, newest selectable, ModFile -> core dep major == CoreMajor?
  -> report on stderr (output.Info): "Resolved <path> -> <major> <version> (<strategy>)"
                                     "  skipped <major>: <reason>" per skipped major
  -> render.FromModule(ModuleOpts{Published: res, ...})
       k.AcquireModuleFromRegistry(ctx, res.Import(), res.Version)
       ResolveModuleValues(k, mod.Package, "<path>@<version>", -f)   origin label, not a dir
       k.SynthesizeInstance(...)                     unchanged
       resolvePlatformEnv(...)                       unchanged
       renderInstance(..., moduleRoot="", sourceLocal=false)
```

A published module has no local module context, so `moduleRoot` is empty: replacement warnings then come only from the platform's rows, and the render carries no local provenance.

### Exit codes and messages

| Condition | Exit | Message shape |
| --- | --- | --- |
| Rendered or applied | 0 | resolution report on stderr, manifests or apply summary as today |
| Path with `@` suffix | 2 | `module path must not carry a major; use: <path> --version <vN>` |
| Malformed `--version` | 2 | `--version "<v>" must be vN (float) or X.Y.Z (pin), e.g. v1 or 1.0.4` |
| `--version` with a local directory | 2 | `--version applies only to a published module; <dir> is a local directory` |
| Ambiguous dotted argument | 2 | `"<arg>" is both a directory and a module path; use ./<arg> for the directory` |
| Pin not published, no versions, nothing compatible | 2 | refusal naming path, registry and (for the walk) every skipped major with its reason |
| Registry unreachable | 3 | `listing published versions of <path> (registry <registry>): <cause>` |
| `instance build` on a module package | 2 | `<path> is a module, not an instance; build it with: opm module build <path>` |

### Example output

```text
$ opm module build opmodel.dev/modules/web_app > manifests.yaml
INFO Resolved opmodel.dev/modules/web_app -> v1 1.0.4 (highest major on core v2)
INFO   skipped v2: requires core v3
INFO Building synthetic instance "web-app-debug" for module "web-app"
...

$ opm module build opmodel.dev/modules/web_app@v1
Error: module path must not carry a major; use: opmodel.dev/modules/web_app --version v1
(exit 2)

$ opm instance build ./my_app
Error: /home/me/src/my_app is a module, not an instance; build it with: opm module build /home/me/src/my_app
(exit 2)
```

### `opm instance build` decides by kind

`runInstanceBuild` always calls `render.FromInstanceFile`. `FromInstanceFile` computes the module context from `instanceDir` (not `filepath.Dir(arg)`), so a directory argument resolves its own `cue.mod`. `ValidateInstanceInputPath` stops judging by file names; an `oerrors.ErrWrongKind` from `AcquireInstanceFromDir` whose package is a module becomes the exit-2 refusal naming `opm module build <path>`. Because every `instance` render command shares `FromInstanceFile`, `instance apply`, `vet` and `diff` gain the same refusal; their exit code for a module directory moves from 1 to 2.

## Research & Decisions

### Grammar: major-free path plus `--version`, or `path@version`

**Context**: Enhancement 0016 D5 fixes the published-module grammar for `opm instance init`. `module build` and `apply` could have used the `@vN` suffix every other CUE tool uses.
**Explored**: 0016 `03-decisions.md` D5 and experiments 01, 02 and 07; the conversation that produced this change.
**Options considered**:
1. `path@vN` / `path@vX.Y.Z` positional: familiar from CUE, but a second spelling beside init's.
2. Major-free path plus `--version` (0016 D5): one spelling across the CLI; no version picks the newest line this CLI can build.
**Decision**: Option 2, user decision 2026-09-28.
**Rationale**: One resolver and one spelling. The core-compatible default is the answer a deployer wants and is cheap to compute (one module-file read per candidate major).

### Floating selection rule

**Context**: "Newest" was agreed for floats; `scaffold`'s rule can return a dev build on a dev-only major (0016 experiment 07).
**Decision**: Newest stable, else newest release prerelease, never a dev build; an exact pin may name any published tag.
**Rationale**: Same rule as 0016 D5:R2, so `module apply` and `instance init` never disagree about "newest".

### Removing the module-directory branch from `opm instance build`

**Context**: The branch duplicates `opm module build <dir>` (same `FromModule` call, same flags) and blocks instance package directories.
**Options considered**:
1. Keep the branch, try instance first and fall back to module: keeps compatibility, but the command's meaning then depends on what happens to be in a directory.
2. Remove it (chosen): one meaning per command group, at the cost of a breaking change.
**Decision**: Option 2, user decision 2026-09-28. It lands as its own section and commit, marked breaking.

## Risks / Trade-offs

- [No-version resolution costs one module-file read per major walked] -> Only majors above the selected one are read; the CUE cache serves repeats. `--version` skips the walk.
- [Assumption: `AcquireInstanceFromDir` on a module package wraps `ErrWrongKind`] -> Section 1 proves it with a test before section 3 relies on it; if it wraps something else, the refusal keys on that sentinel instead and the finding goes here.
- [Assumption: `ModuleVersions` on a major-free path lists every major against an in-memory `modregistrytest` registry as it does against GHCR] -> Proven in section 1; if not, the resolver lists per major by probing `@v0`..`@vN`, recorded here.
- [A bare dotted relative directory changes from "built" to "refused as ambiguous"] -> The refusal names the `./` spelling. The directory forms `opm mod init` scaffolds (`./my_app`, `.`) are unaffected.
- [`opm instance build <module-dir>` breaks existing scripts] -> The refusal message names the exact replacement command.

## Migration Plan

Sections 1 and 2 are additive. Section 3 is the breaking step: scripts using `opm instance build <module-dir>` move to `opm module build <module-dir>`; scripts passing `--name` to `opm instance build` drop it. Rollback is reverting the section's commit.
