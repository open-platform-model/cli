# Design: cli-config-platform-module

## Context

See `proposal.md` § Why. Current shape: `internal/config/templates.go` embeds a data-only `DefaultPlatformTemplate` (scalar `version` subscriptions), `internal/config/platform.go` validates it against the embedded `#PlatformFile` projection schema with an explicit import ban, and `internal/platform/resolve.go` decodes it into `synth.PlatformInput`. Under 0019 D5 the subscription shape no longer exists in core and `synth.Platform` is deleted by the library wave; the local default platform becomes a CUE module.

Command syntax is unchanged: `opm config init [--force]`, `opm config vet`. No new flags.

## Goals / Non-Goals

**Goals**

- `opm config init` scaffolds `~/.opm/platform/` (module form) offline; `opm config vet` proves it builds.
- A single documented maintenance loop: edit `cue.mod` pin, run vet.

**Non-Goals**

- No resolution or render-path changes (`internal/platform`, `internal/workflow`): `cli-render-switch`.
- No cluster-CR-to-module generation: `cli-platform-cr-generation`. The `#PlatformFile` projection schema is retired only for the local file; the CR-spec decode keeps whatever it uses until that change.
- No `opm config platform bump` command (Principle VII; the loop is two steps and `cue mod get` exists).

## Research & Decisions

### The local default becomes a directory, not a smarter file

**Context**: a platform with imports needs a `cue.mod`; a single file cannot carry one.
**Options considered**:
1. `~/.opm/platform/` module directory, sibling of `config.cue` — real module, loader-compatible, `--config` override moves it too (existing `PlatformFilePath` sibling rule generalizes).
2. Keep `~/.opm/platform.cue` as data and generate a module in a cache dir at render time — preserves the old file format but re-introduces generation machinery this change exists to avoid, and vet could not validate what the user actually edits.
**Decision**: option 1. `internal/config/paths.go` replaces `PlatformFile` with the platform module directory (`PlatformDir`), same sibling rule.
**Rationale**: the module IS the artifact (0019: "the platform module is the resolution"); the user edits the thing the render consumes.

### Module path: `opmodel.dev/platforms/local@v0`

**Context**: every CUE module declares a path; this one is never published.
**Options considered**:
1. `opmodel.dev/platforms/local@v0` — the namespace 0019 D6 explicitly reserves as reserved-unpublished for platforms.
2. An example.com-style placeholder — avoids the production namespace but invents a second convention beside the reserved one.
**Decision**: option 1.
**Rationale**: a main module's own path is never fetched, publishing platforms is disallowed outright (D6), and the reserved namespace exists precisely so generated/local platforms have a home that can never collide with a published artifact.

### Pins live in `cue.mod`; `platform.cue` carries no versions

**Context**: D5 derives an entry's `version` from the imported catalog; two answers to one question is the shape D5 removed.
**Decision**: the template's `cue.mod/module.cue` pins core + both catalogs; `platform.cue` entries are `{#catalog: <import>}` only. Template comments document the bump loop (edit pin, `opm config vet`).
**Rationale**: byte-for-byte the D5 contract; the derived readout plus the key binding turn a wrong pin or wrong import into a build conflict vet surfaces.
**As implemented**: the pins are Go literals (`DefaultCorePin`, `DefaultCatalogPins`, index-aligned with `DefaultCatalogPaths`) rendered into the embedded `cue.mod`, not a CUE string block; the root `platform-pins.sh` bumps each by its key comment, and the legacy data literal plus `spec_test.go` derive from the same values, so no second literal tracks them. Core is a hand-pinned literal because the library at this pin cannot supply its verified release; `cli-render-switch` may switch it to `schema.DefaultSchemaVersion()`.

Template sketch (final import spellings verified against the published catalogs at implementation time — the catalog root package name is confirmed from the artifact, not assumed):

```cue
// ~/.opm/platform/cue.mod/module.cue
module: "opmodel.dev/platforms/local@v0"
language: version: "v0.17.0"
deps: {
    "opmodel.dev/core@v2":         v: "v2.0.0-alpha.<D5>"
    "opmodel.dev/catalogs/opm@v4": v: "v4.0.1"
    "opmodel.dev/catalogs/k8s@v1": v: "v1.0.0-alpha.2"
}

// ~/.opm/platform/platform.cue
package platform

import (
    core   "opmodel.dev/core@v2"
    opmcat "opmodel.dev/catalogs/opm@v4:<pkg>"
    k8scat "opmodel.dev/catalogs/k8s@v1:<pkg>"
)

core.#Platform

metadata: name: "cluster"
type: "kubernetes"

#registry: {
    "opmodel.dev/catalogs/opm@v4": #catalog: opmcat
    "opmodel.dev/catalogs/k8s@v1": #catalog: k8scat
}
```

### Vet builds the module; init stays offline

**Context**: validation of a module with imports requires resolving them.
**Options considered**:
1. Vet builds through the kernel's shape-gated platform loader (`LoadPlatformPackage` today; the source-carrying acquire once the library wave lands) — real proof, registry I/O on a cold module cache only.
2. Shallow vet (parse `platform.cue`, check `cue.mod` syntax, cross-check entry keys against import paths) — offline always, but proves nothing about pins and duplicates checks the schema already makes structural.
**Decision**: option 1, with the division stated in help text: init writes pins offline; vet verifies resolvability.
**Rationale**: the maintenance loop's whole point is "did my pin bump work"; only a build answers that. Unit tests keep vet hermetic by pointing `--config` at a fixture platform module whose deps resolve from the test registry env (the repo's existing GHCR-resolving test posture).

### Legacy file migration is loud, and init cleans it up

**Context**: existing users have `~/.opm/platform.cue`; leaving it beside the new directory would be a silent second answer.
**Decision**: vet fails on the legacy file naming it (hint: `opm config init --force`); init removes it after writing the module and prints the removal. Mirrors the existing stale-`providers:` pattern.
**Rationale**: fail-loud-and-name-the-fix is the repo's error style; auto-migrating content (translating scalar pins into `cue.mod`) is more machinery than the two-catalog default warrants, and `--force` re-seeds the same catalogs anyway.

## Error handling

- Init: unchanged errors (exists-without-force validation error; permission errors), plus the removal note for the legacy file. Exit codes unchanged.
- Vet: platform-module build failures surface as `DetailError{Type: "platform module error", Location: <dir>, Hint: ...}` wrapping the loader/CUE cause; a nonexistent pinned build names the dependency (the CUE resolver error carries it); key/import drift names the `#registry` entry path. Legacy-file failure: `Type: "validation failed"`, hint `Run 'opm config init --force' to migrate to the platform module`. Exit codes follow the existing vet mapping (validation error).

Example vet output (failure case, config checks passing first):

```
[x] config.cue valid
[ ] platform module: building ~/.opm/platform: opmodel.dev/catalogs/opm@v4: version v4.9.9 does not exist
    Hint: pin a published build in ~/.opm/platform/cue.mod/module.cue, then re-run 'opm config vet'
```

## Files

`internal/config/templates.go` (two new template consts, old `DefaultPlatformTemplate` removed; `DefaultCatalogPaths` stays the single source of the paths), `internal/config/paths.go` (+`PlatformDir` sibling rule), `internal/config/platform.go` (module build validation; projection-schema use removed from the file path), `internal/config/schema/platform.cue` (trimmed to whatever the CR decode still needs), `internal/cmd/config/init.go`, `internal/cmd/config/vet.go`, tests beside each, `hack/platform.cue` → `hack/platform/` (module mirror), `CLAUDE.md` pin-mirror note.

## Migration Plan

Lands on the 0019 release train (see proposal § Sequencing): branch-ordered after the library wave's core prerelease exists, released together with `cli-render-switch`. Rollback pre-release is a revert. User migration is one command (`opm config init --force`), enforced by vet's legacy-file failure.

## Open Questions

None blocking. The exact D5 core prerelease and catalog builds the template pins are filled at implementation time from what is published (the same hand-pin rule as today).
