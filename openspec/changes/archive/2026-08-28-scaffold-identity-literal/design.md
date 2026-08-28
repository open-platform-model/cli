## Context

See proposal.md § Why. Writers: `internal/scaffold/repair.go` `identityFileContent` (repair path), `internal/scaffold/scaffold.go` `Reidentify` → `cueedit.ResetIdentityVersion` (fetch path), and the committed `templates/*/identity/identity.cue`. `cueedit.SetIdentityVersion` already handles a plain literal as its first branch (`spliceVersion`, `versionLiteral` on `*ast.BasicLit`), idempotently. `ResetIdentityVersion`/`spliceVersionReset` exist only to force the disjunction form and have one caller. The `library` shape gate is the acceptance test: `opm module build <scaffold>` must pass it.

## Goals / Non-Goals

**Goals:**
- Every identity file the CLI emits is `Version: "x.y.z"`; every template loads through the kernel with `version: id.Version`.
- One fewer writer.

**Non-Goals:**
- Changing what `version set` tolerates in files it did not write.
- A publish-side loader gate (sibling change).
- Touching `tests/fixtures/modules/podinfo` (sibling change; it is a published fixture with its own bump flow).

## Decisions

**D1. Delete `ResetIdentityVersion`; `Reidentify` calls `SetIdentityVersion(dir, InitialVersion)`.**
Alternative: keep `Reset` and change its target form. Rejected: with the templates carrying a literal, `Set` on a literal is exactly the reset (`"1.0.0"` → `"0.1.0"`), and `Reset`'s remaining branches (`hasVersionType` fallback, open-field `| *`) would all be dead code. YAGNI (Principle VII).

**D2. Templates drop the `// x-release-please-version` marker with the disjunction.**
Measured 2026-08-28: `release-please-config.json` declares one Go package, no `extra-files`; `.github/scripts/publish-templates.sh` reads `cue eval ./identity --out text -e Version`. The marker is vestigial and the spec's own "marker intact" scenario is about *tolerance* in files the writer finds, which stands.

**D3. Template version bump: patch on all three.** A template is a published artifact and `publish-templates.sh` skips already-published tags, so the edit ships only with a new version. `1.0.0` → `1.0.1` via `opm module version set`, which also proves the writer on the literal form.

**D4. Repair path writes the same bytes the templates carry**, minus the module-specific values: `identityFileContent` keeps its provenance comment, drops the `#VersionType` block and its comment.

## Research & Decisions

### Does the plain-literal form survive every writer and gate?
**Context**: Constitution I (identity written only by `version set`) means the form must round-trip through the tool.
**Explored**: 2026-08-28 on a scratch copy of a fleet module with `Version: "1.0.1"` and no `#VersionType`: `opm module build` renders; `opm module vet` all green including `Identity conforms to #IdentityPackage`; `publish --dry-run` reports `Version concrete = 1.0.1`; `version set 1.0.1` is a no-op ("already 1.0.1; untouched"); `version set 1.0.2` rewrites the literal in place; a malformed literal is refused by core's `#VersionType` at load.
**Options considered**:
1. Plain literal (chosen): concrete under `IsConcrete()`, simplest writer branch, the identity is the value.
2. Keep disjunction, interpolate in `module.cue`: works around the gate one file away; 3-line comment in every module; leaves a form nothing wants.
3. `#VersionType & "x.y.z"`: concrete and self-checking, but the check duplicates core's and `#VersionType` must stay in the file.
**Decision**: option 1.
**Rationale**: it is what publish's tristate reports (`concrete`, not `concrete (default)`), what every consumer reads, and the least code.

## Risks / Trade-offs

- [A module scaffolded before this change carries the disjunction] → `version set` keeps tolerating it; `opm module build` refuses it with the kernel's (post-library-change) message naming the default. Repair does not rewrite an existing `Version` form; documenting the literal form in the init help text is enough.
- [e2e `mod_init_test.go:164` asserts `*"0.1.0"`] → flipped to assert `Version: "0.1.0"` and no `*`.

## Error handling

No new errors. `SetIdentityVersion`'s existing `ErrIdentityShape` covers a template without a `Version` field.
