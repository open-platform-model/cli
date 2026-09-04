# Design: cli-render-switch

## Context

See `proposal.md` § Why. Current shape (library `alpha.23`): `internal/platform.Resolve` returns `synth.PlatformInput` from three sources through one wire decode; `platform.Materialize` chains `SynthesizePlatform` → `Materialize`; `internal/workflow/render.compileInstance` calls `Kernel.Compile` and adapts `CompileResult` (`Compiled`, `MatchPlan`, `Warnings`) into `Result`; `FromInstanceFile` layers `-f` values through `ProcessModuleInstance` (sourceless instance); apply's write-if-absent seed uses `Result.PlatformSpec`.

Upstream contract (library from `alpha.24`, plus `library-platform-module-generator`): `Kernel.Render(RenderInput{Instance, Platform, RuntimeName, Skew})` → `RenderResult{Compiled, Diagnostics, Warnings}`, `RenderError` unwrapping to `oerrors.{UnresolvedDemandsError, UnmatchedComponentsError, SkewError, TransformError, OverSubscribedContractError}`; `AcquirePlatformFromDir` (source-carrying, shape-gated); `AcquireInstanceFromDir(dir, opts, WithValues(sources...))` (overlay-mode source); `opm/helper/platformmodule.{Roots, Closure, Generate, Files.WriteTo, NewRegistry(RegistryConfig)}`; ADR-005 shares-nothing renders. The operator's switch (PR 119) is the reference implementation of the same contract.

Sibling: `cli-config-platform-module` supplies `config.PlatformDir` and the local module; it lands first in branch order.

## Goals / Non-Goals

**Goals**

- Every render-bearing command on `Kernel.Render`, with no CLI-side platform value, no Go-side values fill, and provenance reporting intact.
- CR ingestion structurally identical to the operator's (same helper, same acquire), so a CR that renders in-cluster renders identically from the CLI.
- `-f` keeps working on instance files.

**Non-Goals**

- A CLI command to edit or bump the cluster Platform CR.
- Per-command skew flags (config key + CR field only).
- Cache eviction policy beyond "safe to delete"; `~/.opm/cache/platforms/` is regenerable state.
- Changing `opm operator install`'s seed (it writes a CR spec from a registry-resolved coordinate; untouched).

## Decisions

### `Resolve` returns a directory and provenance; sources converge on `AcquirePlatformFromDir`

**Options**: (1) `Resolve` keeps returning a typed spec and the render layer generates a module for every source; (2) `Resolve` returns `(dir, Resolution)`, generating only for the CR source, and the render layer acquires once.
**Decision**: option 2. Flag and local sources already are directories; only the CR needs generation. One acquire call after resolution keeps the failure surface identical across sources (spec: "Same failure surface for every source"). `Resolution` gains `Dir` and, for the CR source, the CR's `skewPolicy`.

### CR generation: helper + content-hash cache under the OPM home

**Options**: (1) per-invocation temp dir; (2) `~/.opm/cache/platforms/<sha256 of Files>/`, idempotent write; (3) a fixed `~/.opm/cache/platforms/cluster/` overwritten per run.
**Decision**: option 2 (user decision). `Generate` is pure, so the hash is computed before any write; if the directory exists with matching content nothing is written, otherwise write to a staging sibling and rename (the same absent-or-complete property the operator's layout has, without generations). Closure derivation is the only registry I/O and goes through the CUE module cache (`NewRegistry(RegistryConfig{Registry: cfg.Registry, ClientType: "opm-cli", Env: os.Environ()})`); the cache dir is read from the OPM home resolved beside `config.cue`, so `--config` overrides move it too. Option 3 races between concurrent invocations; option 1 re-resolves every run.

### `-f` values go through the kernel's values option

**Decision**: `FromInstanceFile` builds `[]kernel.Source` from the values files with `LoadSourceFromFile` (filenames attributable) and calls `AcquireInstanceFromDir(instanceDir, opts, kernel.WithValues(sources...))`. The default `values.cue` beside the instance file stays part of the package when it declares the package; `-f` adds sources on top. `unifyValuesFiles` and the `ProcessModuleInstance` call are deleted from this path.
**Rationale**: the only alternative is a CLI-side copy of the overlay construction, which the library rule forbids.

### Seed decoded from the built platform

**Decision**: `platform.SpecFromPlatform(p *platform.Platform) (Spec, error)` reads `metadata.name`, `type` and, for each `#registry` entry, its key, `enable` and the derived `version` from `p.Package` through schema paths (`schema.Registry`). `Result.PlatformSpec` becomes that `Spec` (a CLI type: `Name`, `Type`, `Entries`), which `createClusterPlatform` marshals to the CR wire shape exactly as today. `DecodeCRSpec` returns the same `Spec` type so CR decode and seed share one shape.
**Rationale**: the module is the source of truth and the derived version is the only correct one; parsing `cue.mod` would be a second answer.

### Skew policy resolution

**Decision**: `kernel.SkewPolicy` is chosen in `resolvePlatformEnv`: CR source → CR's `spec.skewPolicy` (`Warn`/`Refuse`, absent = warn); otherwise `config.skewPolicy` (`warn`/`refuse`, absent = warn). Reported in the provenance line when it is `refuse`. `oerrors.SkewError` maps to the validation exit code with the kernel's message verbatim.

### Result surface

**Decision**: drop `MatchPlan` and `Components` (no reader; the verbose test constructs them). `Result.Warnings` = `RenderResult.Warnings` plus the D19 local-replacement warning as today. `Result.Pairs` (from `Diagnostics.Pairs`) is added only if the verbose output keeps printing pairs; otherwise nothing replaces `MatchPlan`. Decided at task 3.4 by reading the verbose printer.

### Error mapping

**Decision**: `printValidationError` learns `*kernel.RenderError`: print the message, then the diagnostics rows it carries (unresolved demands with alternatives, unmatched components, over-subscribed keys, failed pairs) in the existing validation-output style; exit code = validation for every render cause (the operator distinguishes reasons for status; the CLI has one failure channel). Pre-evaluation refusals that indicate a CLI defect (missing `Source`) surface as general errors with the kernel's message.

### Integration mains

**Decision**: `tests/integration/platform-materialize` becomes `platform-build`: resolve the seeded local module, `AcquirePlatformFromDir`, assert the registry entries' derived versions. `render-parity` runs Path A (local module dir → synth) and Path B (registry acquire → synth) both through `Render` against the same platform directory and compares digests.

## Risks / Trade-offs

- [Library release with the helper and values option not yet published] → gated; `go.mod` bump is task 1.1 and every task after 1 waits for it.
- [Content-hash cache grows] → each entry is two small files; documented as safe to delete; no eviction now (YAGNI).
- [Cold CUE module cache makes the first CR render slower (closure fetches module files)] → same artifacts the acquire fetches anyway; one-time per pin set.
- [A legacy CR without `version` used to fail at synthesis with a hint; now at generation] → same hint, earlier and before registry I/O (spec updated).
- [`-f` semantics shift from "fill then validate" to "unify in the package"] → same CUE result for consistent values; a conflict now names the file (better attribution).

## Migration Plan

Lands after `cli-config-platform-module` on one branch train; released together (one MAJOR-by-behaviour CLI release). User migration: `opm config init --force` (sibling change) and `--platform` pointing at a directory. Rollback is a revert of the train.

## Open Questions

None. Resolved at task 3.4: verbose output keeps the pair listing, so `Result.Pairs` (from `RenderDiagnostics.Pairs`) replaces `MatchPlan`; the component summaries (resource/trait FQNs, labels) and the skipped-pair debug lines are dropped, since the kernel's diagnostics carry neither and deriving them would mean navigating the instance value from Go.
