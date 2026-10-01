## Revisions

- **Revision 2** (2026-10-01, superseded): changed-implies-bumped compared each template with the previous release tag. Review 2 showed that this base moves past an unbumped change once a release lands on top of it. At v1.0.0-alpha.26, minimal and standard passed as "unchanged since alpha.25" although they differ from the v1.0.2 that alpha.22 published, and a synthetic release on top of an unbumped change passed the next PR and the next release.
- **Revision 3** (2026-10-01):
  - Changed-implies-bumped compares **content against GHCR**, not git. For an already-published version, the script fetches the published module zip anonymously, builds the tree's zip with the publisher's own file selection, and compares the two file by file (D3). It needs no history, no tags, no `fetch-depth: 0` and no re-run exception, and every fetch or comparison error fails.
  - The release workflow now runs `publish-templates` before `goreleaser` builds, so a template failure leaves the release a draft instead of shipping binaries without templates (D5).
  - The verification harness strips `<dir>` before forwarding `"$@"`, and puts `cue` behind a shim too.
- **Revision 4** (2026-10-01, this text), after review 3. Review 3 passed five trees that must fail:
  - S1: an unbumped template with a symlinked component file;
  - S2: a bumped template whose `debugValues` live only in a symlink;
  - P: a prerelease bump;
  - M: a module path renamed away from its directory;
  - V: a broken identity, which aborted the run and hid a second failure.

  The fixes:
  - **The tree equals its zip** (D6, new). For every template, GO ones included, the tree's file list must equal its module zip's, and the gate refuses symlinks, special files and nested `cue.mod` directories by name. D3 no longer calls modzip's omissions harmless: an omitted file is one vet read and no user receives.
  - **Version order** (D7, new). A GO version is stable and above the highest published stable version of its major. An already-published version must be that highest version, so a rollback to an older version with its old content fails too.
  - **Identity and verdict** (D2). The module path must be `opmodel.dev/templates/<dir>@v<major>`. The publish phase acts on the gate's verdict and no longer probes GHCR a second time. The identity is read inside `gate`, so a broken one fails only its own template. The refusal count is matched anchored.
  - Both workflows install the cue that `go.mod` requires, so the tree zip and `opm module publish` share one modzip (D3).
  - D2, D3, D4, D5, Risks and the prototype table are revised, and R8 is new. D1, R1 to R7 stand.

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
- What vet reads is what publishes: a tree holding anything its module zip omits fails, whether or not its version is published.
- The version `opm module init` resolves (the highest published stable version of the template's major) is the version on `main`, or `main` carries a higher, unpublished one.
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
preflight: every tool the script calls (cue, curl, jq, unzip, sha256sum, diff, find, sort, ...) is on PATH, or exit 1
gate phase (both modes), for each templates/<t>/; the first failing step fails the template:
  identity  := cue eval ./identity -e Version, -e ModulePath  # inside gate, so a broken one fails only <t>
               Version is stable X.Y.Z; ModulePath == opmodel.dev/templates/<t>@v<major>
  layout    := no symlink, no special file, no nested cue.mod   # D6
  tidy      := opm module tidy --check ./<dir>                # existing gate, moved here
  vet       := opm module vet ./<dir>                         # new
  tree zip  := cue mod publish v<version> --out; file list == the tree's file list   # D6
  dryrun    := opm module publish --dry-run ./<dir>           # now also in publish mode
     GO                                       -> versions (D7): above the highest published stable -> verdict publish
     only refusal "<path> already holds v<v>"  -> versions (D7): is that highest; D3 content check -> verdict skip
     any other refusal                        -> fail
  record failure, continue to next template (report all, not first)
any failure -> print "gates failed for: <list>; nothing published", exit 1
--dry-run   -> exit 0
publish phase (release only), for each template in order, acting on its verdict:
  skip    -> "already published; skipped" (the caller-side filter of 0011:D15)
  publish -> opm module publish ./<dir>; a failure exits 1
```

`opm module publish` re-runs its own gates during the push. That duplicate is intentional and cheap: the dry-run in the gate phase is what makes "every gate before any push" true across templates.

**The publish phase acts on the verdict.** Revision 3 probed GHCR a second time in the publish phase and skipped any version it found. That probe was keyed on the directory name, not the module path, so case M (the module path renamed to `minimalx`, version kept) got `GO` from the dry-run for the new path and was then skipped by the probe for the old one: exit 0, nothing published, a renamed template on `main`. A second probe also races: a version another run pushed between the two phases would be skipped without its content ever being compared. Now the gate's verdict is final. A version published in between makes `opm module publish` refuse ("already holds"), the run fails, and a re-run compares the content first.

**The module path is the directory.** The gate requires identity `ModulePath` to equal `opmodel.dev/templates/<dir>@v<major>`, with `<major>` taken from `Version`. The publish dry-run's own "Coordinates agree" gate then ties `cue.mod/module.cue` to the same path. So the directory, the GHCR repository that every check reads, and the repository `opm module publish` pushes to are one name.

**Every template is reported.** The identity is evaluated inside `gate`. Before, `version=$(cue eval ...)` ran at the top level under `errexit`, so case V (a syntax error in minimal's identity) aborted the run at minimal and never reported standard's unbumped comment edit.

**The refusal match is anchored.** "Only refusal is already-published" requires the summary line to end in ` 1 refusal` (`grep -Eq ' 1 refusal$'`), so `11 refusals` cannot match, and the headline to name the template's own path and version: `opmodel.dev/templates/<t>@v<major> already holds v<version>`.

### D3. Changed implies bumped: the published artifact must equal the tree

The check runs when a template's dry-run refuses only for "already holds" and its version is the highest published stable version of its major (D7). That refusal comes from opm's own version listing (`gateAlreadyPublished`, `internal/publish/registry.go:33`), so GHCR is known to hold `v<version>`. The template then passes only when that published artifact holds exactly the tree's files.

```bash
# zip_layer <manifest-file> -> digest of the manifest's single application/zip layer (error unless exactly one)
# published_zip <repo> <tag> <out>
#   anonymous pull token (scope=repository:<repo>:pull; never GHCR_AUTH)
#   GET manifest <tag>; zip_layer; digest must match ^sha256:[0-9a-f]{64}$
#   GET blob <digest> (-L: GHCR redirects to a signed URL); sha256 of the file must equal the digest
# tree_zip <dir> <version> <out>          # built for every template before the dry-run (D6)
#   (cd <dir> && cue mod publish v<version> --out <tmp>)   # local OCI image layout, no push
#   index.json -> its single manifest -> zip_layer -> copy that blob
# same_as_published <t> <dir> <version>
#   published_zip, unzip it and the tree zip into published/ and tree/
#   diff -ru published tree:  0 -> pass   1 -> fail with the diff   other -> fail
```

Every step checks its own status, and any error fails the template with a message naming the step. An error is never read as "not published". `gate` runs inside a `||` list, where `errexit` is off, so the explicit checks are what make this hold.

**Canonical form.** The comparison uses the set of (path, bytes) pairs that the module zip holds, not the zip's bytes. It is built with the publisher's own code:

- **Same file selection.** For `source: kind: "self"`, `cue mod publish --out` calls `modzip.CreateFromDir` (`cmd/cue/cmd/modpublish.go:186` at v0.17.1), the function `opm module publish` zips with. The publish gates refuse any other source kind (`gateSourceSelf`, `internal/publish/gates.go:155`). So the tree zip is the zip `opm module publish` would push, built by the publisher's own code instead of a hand-written copy of its rules.
- **The omissions are not harmless.** modzip silently leaves out symlinks and other irregular files, nested modules (any directory holding a `cue.mod`), VCS directories, `cue.mod/vendor/`, `cue.mod/local-module.cue` and a root `.hg_archival.txt` (`mod/modzip/zip.go:279-337`, `:776-834`). Revision 3 called this harmless because both sides of the comparison apply the same selection. Review 3 showed the gap: vet and tidy read the **tree**, the comparison reads the **zip**. A symlinked component file (S1) is vetted, absent from both zips, and so "identical" to the published artifact. A GO template whose `debugValues` live only in a symlink (S2) passes vet and publishes an artifact without them, which does not render. D6 closes this for every template: the tree must equal its zip.
- **No push.** `--out` implies `--dry-run` and writes an OCI image layout to a local directory. Measured: with `opmodel.dev/templates` routed to a dead address it still succeeds, so it never contacts the module's own repository. It does resolve the template's dependencies, which vet has already fetched.
- **Files, not zip bytes.** modzip defines a module as its files: "File permissions and timestamps are also ignored" (`mod/modzip/zip.go:39-40`). Zip encoding (deflate output, header fields) depends on the toolchain that built the publisher, so comparing zip digests would turn a Go or cue upgrade into a false "changed". Today the bytes match as well: all three tree zips are byte-identical to the GHCR v1.0.3 layers. That confirms the selection, but the gate does not rely on it.
- **Comments are content** (R6), so the comparison is byte-exact per file and not CUE-semantic.
- **One modzip.** Both workflows install cue at the version the cli's `go.mod` requires: `go install "cuelang.org/go/cmd/cue@$(go list -m -f '{{.Version}}' cuelang.org/go)"`, run after checkout and `setup-go`. The tree zip and `opm module publish` then run the same modzip, and a `cuelang.org/go` bump moves both together; a hard-coded `@v0.17.1` would silently skew at the first Dependabot bump. The five other installs of cue v0.17.1 (`pr.yml` unit, fixtures and e2e, `ci.yml`, `publish-fixtures.yml`) keep their pin: none of them compares zips with what the cli publishes, so moving them is a separate cleanup. The published side is fetched, never rebuilt, so a skew in a local run with another cue can only show up as a difference, never hide one in a file the published artifact holds.

Every GHCR read (the token, the tags list of D7, the manifest and the blob) goes to `ghcr.io` directly with an anonymous pull token. The script no longer reads `GHCR_AUTH`, so the release step stops passing it; `docker login` stays, because `opm module publish` pushes with those credentials. The templates are public, because `opm module init` fetches them anonymously.

**Why content and not git** (R4). The question is "would `opm module init` get this tree?", and GHCR holds the answer. Any git base is a proxy for it, and review 2 showed the previous-release-tag proxy drifting. The content check is stateless:

- Re-runs, shallow or tag-less checkouts, direct pushes, releases cut on top of an unbumped change, and re-runs of old PRs all reduce to the same question.
- A re-run after a partial publish needs no exception: the template the earlier attempt pushed is now published, identical and the highest version, so it passes and is skipped.
- A change and its revert are no change (case H).
- Moving a template to an older published version fails (cases I and RB, by D7 before the content is compared).

**Workflow wiring.** Nothing changes in either checkout: no `fetch-depth`, no `BASE_REF`. The cue install step in both jobs derives its version from `go.mod` (above). `curl`, `jq`, `unzip`, `sha256sum`, `diff` and GNU `find` are on `ubuntu-latest`, and the D2 preflight names any that are missing.

### D4. Messages and exit codes

The script exits 0 when every gate passes (and, in publish mode, every needed push succeeds) and 1 otherwise. New lines, beside `opm module vet`'s own diagnostics:

- `==> <t>: cannot read Version and ModulePath from ./templates/<t>/identity`, after cue's own error
- `==> <t>: v<version> is not a stable version; 'opm module init' resolves only stable template versions`
- `==> <t>: ModulePath is <path>; the template in templates/<t>/ at v<version> must be opmodel.dev/templates/<t>@v<major>`
- `==> <t>: symbolic links never publish (a module zip omits them); replace them with files:`, then one indented path per line; likewise `special files never publish; remove them:` and `a nested cue.mod makes a nested module, which never publishes; remove it:`
- `==> <t>: 'opm module vet' failed; a template must render its own debugValues`
- `==> <t>: the module zip does not hold exactly the tree's files, so the published module would not be the vetted one:`, then `    only in the tree: <path>` lines (at most 60)
- `==> <t>: comparing the tree with its module zip failed`
- `==> <t>: cannot list the published versions on GHCR; a fetch error never counts as unpublished`
- `==> <t>: v<version> is not above the highest published v<H>; run 'opm module version set <semver> ./templates/<t>/' with a higher version.`, then the "behind main" hint
- `==> <t>: v<version> is published, but 'opm module init' fetches the highest published v<H>; the tree must be at v<H>, or above it with a bump.`, then the hint
- `==> <t>: GHCR lists v<version>, but the publish dry-run found it unpublished`, and its converse `the publish dry-run found v<version> published, but GHCR does not list it`
- `==> <t>: v<version> passes every gate and is not published yet`
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
- publish mode: `==> <t>: v<version> already published; skipped`, `==> <t>: publishing v<version>`, and on a failed push `==> <t>: publishing v<version> failed; nothing after it was published, and a re-run skips what this run pushed`

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
| Dispatch on a published release | templates job runs; goreleaser refuses at the draft check | same: the templates job runs first (pushes nothing; it fails once a later release published a higher template version, D7), then goreleaser refuses at the draft check |
| Dispatch on a tag with no release | templates job runs; goreleaser refuses | same |

**Cost.** The binaries wait for the template job: building `opm`, installing cue, and about 10 s of gates measured locally, so one to two minutes on a runner. A gate failure at release time leaves a draft whose tag cannot move. The fix is a bumped template on `main` and the next patch, and the owner removes the abandoned draft in the browser (the workflow never deletes a release; release-please anchors the next release PR on the forced tag while the previous release is a draft, per draft-first U2). The release PR's `template-gates` runs the same gates on the same tree first, so this happens only when that check was red and the PR merged anyway, or GHCR changed in between. A draft left behind for long enough can also become unfinishable: once a later release publishes a higher version of one of its templates, its own GO version is no longer above the highest (D7), so it too is replaced by the next patch.

Two runs for one tag (a dispatch racing a push run) can both reach the push of the same new template version. The publish phase acts on its gate verdict and does not probe GHCR again (D2), so the second run calls `opm module publish`, whose already-published gate refuses the push; that run exits 1 and its goreleaser stops at step 2, while the other run finishes the release. A re-run of the failed run then sees the version published, compares its content and skips it. No concurrency group is added to `publish-templates`.

### D6. The tree equals its module zip, for every template

For every template, whether its dry-run says GO or "already holds", the gate requires the module zip to hold exactly the tree's files, so the published artifact is the tree vet rendered:

```bash
# layout <t> <dir>: refuse by name, before tidy and vet read anything
find "$dir" -type l                       # symbolic links: modzip omits them
find "$dir" ! -type d ! -type f           # special files: modzip omits them
find "$dir" -mindepth 2 -iname cue.mod    # a nested module: modzip omits its whole directory
# zip_holds_tree <t> <dir> <tree zip>: after vet, with the zip from cue mod publish --out
diff <(unzip -Z1 "$zip" | sort) <(find "$dir" ! -type d -printf '%P\n' | sort)   # LC_ALL=C
#   0 -> pass   1 -> fail, listing "only in the tree: <path>"   other -> fail
```

- The layout checks name the three omissions a template author can plausibly commit. The listing comparison catches everything else modzip leaves out (a `.hg_archival.txt`, `cue.mod/vendor/`, `cue.mod/local-module.cue`, a VCS directory), without copying modzip's rules: whatever its version omits shows up as "only in the tree".
- It runs on GO templates too. S2 is a GO template: its version is unpublished, so the content check never ran, and revision 3 would have published a zip without `debugValues`.
- Directories are not compared: a module zip holds no directory entries, and empty directories are not module content (`mod/modzip/zip.go:39`).
- A git checkout cannot hold a special file, but a local tree can, and a symlink and a nested module can both be committed.

### D7. Version order: what `opm module init` resolves is the tree

`opm module init <name>` floats to the highest published **stable** version of the baked major (`internal/scaffold/scaffold.go:107`, `highestStable`; baked `DefaultMajor` in `Official`, `internal/scaffold/ref.go:38`). The content check alone compares a tree with the version it declares, which is not always the version users get:

- P: `version set 1.0.4-rc.1` passes as GO and publishes a prerelease that `opm module init` never resolves; users keep getting v1.0.3, which no longer matches `main`.
- RB: minimal's whole tree rolled back to its v1.0.2 content (`git checkout v1.0.0-alpha.22 -- templates/minimal`) is published and identical, so revision 3 passed it; users still get v1.0.3.

The gate lists the template's published tags (anonymous `GET /v2/<repo>/tags/list?n=1000`, following `Link: rel="next"`, because GHCR pages at 100 tags by default, measured on `opmodel.dev/core`). GHCR answers 403 `DENIED` to the anonymous token request for a repository it does not know (measured with a never-published name), and cue's modregistry, which the dry-run and `opm module init` list versions with, reads 403 and 404 as "no such module" (`isNotExist`, `mod/modregistry/client.go:566`). The script reads them the same way, so a new template has no versions in both listings; any other answer fails the template. From the tags it takes the stable versions of the declared major (`v<major>.X.Y`, no prerelease) and their highest, `H`. Then:

| Dry-run | Rule | Fails when |
| --- | --- | --- |
| any | `Version` is stable `X.Y.Z` (checked first, from the identity) | a prerelease or build suffix (P) |
| GO | `Version` is above `H` (or there is no `H`) and not among the tags | a version at or below `H`, for example a branch behind `main` (LOW) |
| already holds | `Version` equals `H` and is among the tags, then the D3 content check | an older published version (I, RB) |

- Versions compare numerically per field (`sort -t. -k1,1n -k2,2n -k3,3n`), which is exact for strict `X.Y.Z`.
- The dry-run's "already holds" comes from opm's version listing, the tags from the script's own read. If they disagree (a push between the two reads, or a local mapping other than GHCR), the template fails instead of trusting either.
- Only the declared major counts, because that is the float `opm module init` resolves. Moving a template to a new major changes its module path, so the new major starts with no `H`.

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

### Prototype of D2, D3, D6 and D7 (revision 4)

The prototype is the revised script (shellcheck clean). It ran in a fresh `file://` clone of this worktree with its real tags and `origin` removed. `opm` was built from this worktree (no Go changes since 437bebd), and cue was installed at the version `go list -m -f '{{.Version}}' cuelang.org/go` printed (v0.17.1). Every run:

- `env -i`, `/bin/bash`, `PATH` limited to the shims, `/usr/bin` and `/bin`, with empty `HOME`, `DOCKER_CONFIG` and `XDG_CONFIG_HOME`. Before each run they were checked to hold nothing but caches, and after the last run `docker/` and `xdg/` were still empty.
- `OPM_BIN` pointed at a shim that execs the real `opm` for everything except `module publish` without `--dry-run`. That call it logs as `WOULD PUBLISH <dir>` and exits 0, or with `FAULT=race` fails as an "already holds" refusal does.
- `cue` and `curl` on `PATH` were shims: the first refuses `cue mod publish` without `--out` (no log contains a refusal), the second injects the GHCR faults.
- No registry was written, and no local registry container was used.

Review 3's attack branches (S1, S2, P, M, V) were fetched into the clone at the reviewer's exact commits.

| Case | Mode | Result |
| --- | --- | --- |
| A: this worktree, unchanged | both | exit 0; each of minimal, standard, advanced "v1.0.3 already published and identical to the tree"; publish: three "skipped", no WOULD PUBLISH |
| D: minimal pin catalogs/opm v4.4.4 to v4.4.3 plus `version set 1.0.4` | dry-run / publish | exit 0, minimal `GO` and "passes every gate"; publish exactly one `WOULD PUBLISH ./templates/minimal/` |
| F (unrelated file), H (C plus its revert), X (mode bit only) | dry-run | exit 0 |
| J: tag v1.0.0-beta.1 (the release that published every 1.0.3) | publish | exit 0, all identical, all skipped |
| N: new template `zzprobe` (GHCR answers its token request with 403) | both | exit 0; dry-run `GO`; publish exactly one `WOULD PUBLISH ./templates/zzprobe/` |
| no history: `git archive` export and a depth-1 clone | dry-run (export: both) | exit 0, all identical |
| S1: symlinked component file, no bump | both | exit 1, "symbolic links never publish", `extra.cue` (revision 3: identical, exit 0) |
| S2: `debugValues` only via a symlink, bumped to 1.0.4 | both | exit 1, the same line, `debug.cue`; no WOULD PUBLISH (revision 3 published it; that zip's content alone fails vet with "not fully concrete") |
| S1, S2 with the layout check removed | dry-run | exit 1 from the listing alone: "only in the tree: extra.cue" / "debug.cue" |
| NM (nested `cue.mod`), HG (`.hg_archival.txt`), FIFO (uncommitted `mkfifo`) | dry-run | exit 1, each naming the path with its D4 line |
| P: pin change with `version set 1.0.4-rc.1` | both | exit 1, "is not a stable version"; no WOULD PUBLISH (revision 3 published it) |
| M: module path renamed to `minimalx`, directory and version kept | both | exit 1, "ModulePath is opmodel.dev/templates/minimalx@v1; ... must be opmodel.dev/templates/minimal@v1" (revision 3: exit 0, skipped by the second probe) |
| V: broken minimal identity plus an unbumped comment in standard | dry-run | exit 1, `gates failed for: minimal standard` (revision 3 aborted at minimal) |
| race: D with `FAULT=race` | publish | exit 1, "publishing v1.0.4 failed"; minimal is never reported skipped |
| B: advanced fix removed, bumped to 1.0.4 | both | exit 1, vet fails advanced (unresolved disjunction) |
| C (pin edit, no bump), E (comment only), CRLF (line endings only) | dry-run (C: both) | exit 1, the diff names the changed file |
| G: C, then an unrelated commit; R: C released as beta.5, a PR on top, released as beta.6 | dry-run / publish | exit 1 each, minimal differs, nothing published |
| I: advanced set back to 1.0.2; RB: minimal's whole tree rolled back to its v1.0.2 content | both | exit 1, "v1.0.2 is published, but 'opm module init' fetches the highest published v1.0.3" |
| LOW: D with GHCR listing a v1.0.9 (`FAULT=tags-high`) | both | exit 1, minimal "v1.0.4 is not above the highest published v1.0.9"; the others "v1.0.3 is published, but ... v1.0.9" |
| alpha.22, alpha.25, alpha.26, alpha.27 | publish | exit 1 before any push: vet fails advanced; minimal and standard at v1.0.2 fail the order rule |
| faults: token, tags, manifest, blob, digest mismatch, `cue mod publish --out` | dry-run (blob: both) | exit 1, every template named with the matching D4 line, no WOULD PUBLISH |
| preflight: `env -i PATH=/nonexistent /bin/bash <script> --dry-run` | dry-run | `==> missing tool: cue`, exit 1 |
| IGN: an untracked, gitignored `notes.log` in minimal (local only) | dry-run | exit 1, "Only in tree: notes.log" against the published zip: the documented local false failure |

Positive control: with the published-version order check removed from a scratch copy, alpha.22's minimal and standard compare "v1.0.2 already published and identical to the tree", so the canonical form still reproduces a publish made by an older opm, and RB passes (it is caught by D7 alone). A warm-cache dry-run of case A takes 9 s.

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

### R8. Which version a template may declare

**Context**: review 3, cases P and RB (D7). The content check proves that the tree equals the artifact at the version it declares, not that `opm module init` resolves that version.

**Options considered**:
1. No order rule (revision 3). A prerelease GO publishes a version `init` never resolves, and a rollback to an older published version with its old content passes.
2. Stable only, GO above the highest published stable version of the major, a published version equal to it (chosen).
3. The same, compared across every major in the repository.
4. Read the version `init` would resolve through opm itself. No opm command prints a module's version list, and adding one is a product change, outside a CI fix.

**Decision**: option 2. **Rationale**:
- It is exactly `init`'s float: `highestStable` over the versions of the baked major.
- Option 3 is stricter than any user can observe, since `init` never crosses the baked major, and a new major starts a new module path anyway.
- It needs one anonymous tags request per template, read the way cue's modregistry reads it (403 and 404 as none), so the script's list and the dry-run's agree.

The rule also covers a branch that fell behind `main`: if `main` published v1.0.5, a branch declaring an unpublished v1.0.4 fails with "not above the highest published v1.0.5" and the hint to update the branch.

## Risks / Trade-offs

- [Release-time failure] After D5, a template gate failure at release leaves the release a draft, never a published cli without its templates.
  - A transient failure (GHCR, network) is fixed by re-running the failed jobs.
  - A content failure burns the tag: bump the template on `main`, cut the next patch, and the owner removes the draft.
  - The release PR runs the same gates first, but that check is advisory, because no status check is required on `main`. **Recommendation (owner setting, not part of this change)**: make `Template Publish Gates (dry-run)` a required status check on `main`, so a red release PR cannot merge and no tag is burnt.
- [Direct pushes] A required check binds PRs only. A direct push that changes a template without a bump still lands. After this change, every following PR and every release fails until a bump lands, including releases cut on top of it (cases G, R5, R-PR, R6), and the bump then publishes the content. The stale content never ships silently.
- [GHCR dependency] The gate reads GHCR for every template on every run, anonymously: a token and a tags list per template, plus a manifest and a blob of a few KB for each published one. An outage fails the gate, as the dry-run's version listing already does. These reads go to `ghcr.io` only, while "already holds" comes from opm's mapping (`OPM_REGISTRY`, which the script defaults to GHCR). In CI the two agree, and D7 fails a template when they do not. A local run with another mapping can therefore fail; this does not apply to either workflow.
- [403 reads as none] Like cue's modregistry, the tags read takes a 403 or 404 as "no versions" (D7), because GHCR answers 403 for a repository it does not know. A 403 for a template that does exist (a private package, or a GHCR fault) fails closed when the dry-run says "already holds" (GHCR does not list the version). Under a GO verdict it skips only the "above the highest" check, and the push itself then still runs every publish gate.
- [New template visibility] A new template's GHCR package must be public after its first publish, as `opm module init` already requires. Until it is, the release run's authenticated `opm` sees the version and the anonymous tags read does not, so the next release fails (closed) for that template.
- [Toolchain skew] Both workflows install the cue that `go.mod` requires (D3), so in CI the tree zip and the push use one modzip. A local run with another cue can see a different selection; on the published side that can only show up as a difference.
- [Local files outside git] A local run zips whatever is on disk, as `opm module publish` would. An untracked or gitignored file under `templates/<t>/` (a `notes.log`, case IGN) is in the tree zip and not in the published one, so the content check fails; a `cue.mod/local-module.cue` fails the D6 listing. This is a false failure only locally: a CI checkout holds neither. `git clean -ndX templates/` lists the ignored files to remove before a local run.
- [Re-run of an old PR] A re-run compares the PR's tree with GHCR as it is now. It fails when GHCR now holds the PR's declared version with other content, or a higher version than the PR declares (D7), which is a real conflict, and the message says to update the branch first.
- [GNU tools] The script uses GNU `find -printf`, which `ubuntu-latest` has; a run on macOS needs findutils. The script is CI tooling, not part of the shipped cli.
- [Vet stops rendering] The gate's strength rests on `opm module vet` rendering. The `mod-vet` spec requires it; a change that drops the render would have to change that spec.
- [Friction] Every template edit needs a bump, comment-only ones included (R6), and `task deps:update` cli PRs need a manual `version set` until R5 lands. A bad direct push turns unrelated PRs red until it is fixed (case G). That is the point, but it is noisy.
- [Release latency] The binaries now wait one to two minutes for the template job (D5).

## Migration Plan

None. The current templates (all at v1.0.3, the highest published stable version of each) pass every new gate unchanged: no symlink, special file or nested module, each tree equals its zip, and each compares identical to its published artifact (case A).
