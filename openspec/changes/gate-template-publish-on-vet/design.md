## Revisions

- **Revision 2** (2026-10-01, superseded): changed-implies-bumped compared each template with the previous release tag. Review 2 showed that this base moves past an unbumped change once a release lands on top of it. At v1.0.0-alpha.26, minimal and standard passed as "unchanged since alpha.25" although they differ from the v1.0.2 that alpha.22 published, and a synthetic release on top of an unbumped change passed the next PR and the next release.
- **Revision 3** (2026-10-01, this text):
  - Changed-implies-bumped compares **content against GHCR**, not git. For an already-published version, the script fetches the published module zip anonymously, builds the tree's zip with the publisher's own file selection, and compares the two file by file (D3). It needs no history, no tags, no `fetch-depth: 0` and no re-run exception, and every fetch or comparison error fails.
  - The release workflow now runs `publish-templates` before `goreleaser` builds, so a template failure leaves the release a draft instead of shipping binaries without templates (D5).
  - The verification harness strips `<dir>` before forwarding `"$@"`, and puts `cue` behind a shim too.
  - D3, D4, R3, R4 and Risks are rewritten, and D5 is new. D1, D2, R1, R2 and R5 to R7 stand.

## Context

`.github/scripts/publish-templates.sh` is the one gate list for the official templates, run by two jobs:

- PR CI, `pr.yml` `template-gates` (lines 98-114): builds `opm` from the PR, runs the script with `--dry-run`. A dry run whose only refusal is "already holds" passes (script lines 69-82).
- Release, `release.yml` `publish-templates` (lines 146-181): checks out the release tag, builds `opm` from it, runs the script without flags. Per template: skip if GHCR holds the tag (lines 61-65), else `tidy --check`, then `opm module publish` (lines 66-68).

The gates inside `opm module publish` cover identity, coordinates, namespace, package name, `source: kind: "self"`, override and kernel loader shape (`internal/publish/gates.go`). None of them evaluates `debugValues` or renders. `opm module vet` does both: it checks values against `#config`, then renders the synthesized instance exactly as `opm module build` does against a platform generated from the module's own pins, without a cluster (`internal/cmd/module/vet.go:40-45`, `:84-87`, `:168-195`; spec `mod-vet`, requirement "mod vet command validates module without generating manifests").

`vet` has rendered only since commit 874bce8 (2026-09-29, first in v1.0.0-alpha.23). Before that it stopped after the `#config` check. This is why the gate could not have been "just vet" earlier, and why it is meaningful now: the release job always builds `opm` from the tag it publishes, so the vet it runs is the vet that ships.

One test already renders a template: `tests/e2e/mod_init_test.go:127`, `TestE2E_ModInit_ThenVet`, publishes `standard` to an in-process registry, scaffolds from it with `opm mod init` and requires the scaffold to pass `opm module vet` and a publish dry-run. It covers only `standard`, and it vets the re-identified scaffold rather than the template tree. The two checks complement each other: the script vets what is published, and the test vets what a user gets (the `template-modules` requirement that a fresh scaffold passes vet). Section 4 extends the test to all three templates.

**What a publish pushes.** `opm module publish` zips the module directory with `modzip.CreateFromDir` (`internal/publish/registry.go:93`) and pushes it with `modregistry.PutModule`. That function reads `cue.mod/module.cue` out of the zip to form the second layer (`checkModule`, `cuelang.org/go@v0.17.1/mod/modregistry/client.go:441`). The GHCR manifest therefore carries one `application/zip` layer that holds the artifact's whole content (measured on `opmodel.dev/templates/minimal` v1.0.3: config, zip layer, modulefile layer).

**How changes reach cli `main`** (measured 2026-10-01): `gh api repos/open-platform-model/cli/rules/branches/main` returns only the mention-guard workflow rule. The `Protected` ruleset (PR requirement) is disabled, branch protection is absent, and no status check is required. Direct pushes land, and a red PR check does not block a merge.

**Release job graph on cli `main`** (437bebd, after draft-first release, PR 260):

- release-please, acting as the release App, creates the tag (`force-tag-creation`) and a **draft** GitHub Release.
- Then `goreleaser` and `publish-templates` run **in parallel**.
- `goreleaser` checks that the release is still a draft (a step under a per-tag concurrency group), uploads every asset and publishes the release last.
- `publish-templates` deliberately has no draft check. It touches neither the tag nor the release, so a dispatch can restore templates for a published release (`adopt-draft-first-release` design D3).
- A template failure today therefore still ships the cli binaries.

No Go code, command, flag or exit code changes. The config rules about command syntax, flags and example output do not apply; the script's messages are specified instead.

## Goals / Non-Goals

**Goals:**

- A template whose `debugValues` does not render under the cli being built cannot publish, and fails the PR that introduces it.
- No template is pushed unless every template passed every gate in the same run.
- A template whose tree differs from the artifact GHCR holds at its declared version fails the PR check and the release run before anything is pushed, whatever route and history the change took to reach `main`.
- A template failure at release time leaves the cli release a draft, never published without its templates.

**Non-Goals:**

- A render gate inside `opm module publish` for every module (see R2).
- Making the workspace `deps:update:templates` task bump template versions. That is a workspace-repo change; R5 describes it and recommends where it lands.
- Making `Template Publish Gates (dry-run)` a required check, or enabling the `Protected` ruleset. Both are owner settings, outside any change.
- Republishing or retracting `advanced` v1.0.2. Published versions are immutable. `opm module init` resolves the newest stable version (v1.0.3), so v1.0.2 is reached only by an explicit `advanced@1.0.2` pin.
- Changing `hack/fixtures.sh` (see R3).

## Decisions

### D1. Vet every template, in both modes, through the shared script

The gate phase runs `"$opm" module vet "./$dir"` on every `templates/*/` tree, whether or not GHCR already holds its version, in both `--dry-run` and publish mode. A template that is already published and unchanged is still vetted, because the cli under test may no longer render it, and `opm module init` from that cli would scaffold a broken module.

The vet runs through the script, not as a separate workflow step, so the PR job and the release job keep one gate list (the `pr.yml:99-101` comment's promise).

### D2. Two phases: gate all, then publish

```
preflight: every tool the script calls (cue, curl, jq, unzip, sha256sum, diff) is on PATH, or exit 1
gate phase (both modes), for each templates/<t>/:
  version   := cue eval ./identity -e Version               # unchanged
  tidy      := opm module tidy --check ./<dir>               # existing gate, moved here
  vet       := opm module vet ./<dir>                        # new
  dryrun    := opm module publish --dry-run ./<dir>          # now also in publish mode
     GO                                  -> pass (unpublished; publish mode pushes it)
     only refusal "already holds"        -> D3 content check
     any other refusal                   -> fail
  record failure, continue to next template (report all, not first)
any failure -> print "gates failed for: <list>; nothing published", exit 1
--dry-run   -> exit 0
publish phase (release only), for each template:
  GHCR holds v<version> -> skip (unchanged filter, 0011:D15)
  else                  -> opm module publish ./<dir>
```

`opm module publish` re-runs its own gates during the push. That duplicate is intentional and cheap: the dry-run in the gate phase is what makes "every gate before any push" true across templates.

### D3. Changed implies bumped: the published artifact must equal the tree

The check runs when a template's dry-run refuses only for "already holds". That refusal comes from opm's own version listing (`gateAlreadyPublished`, `internal/publish/registry.go:33`), so GHCR is known to hold `v<version>`. The template then passes only when that published artifact holds exactly the tree's files.

```bash
# zip_layer <manifest-file> -> digest of the manifest's single application/zip layer (error unless exactly one)
# published_zip <repo> <tag> <out>
#   anonymous pull token (scope=repository:<repo>:pull; never GHCR_AUTH)
#   GET manifest <tag>; zip_layer; digest must match ^sha256:[0-9a-f]{64}$
#   GET blob <digest> (-L: GHCR redirects to a signed URL); sha256 of the file must equal the digest
# tree_zip <dir> <version> <out>
#   (cd <dir> && cue mod publish v<version> --out <tmp>)   # local OCI image layout, no push
#   index.json -> its single manifest -> zip_layer -> copy that blob
# same_as_published <t> <dir> <version>
#   published_zip, tree_zip, unzip both into published/ and tree/
#   diff -ru published tree:  0 -> pass   1 -> fail with the diff   other -> fail
```

Every step checks its own status, and any error fails the template with a message naming the step. An error is never read as "not published". `gate` runs inside a `||` list, where `errexit` is off, so the explicit checks are what make this hold.

**Canonical form.** The comparison uses the set of (path, bytes) pairs that the module zip holds, not the zip's bytes. It is built with the publisher's own code:

- **Same file selection.** For `source: kind: "self"`, `cue mod publish --out` calls `modzip.CreateFromDir` (`cmd/cue/cmd/modpublish.go:185` at v0.17.1), the function `opm module publish` zips with. The publish gates refuse any other source kind (`gateSourceSelf`, `internal/publish/gates.go:155`), and the dry-run has passed every gate but "already holds" before the comparison runs. So the tree side applies modzip's own omissions (VCS directories, nested modules, `cue.mod/local-module.cue`, symlinks) instead of a hand-written copy of them.
- **No push.** `--out` implies `--dry-run` and writes an OCI image layout to a local directory. Measured: with `opmodel.dev/templates` routed to a dead address it still succeeds, so it never contacts the module's own repository. It does resolve the template's dependencies, which vet has already fetched.
- **Files, not zip bytes.** modzip defines a module as its files: "File permissions and timestamps are also ignored" (`mod/modzip/zip.go:39-40`). Zip encoding (deflate output, header fields) depends on the toolchain that built the publisher, so comparing zip digests would turn a Go or cue upgrade into a false "changed". Today the bytes match as well: all three tree zips are byte-identical to the GHCR v1.0.3 layers. That confirms the selection, but the gate does not rely on it.
- **Comments are content** (R6), so the comparison is byte-exact per file and not CUE-semantic.
- **Version skew can only fail closed.** The installed cue (v0.17.1, both jobs) and the cli's `cuelang.org/go` may diverge after a dependency bump. The published side is fetched, never rebuilt, so a file that one selection keeps and the other omits shows up as a difference. Skew can cause a false failure, but it cannot hide a change to a file the published artifact holds.

The fetch reads `ghcr.io` directly with an anonymous pull token, as `published()` already does for the publish filter. The release job's `GHCR_AUTH` is not used for it. The templates are public, because `opm module init` fetches them anonymously.

**Why content and not git** (R4). The question is "would `opm module init` get this tree?", and GHCR holds the answer. Any git base is a proxy for it, and review 2 showed the previous-release-tag proxy drifting. The content check is stateless:

- Re-runs, shallow or tag-less checkouts, direct pushes, releases cut on top of an unbumped change, and re-runs of old PRs all reduce to the same question.
- A re-run after a partial publish needs no exception: the template the earlier attempt pushed is now published and identical, so it passes and is skipped.
- A change and its revert are no change (case H).
- Moving a template to an older published version fails (case I).

**Workflow wiring.** Nothing changes in either checkout: no `fetch-depth`, no `BASE_REF`. Both jobs already install cue v0.17.1. `curl`, `jq`, `unzip`, `sha256sum` and `diff` are on `ubuntu-latest`, and the D2 preflight names any that are missing.

### D4. Messages and exit codes

The script exits 0 when every gate passes (and, in publish mode, every needed push succeeds) and 1 otherwise. New lines, beside `opm module vet`'s own diagnostics:

- `==> <t>: 'opm module vet' failed; a template must render its own debugValues`
- `==> <t>: v<version> already published and identical to the tree`
- the content failure, with the diff (header timestamps stripped, at most 60 lines):

  ```
  ==> minimal: the tree differs from the published v1.0.3:
  diff -ru published/cue.mod/module.cue tree/cue.mod/module.cue
  --- published/cue.mod/module.cue
  +++ tree/cue.mod/module.cue
  @@ -7,7 +7,7 @@
  -		v: "v4.4.4"
  +		v: "v4.4.3"
      Published versions are immutable: run 'opm module version set <semver> ./templates/minimal/'.
      If this branch is behind main, update it first: main may already publish that version.
  ```

- `==> <t>: cannot fetch the published v<version> from GHCR; a fetch error never counts as unpublished`
- `downloaded blob does not match <digest>`, then the fetch line above
- `==> <t>: cannot build the tree's module zip with 'cue mod publish --out'`
- `==> <t>: cannot unpack a module zip`
- `==> <t>: comparing the tree with the published v<version> failed`
- `==> missing tool: <name>`
- `==> gates failed for: <t> [<t> ...]; nothing published`

### D5. publish-templates runs before goreleaser builds the release

The new release job graph is `release-please -> publish-templates -> goreleaser`:

- `goreleaser` gains `needs: [release-please, publish-templates]`. It still reads release-please's outputs. Its job condition stays `always() && (releases_created == 'true' || workflow_dispatch)`, because a dispatch skips release-please and a skipped need would skip the job.
- Its steps become:
  1. The draft check, unchanged, still first and still inside the per-tag concurrency group.
  2. **New**: "Require the published templates". It fails unless `needs.publish-templates.result == 'success'`, with an `::error::` line and then plain lines (the draft check's shape): `publish-templates did not succeed (<result>); release <tag> stays a draft and no binary is built.` and the recovery hint.
  3. Checkout, Go and goreleaser, unchanged.
- `publish-templates` keeps its condition, its permissions and its lack of a draft check, for the same reason as before: it touches neither the tag nor the release, and a dispatch for a published release that lost its templates stays valid.
- The header comment and its recovery runbook gain one entry for a failed template job.

The template requirement is a step after the draft check, not a job-level `if`. With a job-level condition, a failed template job would *skip* goreleaser. A dispatch on a published release whose template job fails (alpha.22 to alpha.27 do, see the prototype table) would then lose the draft check's "already published, cut the next patch" refusal that the spec scenario "Manual recovery refuses a published release" requires. As a step, every existing draft-check outcome is unchanged, and the template requirement adds one more refusal after them. `always()` also runs goreleaser for a cancelled template job, and the step then refuses with the result `cancelled`.

| Situation | Before | After |
| --- | --- | --- |
| A template gate fails on the release push | cli release published, templates not pushed | release stays a draft; goreleaser fails at step 2 |
| Transient template failure (GHCR, network) | same as above | re-run the failed jobs, or dispatch with the tag: templates, then binaries |
| Dispatch on a draft | both jobs in parallel | templates first, then goreleaser finishes the draft |
| Dispatch on a published release | templates job runs; goreleaser refuses at the draft check | same: the templates job runs first (pushes nothing once every version is published and identical), then goreleaser refuses |
| Dispatch on a tag with no release | templates job runs; goreleaser refuses | same |

**Cost.** The binaries wait for the template job: building `opm`, installing cue, and about 10 s of gates measured locally, so one to two minutes on a runner. A gate failure at release time leaves a draft whose tag cannot move. The fix is a bumped template on `main` and the next patch, and the owner removes the abandoned draft in the browser (the workflow never deletes a release; release-please anchors the next release PR on the forced tag while the previous release is a draft, per draft-first U2). The release PR's `template-gates` runs the same gates on the same tree first, so this happens only when that check was red and the PR merged anyway, or GHCR changed in between.

Two runs for one tag (a dispatch racing a push run) can both reach the push of the same new template version. opm's already-published gate refuses the second push, and that run's goreleaser stops at step 2, while the other run finishes the release. No concurrency group is added to `publish-templates`.

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

### Prototype of D2/D3 (revision 3)

The prototype is the revised script (shellcheck clean). It ran in a `file://` clone of this worktree with its real tags. `opm` was built from this worktree (no Go changes since 437bebd), and the cue was v0.17.1. Every run:

- `env -i`, with empty `HOME`, `DOCKER_CONFIG` and `XDG_CONFIG_HOME`. Before each run they were checked to hold nothing but caches, and after the last run `docker/` and `xdg/` were still empty.
- `OPM_BIN` pointed at a shim that execs the real `opm` for everything except `module publish` without `--dry-run`. That call it logs as `WOULD PUBLISH <dir>` and exits 0.
- `cue` on `PATH` was a shim that refuses `cue mod publish` without `--out`. No log contains a refusal.
- No registry was written, and no local registry container was used.

| Case | Mode | Result |
| --- | --- | --- |
| A: this worktree, unchanged | both | exit 0; each of minimal, standard, advanced "v1.0.3 already published and identical to the tree"; publish: three "skipped", no WOULD PUBLISH |
| B: advanced fix removed, bumped to 1.0.4 | both | exit 1, vet fails advanced (unresolved disjunction), `gates failed for: advanced; nothing published`; no WOULD PUBLISH |
| C: one-line edit, minimal pin catalogs/opm v4.4.4 to v4.4.3, no bump | both | exit 1, the diff shows exactly that line, bump message; no WOULD PUBLISH |
| D: C plus `version set 1.0.4` | dry-run | exit 0, minimal `GO` at v1.0.4, the others identical |
| D | publish | exit 0, exactly one `WOULD PUBLISH ./templates/minimal/` |
| E: comment-only edit to standard, no bump | dry-run | exit 1, diff shows the comment line |
| F: unrelated file outside `templates/` | dry-run | exit 0 |
| G: C, then an unrelated commit on top (a direct push followed by an innocent PR) | dry-run | exit 1, minimal differs |
| H: C, then its revert | dry-run | exit 0 |
| I: advanced identity set back to 1.0.2 on the current tree | both | exit 1: the diff against published v1.0.2 shows both `updateStrategy` lines in `components.cue` and the stale pins (catalogs/opm v4.4.0, core v2.0.0-alpha.10) |
| R5: C tagged as release v1.0.0-beta.5 (a release on top of an unbumped change) | publish | exit 1, minimal differs, nothing published |
| R-PR: an unrelated commit on top of R5 (the next PR) | dry-run | exit 1, minimal differs |
| R6: that commit tagged v1.0.0-beta.6 (the next release) | publish | exit 1, minimal differs, nothing published |
| J: tag v1.0.0-beta.1 (the release that published every 1.0.3) | publish | exit 0, all identical, all skipped |
| alpha.22 (the release that published every 1.0.2) | publish | exit 1 for advanced's vet only; minimal and standard "v1.0.2 already published and identical to the tree" |
| alpha.25, alpha.26, alpha.27 (re-pinned at 1.0.2) | publish | exit 1 before any push: vet fails advanced; minimal and standard differ from v1.0.2 |
| N: new template `zzprobe` (no GHCR repository) | both | exit 0; dry-run `GO`; publish exactly one `WOULD PUBLISH ./templates/zzprobe/` |
| no history: `git archive` export (no `.git`) and a `file://` depth-1 clone (`--is-shallow-repository` true) | dry-run (export: both) | exit 0, all identical |
| faults (curl or cue shim): token, manifest, blob, digest mismatch, `cue mod publish --out` | dry-run (blob: both) | exit 1, every template named with the matching line of D4, no WOULD PUBLISH |

The alpha.26 row is review 2's replay: there the tag-based check passed minimal and standard, and the content check fails them. The alpha.22 row is a second positive control. The v1.0.2 that alpha.22's own pipeline published compares equal to alpha.22's trees, so the canonical form reproduces a publish made by an older opm as well.

The extended e2e test (section 4) was prototyped in revision 2 and is unchanged: all three subtests pass on `main` (about 10 s together, no cluster, in-process registry), and the advanced subtest fails with the unresolved disjunction on case B's template tree.

### R1. vet versus build

**Options considered**:
1. `opm module vet`: renders as build does, prints one validation line per object, no manifests, exit 2 on failure. Its render is a spec requirement (`mod-vet`).
2. `opm module build > /dev/null`: same verdict (vet.go:40-41), but its diagnostics are less author-facing and it emits manifests to discard.
3. Only the e2e scaffold test (`TestE2E_ModInit_ThenVet`), extended to every template.

**Decision**: vet in the script, and also extend the e2e test (section 4). **Rationale**: vet is the command the spec defines as the author's pass/fail verdict. The e2e test alone does not gate the release job, which runs only the script, and it checks the re-identified scaffold rather than the tree that is published; the script alone does not check the scaffold a user receives. Both are cheap.

### R2. Where the render gate lives

**Options considered**:
1. In `publish-templates.sh` (chosen).
2. A new gate inside `opm module publish` (render `debugValues`, refuse on failure) for every module publisher.
3. A separate workflow step in `pr.yml` and `release.yml`.

**Decision**: option 1. **Rationale**: option 2 changes `opm module publish` for every publisher (modules fleets, fixtures that deliberately do not render), a product decision that belongs in an enhancement amending 0011, not in a CI fix. Option 3 splits one gate list across two YAML files, and the release path would drift from the PR path.

### R3. Where changed-implies-bumped lives

**Options considered**:
1. Inline in `publish-templates.sh` (chosen).
2. Run `FIXTURES_DIR=templates hack/fixtures.sh check`.
3. Defer to a follow-up change.

**Decision**: option 1. **Rationale**: option 2 would make `hack/fixtures.sh`, which must stay byte-identical with opm-operator's copy (workspace `task fixtures:lint`), carry template-only gates (tidy, vet) and a different comparison (GHCR content, not a merge-base). Its policy text also assumes the testing domain. Option 3 leaves open the exact failure behind GHCR v1.0.2 (stale pins), while the same file and jobs are already being edited. One observation for the fixtures plan: `hack/fixtures.sh check` compares against a merge-base, so it has the direct-push gap this change closes for templates, and the same content check would close it there. That is the fixtures plan's decision, not this change's.

### R4. What the bump check compares against

**Context**: review 2's major finding. The previous-release-tag base moves past an unbumped change once a release lands on top of it, and the stale content is then never published.

**Explored**: the prototype table above. Review 2's `proto-fix2.sh` (the oldest reachable release tag that declares the current version) was also read.

**Options considered**:
1. The merge-base of `BASE_REF` and `HEAD`, PR job only (revision 1). It misses every change that reaches `main` without a PR, which includes the three commits behind stale v1.0.2.
2. The previous release tag, both modes (revision 2). Its base moves forward over an unbumped change: review 2's alpha.26 replay and its synthetic beta.6 pass.
3. The oldest reachable release tag whose tree declares the current version (review 2's fix). It repairs option 2, but it is still a proxy that infers GHCR's content from git:
   - It needs full history and tags.
   - It fails closed when no tag declares a version, for example when the introducing release's template job failed.
   - It keeps `version_at`-style evaluation of old trees.
4. The content GHCR holds at the declared version (chosen).
5. Auto-bump unpublished changes at release time. That would publish a version no commit declares, against the identity package being the single source of the version.

**Decision**: option 4. **Rationale**:
- It compares with what `opm module init` actually fetches, so there is no proxy to drift, and it is stateless (D3).
- The re-run exception, `version_at`, `sort -V` and `fetch-depth: 0` all go away.
- Revision 2 rejected this option because it seemed to need a re-implementation of CUE's module-zip file selection. `cue mod publish --out` runs the publisher's own `modzip.CreateFromDir` instead, and the measured trees reproduce the published bytes exactly.
- The cost is one anonymous download of a few KB per published template per run, plus the GHCR dependency the dry-run already has.

### R5. The workspace `deps:update:templates` task (recommendation, workspace repo)

Root `Taskfile.yml:118-150` re-pins each `cli/templates/*/cue.mod/module.cue` with `cue mod get` and `cue mod tidy`, and never touches the template's version. After this change, every cli PR that `task deps:update` produces and that moves a template pin fails `template-gates` until someone runs `opm module version set` by hand.

**Recommendation**: the task bumps the patch version of every template whose `module.cue` it changed, but only when GHCR already holds the template's declared version. If the declared version is still unpublished (bumped earlier in the same release cycle), that pending version already covers the new pins, and a second bump would skip a version for nothing. Shape, mirroring `.tasks/deps/fixtures.sh` lines 21-54 (`bump_deps` reports whether anything moved, then `opm module version set` on the next patch):

```
moved := module.cue differs after cue mod get + tidy
if moved and GHCR holds v<Version>:      # same anonymous manifest probe as publish-templates.sh
  opm module version set <Version with patch+1> .
```

- It needs `opm` on `PATH`, which `deps:pins:fixtures` already requires.
- Move the loop into `.tasks/deps/templates.sh`, beside `fixtures.sh`, and share the patch-increment helper.
- Update the `deps:update` row in the workspace `AGENTS.md` to say it also bumps changed templates.
- The cli commit stays `fix(deps)`, since template pins are shipped.

**Where it lands**: as its own commit in the workspace PR that reworks `.tasks/deps/fixtures.sh` (the fixtures plan), which has the same directory, the same bump idiom and one shared helper. The latest-tag PR (`.tasks/deps/latest-tag.sh`) is unrelated in substance. If the fixtures PR stalls, ship it alone. Either way it should merge before, or together with, this cli change, so the first `task deps:update` after it does not produce a red cli PR.

### R6. Comment-only edits need a bump

The gate fails a comment-only template edit at a published version (case E). That is intended:
- `opm module init` copies comments into users' modules, so a comment change is a content change of the artifact.
- The fixtures rule is the same.
- Commit a50cd73 bumped standard from 1.0.1 to 1.0.2 for comments alone.

Excluding comments would need a CUE-aware comparison instead of a byte comparison of files.

### R7. enhancement.yaml

None. The change fulfils the "vetted" intent of 0011:D25, but 0011 is archived with status `delivered`; the `template-modules` delta cites D25 as its source. No new decision is implemented.

## Risks / Trade-offs

- [Release-time failure] After D5, a template gate failure at release leaves the release a draft, never a published cli without its templates.
  - A transient failure (GHCR, network) is fixed by re-running the failed jobs.
  - A content failure burns the tag: bump the template on `main`, cut the next patch, and the owner removes the draft.
  - The release PR runs the same gates first, but that check is advisory, because no status check is required on `main`. **Recommendation (owner setting, not part of this change)**: make `Template Publish Gates (dry-run)` a required status check on `main`, so a red release PR cannot merge and no tag is burnt.
- [Direct pushes] A required check binds PRs only. A direct push that changes a template without a bump still lands. After this change, every following PR and every release fails until a bump lands, including releases cut on top of it (cases G, R5, R-PR, R6), and the bump then publishes the content. The stale content never ships silently.
- [GHCR dependency] The gate reads GHCR for every published template on every run: a token, a manifest and a blob of a few KB each, anonymously. An outage fails the gate, as the dry-run's version listing already does. The content fetch reads only `ghcr.io`, as `published()` does, while "already holds" comes from opm's mapping (`OPM_REGISTRY`, which the script defaults to GHCR). In CI the two agree. A local run with another mapping can disagree: a version that registry holds and GHCR lacks fails the fetch, and a version only GHCR holds counts as unpublished for that run. This is a local-only case and does not apply to either workflow.
- [Toolchain skew] The tree zip comes from the installed cue (v0.17.1), and the cli publishes with its own `cuelang.org/go`. A future divergence in modzip's file selection can only cause a false failure (D3), which would show up as a diff naming the file.
- [Re-run of an old PR] A re-run compares the PR's tree with GHCR as it is now. It fails only when GHCR now holds the PR's declared version with other content, which is a real conflict, and the message says to update the branch first.
- [Vet stops rendering] The gate's strength rests on `opm module vet` rendering. The `mod-vet` spec requires it; a change that drops the render would have to change that spec.
- [Friction] Every template edit needs a bump, comment-only ones included (R6), and `task deps:update` cli PRs need a manual `version set` until R5 lands. A bad direct push turns unrelated PRs red until it is fixed (case G). That is the point, but it is noisy.
- [Release latency] The binaries now wait one to two minutes for the template job (D5).

## Migration Plan

None. The current templates (all at v1.0.3) pass every new gate unchanged, and each compares identical to its published artifact (case A).
