# Design — cli-authoring-commands

## Overview

A thin change by construction: the writer mechanism already shipped with the pipeline; this closes its contract gaps and puts the two commands over it. The init rewrite that an earlier draft carried here moved to `cli-template-modules` (fetch-based scaffold made repair and scaffold one coherent rewrite); this change's job is to leave behind a complete writer toolkit.

## Research & Decisions

### Idempotency lives in `cueedit`, reported to the caller

**Context**: D3 requires "setting the version it already has is a no-op"; experiment 01 measured it as "file not written, no mtime change". The landed `SetIdentityVersion` always writes.
**Decision**: `SetIdentityVersion` compares spliced bytes against existing bytes and skips the write when equal, returning `Changed`. A new `ReadIdentityVersion(dir) (value string, defaulted bool, err error)` reuses the same parse so the command can report "already 1.3.0". `publish --version` is unaffected (fills open fields only).
**Rationale**: leaving the check in the command would leave the library primitive writing gratuitously for every future caller; the byte comparison is exact and free.

### `version set` is offline; conformance stays with vet/publish

**Context**: D8 refuses a non-conformant identity file "as a schema failure"; full `#IdentityPackage` unification needs core's schema from the registry; the command's habitat is a laptop pre-commit.
**Decision**: refuse on `cueedit`'s structural contract only (present, parseable, `Version` at the schema-fixed path — the same `ErrIdentityShape` the pipeline uses), no registry I/O, refusal points at `opm module vet`.
**Rationale**: D8's measured refusal cases are structural — the schema-fixed *path* is the locator, and the locator failing is the refusal. Type-level defects are caught one command later by vet/publish, which run the real unification. An offline writer that sometimes dials the registry is the worse surprise.

### `SetCueModModule` lands here, used later

**Context**: `cli-template-modules`' repair and re-identification both write the `cue.mod` `module:` line; nothing writes it today (the pipeline's reader is read-only by design — D16 forbids publish editing it).
**Decision**: the splice-writer ships here with the rest of the toolkit, exercised by unit tests only until its consumers land.
**Rationale**: same pattern as `StripProvenance` and the original `cueedit` — the mechanism lands in the first slice that owns the file, the commands compose it later.

## Technical Notes

- Command wiring: `version` subgroups under both groups; one shared `runVersionSet(kind, args)` in `cmdutil`; exit 0 on set *and* no-op (idempotent success), 2 on refusal.
- Graduation fixtures: copies of `catalog_opm/src/identity/identity.cue` (defaulted value + release-please marker — the write must preserve the marker line shape) and `modules/web_app/identity/identity.cue`, per the gate's "real repos as fixtures" wording.
- The `Changed` return ripples to exactly one existing caller (`internal/publish`'s fill path), updated in the same commit.
