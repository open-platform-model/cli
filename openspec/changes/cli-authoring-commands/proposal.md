# Proposal — cli-authoring-commands

> Slice `cli-authoring-commands` of enhancement `0011`, first of two changes (this one: the writers and `version set`; the `mod init` rewrite — scaffold-from-template and repair — is `cli-template-modules`, and the slice's `openspec_ref` will cite both). Decisions D3, D8, plus the authoring halves of D12 and D16.

## Why

Publishing reads identity and refuses defects, but nothing *writes* identity deliberately. The intended flow — `version set` → review the diff → commit → `publish`, putting a commit between deciding a version and pushing an artifact (D3) — has no command, and the refusal actions the pipeline already prints (`opm catalog version set 1.3.0` in refusals 1, 5, and 6) name a command that does not exist. The writer mechanism shipped with the pipeline (`internal/cueedit`, byte-splicing, golden-tested) but has two gaps against D3's contract: it always writes even when the value is unchanged, and it exposes no read, so "already set to X" cannot be reported.

This change also lands the remaining writers the init rewrite will need (`cue.mod` `module:` line), so `cli-template-modules` composes writers rather than inventing them.

## What Changes

- **`opm module version set <semver> [path]` and `opm catalog version set <semver> [path]`** — write `identity/identity.cue`'s `Version` in place via `cueedit`, **idempotently**: setting the already-declared version writes nothing (no mtime change, no diff, nothing for a pre-commit hook to react to) and reports "already X". Offline by design — no registry, no schema fetch; a structurally non-conformant identity file (missing schema-fixed `Version`, unparseable) is refused per D8, pointing at `opm module vet` for full conformance.
- **`internal/cueedit` closes its gaps**: no-op detection inside `SetIdentityVersion` (spliced bytes == existing → no write, `Changed` reported), `ReadIdentityVersion` (value + defaulted-ness), and a new `SetCueModModule` splice-writer for the `module:` line (consumed by `cli-template-modules`' repair and re-identification). `publish --version` is unaffected — it only fills open fields, where a no-op is impossible.
- **Not here**: everything `mod init` — argument grammar, templates, scaffold, repair, second confirmation — moved wholesale to `cli-template-modules` so the command is rewritten once, coherently.

## Capabilities

### New Capabilities

- `authoring-commands`: `version set` for both kinds — idempotent, offline, assertion-preserving.

### Modified Capabilities

<!-- none -->

## Impact

- **SemVer: MINOR** — new commands; `cueedit.SetIdentityVersion`'s return gains `Changed` (internal package, one caller in `internal/publish`, updated here).
- **Commands**: `version` subgroups under both `module` and `catalog` groups, one shared runner in `cmdutil`.
- **Packages**: `internal/cueedit` (three additions), `internal/cmdutil`.
- **Consumers**: the pipeline's refusal actions become runnable; `cli-template-modules` builds on the writers; `catalogs-publish-cutover` depends on this slice.
- **Fixtures**: per 0011's graduation gate, idempotency is asserted against **real repos as fixtures** — copies of the shipped catalog and module identity files.
