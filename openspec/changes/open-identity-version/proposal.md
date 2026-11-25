# Proposal

## Why

The publishing spec says that `--version` fills an open identity `Version`. Against core
v2.0.0-beta.4, publish refuses that open `Version` before it fills it, and vet refuses it too. The
identity check validates in concrete mode and drops errors whose text says `incomplete value`. Core's
`VersionMajor` derivation calls a builtin on the open value, and that error says `non-concrete
value`, so the filter keeps it. The publish tests pass only because their stub `#IdentityPackage`
leaves out that derivation. Separately, `TestModVet_ValidModule` always skips: the test never sets a
registry, so vet raises "no registry is configured" and the test reads that as an unreachable
registry (cli issue 326).

## What Changes

- The identity check separates "open" from "wrong" without reading error text. Plain CUE validation
  finds conflicts (a major skew, a malformed value). The field states that publish already computes
  find an absent field. The text match on `incomplete value` goes away.
- Publish fills the `--version` value into the identity before the check. The filled identity then
  gets the full concrete check, so a filled version whose major disagrees with the path is refused
  in the dry run.
- `opm module vet` on an open identity `Version` runs every check that does not need a concrete
  version, then skips the render, says why and how to fill the version, and exits 0. It does not
  print `Module valid (<n> resources)`.
- The publish test stub gains core's `VersionMajor` derivation, and one registry-backed test runs the
  open and fill cases against the real core schema.
- `TestModVet_ValidModule` sets a registry, and skips only on a connectivity failure.

Release class: PATCH (a bug fix that restores specified behaviour). Beta ships it as the next
beta.N.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mod-vet`: an open identity `Version` passes the identity check, and vet skips the render with a
  message instead of failing.
- `artifact-publishing`: the identity check treats an open `Version` as fillable, not as a
  conformance failure, and checks the identity as filled by `--version`.
- `errors-domain`: the list of allowed error-text reads loses the identity-conformance case, so three
  cases stay.

## Impact

- Commands: `opm module vet`, `opm module publish`, `opm catalog publish`.
- Packages: `internal/publish` (identity check, gate order, test stub), `internal/cmd/module` (vet
  render skip, vet tests), `pkg/errors` (the allowed text-match list), `tests/fixtures/` (one new
  vet fixture with an open `Version`).
- Docs: the `VetChecks` godoc and the stub comment; the count of allowed text matches in `AGENTS.md`.
- No new dependency, flag or exit code.

## Enhancement

None.
