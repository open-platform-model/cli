# Design — cli-publish-pipeline

## Overview

One pipeline in `internal/publish`, two thin commands. The pipeline is a Go rendering of 0011's `#PublishPlan` (`schemas/target.cue` — the slice's specification, not a second enforcement point), with the gate semantics fixed by decisions D1–D21 and the message shapes fixed verbatim by `06-operational.md`. The reference implementation is experiment `02-publish-plan-gates` (14 measured verdicts), corrected for two known stalenesses: the flag is `--skip-override-check` (not the experiment's spelling), and modules now carry `identity/identity.cue` with `Version` (D12 reversed the experiment's D2-era module shape).

```
opm module publish [dir] [--version X] [--skip-override-check] [--dry-run]
opm catalog publish [dir] [--version X] [--dry-run]
        │
        ▼
 internal/publish
   1 load       root package + ./identity subpackage (load error surfaces as-is)
   2 identity   unify against core.#IdentityPackage (schema cache) — refusal 10
   3 derive     declaredPath = id.ModulePath (READ; split → repo, major)
                effective version = declared | --version-filled ; tag = "v"+effective
   4 gates      accumulate refusals (see table); print plan
   5 push       modzip.CreateFromDir → modregistry.PutModule   (skipped on --dry-run
                                                                or any refusal)
```

## Research & Decisions

### Push mechanism: public CUE SDK, in-process

**Context**: 0011 says only "CUE performs the push"; no API is named anywhere in the enhancement.
**Explored**: `cue mod publish`'s implementation vs the public surface at cuelang.org/go v0.17.1 (the CLI's exact pin).
**Options considered**:
1. Shell out to the `cue` binary — inherits everything, but resurrects the unimplemented `cue-binary-integration` change's discovery/version-skew problem and makes credentials/registry routing opaque to our error shaping.
2. In-process via public API — `modconfig.NewResolver(&modconfig.Config{CUERegistry: cfg.Registry})` → `modregistry.NewClientWithResolver` → `modzip.CreateFromDir` → `PutModule`. Everything is exported; credentials arrive free (the resolver's transport layers Docker/Podman config under CUE's `logins.json` — push and pull share one path, which is D11's whole argument); `CUERegistry` on the config struct means no `os.Setenv` (improving on the legacy env-mutation loader).
**Decision**: Option 2.
**Rationale**: pure Go, testable against the public in-process `modregistrytest` (immutable tags + auth), registry routing through `ResolveRegistry` precedence exactly like every other command, and our refusal messages stay ours. Two `cue mod publish` behaviors are handled explicitly rather than inherited:
- **`source:` preflight** — publish requires `cue.mod`'s `source: {kind: "self"}` (both shipped trees carry it; `self` selects the whole-directory zip with no VCS machinery). Absent/other → a refusal in the house shape pointing at `cue mod edit --source self`. (Experiment 02's one push-mechanism finding, now a gate instead of a late CUE error.)
- **Tidiness** — `modload.CheckTidy` is internal and is NOT reproduced. The pipeline's full decode (root + identity, resolved against the configured registry) already refuses the failure class that matters (unresolvable dependency tree); extraneous-dep hygiene remains `cue mod tidy`'s job in the authoring flow. Recorded as a deviation in 0011 on landing. `cue-binary-integration` stays unimplemented and un-absorbed — publish does not need the binary.

### The `--version` writer lands here as `internal/cueedit`

**Context**: refusal 1's action offers `opm catalog publish --version 1.3.0`; D12 says `--version` fills an *open* field by **writing the working tree** (D2: what is published is what is committed — an in-memory overlay would publish bytes the tree doesn't hold, since `modzip.CreateFromDir` ships what is on disk). But the surgical AST rewrite is `cli-authoring-commands`' mechanism (D8: locate `Version` by the schema-fixed path, preserve comments/alignment, rebuild the `&` chain).
**Decision**: a minimal `internal/cueedit` package implementing exactly D8's Version rewrite lands in this slice; `--version` fill uses it; `cli-authoring-commands` builds `version set` (and `mod init` repair) on it. Assert-vs-fill semantics per D3/D12: concrete field + agreeing flag → no-op; concrete + disagreeing → refusal 5; open + flag → write, then proceed with the written value; open + no flag → refusal 1.
**Rationale**: the alternative (assert-only until the sibling slice lands) ships refusal messages whose actions don't work — the messages are normative. Implementation-here/commands-there mirrors the accepted `StripProvenance` precedent. Recorded as a boundary adjustment in 0011 on landing.

### Identity states: `absent` / `open` / `concrete`, with defaults counting as concrete

**Context**: D4's tristate; but the shipped catalog identity declares `Version: #VersionType | *"2.0.0-alpha.3"` (release-please owns the default) — neither literal-concrete nor open.
**Decision**: the pipeline evaluates with CUE defaults; a defaulted field is **concrete** with the default as its value (that is the committed, release-automation-owned declaration — publishing it is the intended flow today). `open` = no value and no default (the D3 authoring posture); `absent` = the field or file missing at the schema-fixed path — never publishable, `--version` does not apply (a malformed artifact, not an unfinished one; refusal 10's unification error names it). One cause, one refusal: the absent path must not also report "no version available" (experiment 02's double-refusal fix).

### Gate inventory, evaluation order, and accumulation

Gates run in dependency order; everything evaluable is evaluated and refusals accumulate into a single list (experiment 02: "a refusal list beats a single error"). A tree that fails step 1 short-circuits — nothing downstream is evaluable, and the load error is surfaced verbatim ("blaming the wrong thing" fix).

| # | Gate | Decision | Refusal |
|---|---|---|---|
| 1 | tree + identity package load | — | load error verbatim |
| 2 | no `cue.mod` | D16 | msg 3 (points at `opm mod init`) |
| 3 | `source: {kind:"self"}` present | (mechanism) | new msg, house shape |
| 4 | identity conforms to `#IdentityPackage` | D21 | msg 10 = CUE's unification error through the grouped-error funnel |
| 5 | identity fields concrete (post `--version`) | D4/D3/D12 | msgs 1, 5 |
| 6 | `metadata.*` derives from `id.*` | D12 | msg 7 |
| 7 | `cue.mod` `module:` == declared path | D16 | msg 2 |
| 8 | module package name == `metadata.name` | D1 | new msg, house shape (drafted below) |
| 9 | tag == effective version; tag major == path major | D18 / `#TagRef` | msg 6 |
| 10 | namespace + kind segment (owned domains only) | D13/D14/D1 | `_kindAgrees` refusal, house shape |
| 11 | `local-module.cue` presence | D6 | msgs 4a/4b (`--skip-override-check`, module only) |
| 12 | tag not already published (registry lookup via `modregistry.Client.ModuleVersions`) | D15 | msg 8 |

Gate 12 is the only registry round-trip before the push; unreachable registry there is a connectivity error (exit 3), not a refusal.

**The drafted package-name refusal (gate 8 — no message exists in 06-operational; flag for a 0011 history note):**

```
error: modules/postgres's package does not bind to its name
  package    postgres_db    modules/postgres/module.cue:1
  name       postgres       modules/postgres/identity — via metadata.name

  A consumer's bare `import "opmodel.dev/modules/postgres@v2"` binds the package
  name; when they differ every consumer must alias the import.

  Rename the package:  package postgres
```

### Plan output and `--dry-run`

The plan prints on **every** invocation before any push (it is `#PublishPlan` rendered; experiment 02's format: kind, declared path, cue.mod path, repo, major, tag + tag source, one line per identity field with state/source/value, override gate, then `GO — pushing <repo>:<tag>` or `REFUSED` + the accumulated list). `--dry-run` stops after the plan with exit 0 on GO and exit 2 on REFUSED — the dry run runs *all* gates including the already-published lookup (the thin-editor preview precedent: a dry run surfaces rejections, never defers them). No `--check`, no JSON (future; `#PublishPlan` is the schema it would serialize).

### Exit codes and error display

`ExitSuccess 0` / any refusal `ExitValidationError 2` (status-exit-codes precedent: ran fine, result is bad) / registry unreachable `ExitConnectivityError 3` / unexpected `1`. Refusals print through the existing `PrintValidationError` funnel: messages 1–8 as `ValidationError` with details (aligned two-value columns via a new small `output` helper — `Table` is header-oriented and wrong for this); message 10 relies on the funnel's grouped-CUE-error path, which preserves `file:line`. Positions for messages 1/2/5/6/7 come from `cue.Value.Pos()` on `LookupPath` results — the identity loader keeps `cue.Value`s, never bare decoded structs.

### `opm module vet` checks and cfg threading

Vet currently drops `cfg` entirely, so its loads ignore the resolved registry — fixed here (constructor threads `cfg`; the load path gains the registry). D16 (coordinate agreement), D18 (tag/version agreement — against the declared version; vet has no tag argument, so the check is the derivation/major half), and D21 (identity conformance) run between module load and the values stanza, so a module with no `debugValues` still reports coordinate drift. Output via `FormatVetCheck` lines consistent with existing vet rendering. No `opm catalog vet` in this slice: catalog-side checks run inside `catalog publish` (and its `--dry-run`); a standalone catalog vet is noted for `cli-catalog-gates`.

### Test strategy

In-process registry on the **public** `modregistrytest` (`ocimem` with `ImmutableTags: true` — D15's already-published refusal and immutability are testable hermetically; `NewServer` with `AuthConfig` covers the authenticated-push path). The library's internal `registrytest` is not importable and is not copied — the CLI's harness builds the minimal module/catalog trees it needs as `txtar`-style fixtures. The acceptance matrix is experiment 02's 14-row verdict table, re-derived for D12's module shape (modules now have identity subpackages — the module rows become the same two-row table as catalogs) and D6's flag spelling, plus: defaulted-version concreteness, package-name rule, `source:` preflight, already-published (immutable registry), push-success round-trip (publish then resolve back).

## Open questions (carried, not blocking)

- Whether the plan gains `--json` when the cutover slices script against it (schema exists; deferred YAGNI).
- The git-sourced "you forgot to bump" *warning* (D15) remains unowned by any slice — not implemented here.
- Aligned-column rendering may want promotion into `output` as a general two-value-disagreement helper if `cli-catalog-gates` reuses it (likely).
