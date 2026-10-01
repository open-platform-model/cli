## Context

`.github/scripts/publish-templates.sh` is the one gate list for the official templates, run by two jobs:

- PR CI, `pr.yml` `template-gates` (lines 98-114): builds `opm` from the PR, runs the script with `--dry-run`. A dry run whose only refusal is "already holds" passes (script lines 69-82).
- Release, `release.yml` `publish-templates` (lines 146-181): checks out the release tag, builds `opm` from it, runs the script without flags. Per template: skip if GHCR holds the tag (lines 61-65), else `tidy --check`, then `opm module publish` (lines 66-68).

The gates inside `opm module publish` cover identity, coordinates, namespace, package name, override and kernel loader shape (`internal/publish/gates.go`). None of them evaluates `debugValues` or renders. `opm module vet` does both: it checks values against `#config`, then renders the synthesized instance exactly as `opm module build` does against a platform generated from the module's own pins, without a cluster (`internal/cmd/module/vet.go:40-45`, `:84-87`, `:168-195`; spec `mod-vet`, requirement "mod vet command validates module without generating manifests").

`vet` has rendered only since commit 874bce8 (2026-09-29, first in v1.0.0-alpha.23). Before that it stopped after the `#config` check. This is why the gate could not have been "just vet" earlier, and why it is meaningful now: the release job always builds `opm` from the tag it publishes, so the vet it runs is the vet that ships.

No Go code, command, flag or exit code changes. The config rules about command syntax, flags and example output do not apply; the script's messages are specified instead.

## Goals / Non-Goals

**Goals:**

- A template whose `debugValues` does not render under the cli being built cannot publish, and fails the PR that introduces it.
- No template is pushed unless every template passed every gate in the same run.
- A PR that changes a template's files without bumping a version GHCR already holds fails.

**Non-Goals:**

- A render gate inside `opm module publish` for every module (see R2).
- Fixing the workspace `deps:update:templates` task to bump template versions (workspace repo; follow-up).
- Content comparison between the tree and the GHCR artifact in the release job (see R4).
- Republishing or retracting `advanced` v1.0.2. Published versions are immutable; `opm module init` resolves the newest stable version (v1.0.3), so v1.0.2 is only reached by an explicit `advanced@1.0.2` pin.

## Decisions

### D1. Vet every template, in both modes, through the shared script

The gate phase runs `"$opm" module vet "./$dir"` on every `templates/*/` tree, whether or not GHCR already holds its version, in both `--dry-run` and publish mode. A template that is already published and unchanged is still vetted, because the cli under test may no longer render it, and `opm module init` from that cli would scaffold a broken module.

The vet runs through the script, not as a separate workflow step, so the PR job and the release job keep one gate list (the `pr.yml:99-101` comment's promise).

### D2. Two phases: gate all, then publish

```
gate phase (both modes), for each templates/<t>/:
  version   := cue eval ./identity -e Version      # unchanged
  tidy      := opm module tidy --check ./<dir>      # existing gate, moved here
  vet       := opm module vet ./<dir>               # new
  dryrun    := opm module publish --dry-run ./<dir> # now also in publish mode
     GO                                  -> pass
     only refusal "already holds"        -> D3, else pass
     any other refusal                   -> fail
  record failure, continue to next template (report all, not first)
any failure -> print "gates failed for: <list>; nothing published", exit 1
--dry-run   -> exit 0
publish phase (release only), for each template:
  GHCR holds v<version> -> skip (unchanged filter, 0011:D15)
  else                  -> opm module publish ./<dir>
```

`opm module publish` re-runs its own gates during the push. That duplicate is intentional and cheap: the dry-run in the gate phase is what makes "every gate before any push" true across templates.

### D3. Changed implies bumped, PR only

When `BASE_REF` is set and a template's dry-run refuses only for "already holds", the script runs `git diff --quiet $(git merge-base "$BASE_REF" HEAD) -- <dir>`. A difference fails the template:

```
==> minimal: changed since origin/main but v1.0.3 is already published.
    Published versions are immutable: run 'opm module version set <semver> ./templates/minimal/'.
```

An unresolvable `BASE_REF` fails the run (it means a shallow checkout, a CI wiring bug). Without `BASE_REF` (release run, local use) the check is skipped. The merge-base, not the tip of `BASE_REF`, is the base, so a branch that only lags `main` (a release-please branch after a template bump landed) is not "changed". This is the shape of `hack/fixtures.sh` `changed_since` (lines 116-123) and `cmd_check` (lines 228-234).

`pr.yml` `template-gates` gains `fetch-depth: 0` on its checkout and `BASE_REF: origin/${{ github.base_ref || 'main' }}` on the script step, matching the `fixtures` job (lines 142-156).

### D4. Messages and exit codes

The script exits 0 when every gate passes (and, in publish mode, every needed push succeeds) and 1 otherwise. New lines, beside `opm module vet`'s own diagnostics:

- `==> <t>: 'opm module vet' failed; a template must render its own debugValues`
- the D3 message
- `==> gates failed for: <t> [<t> ...]; nothing published`

## Research & Decisions

### Reproduction

**Context**: the gate must fail on the content that shipped as advanced v1.0.2 and pass on `main`.

**Explored** (scratch copies only, `opm` built from `origin/main` 437bebd, canonical GHCR mapping, empty HOME):

| Tree | `opm module vet` | `publish --dry-run` |
| --- | --- | --- |
| `main` minimal, standard, advanced (v1.0.3) | exit 0 (1, 2, 8 resources) | already holds only |
| advanced at d5b5c01 (v1.0.2, last pre-fix) | exit 2, unresolved disjunction | not run |
| advanced at v1.0.0-alpha.22 (the release that first published v1.0.2) | exit 2, same | not run |
| GHCR `opmodel.dev/templates/advanced` v1.0.2, unpacked | exit 2, same | not run |
| `main` advanced with the two `updateStrategy: type: "RollingUpdate"` lines removed, `version set 1.99.0` | exit 2, same | `GO — pushing ... v1.99.0` |

The last row is the defect in one line: the current gates say GO for a template that does not render. The GHCR v1.0.2 `components.cue` is byte-identical to the alpha.22 tree.

With `opm` built from v1.0.0-alpha.22 itself, `vet` passes all three alpha.22 templates: that vet did not render yet (874bce8, above).

A prototype of the D2/D3 script, run in a scratch clone with `BASE_REF` at `main`:

| Case | Result |
| --- | --- |
| A: `main` unchanged | exit 0; all three "already published and unchanged" |
| B: advanced fix removed, bumped to 1.0.4 | exit 1, vet fails advanced; publish mode also exits 1 before any push |
| C: minimal pin changed (catalogs/opm v4.4.4 to v4.4.3), no bump | exit 1, changed-implies-bumped names minimal |
| D: C plus `version set 1.0.4` | exit 0, minimal `GO` at v1.0.4 |
| E: comment-only edit to standard, no bump | exit 1, changed-implies-bumped names standard |

### R1. vet versus build

**Options considered**:
1. `opm module vet`: renders as build does, prints one validation line per object, no manifests, exit 2 on failure. Its render is a spec requirement (`mod-vet`).
2. `opm module build > /dev/null`: same verdict (vet.go:40-41), but its diagnostics are less author-facing and it emits manifests to discard.

**Decision**: vet. **Rationale**: it is the command the spec defines as the author's pass/fail verdict, and it is the command `template-modules` already requires a scaffold to pass.

### R2. Where the render gate lives

**Options considered**:
1. In `publish-templates.sh` (chosen).
2. A new gate inside `opm module publish` (render `debugValues`, refuse on failure) for every module publisher.
3. A separate workflow step in `pr.yml` and `release.yml`.

**Decision**: option 1. **Rationale**: option 2 changes `opm module publish` for every publisher (modules fleets, fixtures that deliberately do not render), a product decision that belongs in an enhancement amending 0011, not in a CI fix. Option 3 splits one gate list across two YAML files, and the release path would drift from the PR path.

### R3. Where changed-implies-bumped lives

**Options considered**:
1. Inline in `publish-templates.sh`, copying `hack/fixtures.sh`'s `changed_since` (chosen).
2. Run `FIXTURES_DIR=templates hack/fixtures.sh check`.
3. Defer to a follow-up change.

**Decision**: option 1. **Rationale**: option 2 would make `hack/fixtures.sh`, which must stay byte-identical with opm-operator's copy (workspace `task fixtures:lint`), carry template-only gates (tidy, vet), and its policy text assumes the testing domain. Option 3 leaves the exact failure behind GHCR v1.0.2 (stale pins) open while the same file and job are already being edited; it adds about 10 lines.

### R4. Bump check in the release run

**Options considered**:
1. PR only, relying on the PR gate (chosen).
2. Compare the tree with the GHCR artifact's content in the release job.
3. Auto-bump unpublished changes at release time.

**Decision**: option 1. **Rationale**: every change reaches `main` through a PR, the release PR included, so the PR gate sees every tree. Option 2 must reproduce CUE's module-zip file selection to compare reliably, which is more code than the defect warrants. Option 3 publishes a version no commit declares, against the identity package as the single source of the version.

## Risks / Trade-offs

- [Release-time vet failure] The `publish-templates` job runs alongside `goreleaser`, not before it, so a vet failure at release leaves the cli binaries published and the templates unpublished. Mitigation: the release PR runs `template-gates` on the same tree, so this can only happen if GHCR dependencies changed between the PR run and the release, which immutable versions rule out. Fix-forward as for any failed job.
- [Vet stops rendering] The gate's strength rests on `opm module vet` rendering. The `mod-vet` spec requires it; a change that drops the render would have to change that spec.
- [Friction] Every template edit needs a bump, comment-only ones included, and `task deps:update` PRs now need a manual `version set` until the workspace task learns to do it. This is the cost of immutability, the same as fixtures.
- [Two PRs bump to the same version] Both pass on their own; if a release cuts between their merges and the second PR's CI is not re-run, the second lands changed content at a published version, and no later PR notices (its merge-base already contains the change). Same residual exposure as fixtures; R4 option 2 would close it and is the follow-up if it ever happens.
- [Network] vet resolves core and catalogs from GHCR, as the dry-run already does.

## Migration Plan

None. The current templates (all at v1.0.3) pass every new gate unchanged (case A).
