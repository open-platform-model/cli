# Tasks: gate-template-publish-on-vet

Revision 4, 2026-10-01, after review 3:

- Section 1 now also requires the tree to equal its module zip for every template (design.md D6), checks the module path against the directory, reads the identity inside `gate`, anchors the refusal match, and publishes on the gate's verdict instead of a second GHCR probe (D2). Both workflows install the cue that `go.mod` requires.
- Section 2 adds the version order of D7: stable only, GO above the highest published stable version, a published version equal to it.
- The harness removes `origin` from every scratch clone, installs cue the way the workflows do, and gains the `race`, `tags` and `tags-high` faults. Task 2.9 runs `/bin/bash` under `env -i`.
- New cases: S1, S2, M, V, the race, NM, HG and FIFO (section 1), and P, RB and LOW (section 2).

Delivery and the PR title are set in proposal.md. Workers never push, publish, tag or release. Every experiment that changes a template runs in a scratch clone, never in this worktree's `templates/`.

Environment for every `opm` and `cue` command, exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

Each section's gates are `task fmt`, `task lint`, `task test:unit` and `task openspec:check`. `task test:integration` and the full `task test:e2e` are left to CI and reported as skipped (section 4 runs its one e2e test directly).

## Isolation harness (every script run in this change)

No run of `publish-templates.sh` in this change may be able to write a registry or this worktree, whatever the gate under test does. Set it up once under a scratch directory `$S`, outside this worktree:

- `$S/home`, `$S/docker`, `$S/xdg`, `$S/cache`: empty directories.
- `$S/bin/opm-real`: `go build -o $S/bin/opm-real ./cmd/opm` from this worktree.
- `$S/bin/cue-real`: the cue the workflows install, from this worktree: `GOBIN=$S/bin go install "cuelang.org/go/cmd/cue@$(go list -m -f '{{.Version}}' cuelang.org/go)"`, then rename `$S/bin/cue` to `cue-real`.
- `$S/bin/opm-shim`, executable. It execs the real opm, except that a real publish is logged and refused, and `FAULT=race` makes that publish fail as opm does when another run pushed the version first:

  ```bash
  #!/usr/bin/env bash
  set -euo pipefail
  real=${OPM_REAL:?OPM_REAL must name the real opm binary}
  if [ "${1:-}" = module ] && [ "${2:-}" = publish ]; then
    for a in "$@"; do [ "$a" = --dry-run ] && exec "$real" "$@"; done
    if [ "${FAULT:-}" = race ]; then
      echo "RACE: refused: ${*: -1} already holds the version (simulated)" >&2
      exit 1
    fi
    echo "WOULD PUBLISH ${*: -1}"
    exit 0
  fi
  exec "$real" "$@"
  ```

- `$S/shim/cue`, executable. It execs `$CUE_REAL`, except that `cue mod publish` without `--out` prints `REFUSED CUE PUBLISH` and exits 97. With `FAULT=cue-out`, the `--out` call exits 1.
- `$S/shim/curl`, executable. It execs `$CURL_REAL` unless `FAULT` selects one failure in a GHCR read:
  - `token`: the anonymous token request fails (exit 22).
  - `tags`: the tags/list GET fails.
  - `tags-high`: the tags/list GET succeeds, then `v1.0.9` is added to its JSON (a higher version published after the branch was cut).
  - `manifest`: the manifest GET with `-o` fails.
  - `blob`: the blob GET fails.
  - `digest`: the blob GET succeeds, then one byte is appended to its output file.
- Every scratch clone is made with `git clone file://<this worktree> <dir>`, followed at once by `git -C <dir> remote remove origin` and a local user identity, so nothing in it can push back into this worktree.
- `$S/run.sh <dir> [--dry-run]` works in four steps:
  1. It asserts that `$S/docker` and `$S/xdg` hold nothing and `$S/home` holds nothing but `.opm` and `.cache` (exit 99 otherwise).
  2. It runs `dir=$1; shift; cd "$dir"`, so `"$@"` is only the mode flag.
  3. It runs:

     ```bash
     env -i PATH="$S/shim:/usr/bin:/bin" HOME="$S/home" DOCKER_CONFIG="$S/docker" \
       XDG_CONFIG_HOME="$S/xdg" CUE_CONFIG_DIR="$S/xdg/cue" CUE_CACHE_DIR="$S/cache" \
       CUE_REGISTRY="$CUE_REGISTRY" OPM_REGISTRY="$OPM_REGISTRY" FAULT="${FAULT:-}" \
       OPM_BIN="$S/bin/opm-shim" OPM_REAL="$S/bin/opm-real" \
       CUE_REAL="$S/bin/cue-real" CURL_REAL=/usr/bin/curl \
       /bin/bash "${SCRIPT:-.github/scripts/publish-templates.sh}" "$@"
     ```

  4. It prints `EXIT <status>`.

  `SCRIPT` defaults to the script in `<dir>`. The history replays in 2.6, and the cases whose branch is not based on this worktree's script, point it at a copy of this worktree's script.

A run passes isolation when:
- its log has no `WOULD PUBLISH` line except where a task expects one;
- a `--dry-run` log has no `==> <t>: publishing` line;
- no log has a `REFUSED CUE PUBLISH` line.

No local registry container is used.

A local run zips whatever is on disk. Before running on this worktree, `git clean -ndX templates/` must list nothing: an ignored file under a template (a `notes.log`) is a false content failure that a CI checkout cannot produce (design.md Risks).

## 1. Vet gate, tree equals zip, gate every template before publishing any (publish-templates.sh, workflows)

- [x] 1.1 `.github/scripts/publish-templates.sh`: restructure it into the two phases of design.md D2, with the D6 checks.
  - Preflight: fail with `==> missing tool: <name>` unless `cue` (checked first), `curl`, `jq`, `unzip`, `sha256sum`, `diff`, `find`, `sort`, `grep`, `sed`, `head`, `tail`, `cut`, `mktemp`, `cp`, `rm` and `basename` are on `PATH`. Export `LC_ALL=C`. A `work` temp directory is removed on `EXIT`.
  - A `gate <template> <dir>` function, called as an `if` condition, so every step checks its own status. In order:
    - Read `Version` and `ModulePath` with `cue eval ./identity --out text -e ...` inside `gate`; a failure prints the D4 line and fails only this template.
    - `ModulePath` must equal `opmodel.dev/templates/<dir>@v<major of Version>`.
    - `layout`: no symbolic link (`find -type l`), no special file (`find ! -type d ! -type f`), no nested `cue.mod` (`find -mindepth 2 -iname cue.mod`); each prints the D4 line and the paths.
    - `opm module tidy --check` and `opm module vet "./$dir"`.
    - `tree_zip`: `cue mod publish v<version> --out`, then `index.json`, its single manifest and its single `application/zip` layer (`zip_layer`).
    - `zip_holds_tree`: `find "$dir" ! -type d -printf '%P\n' | sort` equals `unzip -Z1 <tree zip> | sort`; a difference prints `only in the tree: <path>` lines, and a `diff` error fails too.
    - `opm module publish --dry-run "./$dir"`: GO sets the verdict `publish`. An output whose summary matches `grep -Eq ' 1 refusal$'` and whose headline holds `opmodel.dev/templates/<t>@v<major> already holds v<version>` sets the verdict `skip` (section 2 narrows this). Anything else fails.
  - The gate loop records each failing template and continues. After it, any failure prints `==> gates failed for: <list>; nothing published` and exits 1. `--dry-run` exits 0 after the gate loop.
  - The publish phase walks the recorded verdicts in template order: `skip` prints `already published; skipped`; `publish` prints `publishing v<version>` and runs `opm module publish "./templates/<t>/"`, and a failure prints the D4 line and exits 1. Remove `published()`: nothing probes GHCR a second time.
  - Keep the `CUE_REGISTRY`/`OPM_REGISTRY` export block and its comment.
  - Update the header comment: the gate list, the two phases, the verdict, and that vet renders `debugValues` against the template's own pins, and why the tree must equal its zip.

  Verify: `shellcheck .github/scripts/publish-templates.sh` is clean.
- [x] 1.2 Workflows.
  - Both cue install steps (`pr.yml` `template-gates`, `release.yml` `publish-templates`) become `go install "cuelang.org/go/cmd/cue@$(go list -m -f '{{.Version}}' cuelang.org/go)"`, named "Install cue (the version go.mod requires)". They already run after checkout and `setup-go`.
  - `release.yml`: the "Publish unpublished templates" step drops its `GHCR_AUTH` env (the script no longer reads it). `docker login` stays.
  - Comments only: `release.yml` `publish-templates` (lines 147-151) and `pr.yml` `template-gates` (lines 99-101) name the vet gate, the tree-equals-zip check and "every gate on every template before any publish".

  Verify: `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/pr.yml .github/workflows/release.yml` is clean, and in this worktree `go list -m -f '{{.Version}}' cuelang.org/go` prints the `go.mod` version.
- [x] 1.3 Build the harness, then run `$S/run.sh <this worktree> --dry-run`. Verify: exit 0. For each of minimal, standard and advanced, the output shows `Module valid (<n> resources)` from vet (1, 2 and 8 at v1.0.3) and an accepted "already holds" dry-run.
- [x] 1.4 Make the scratch clone `$S/clone` (harness rules: `remote remove origin`, a local identity) and check that it carries the real tags. Branch `case-B` from the clone's HEAD:
  - delete both `updateStrategy: type: "RollingUpdate"` lines from `templates/advanced/components.cue` (worker and cache);
  - run `$S/bin/opm-real module version set 1.0.4 templates/advanced`, then commit.

  Run `$S/run.sh $S/clone --dry-run` on it. Verify: exit 1. The output contains `unresolved disjunction "RollingUpdate" | "Recreate" | "OnDelete"`, the vet failure line for advanced, and `gates failed for: advanced; nothing published`. Minimal and standard still report their gates.
- [x] 1.5 Same branch, publish mode: `$S/run.sh $S/clone`. Verify: exit 1, the same failure line, no `publishing` and no `WOULD PUBLISH` line, and the isolation assertions held.
- [x] 1.6 Publish-mode success path. Branch `case-D` from the clone's HEAD:
  - change `opmodel.dev/catalogs/opm@v4` `v: "v4.4.4"` to `"v4.4.3"` in `templates/minimal/cue.mod/module.cue`;
  - run `$S/bin/opm-real module version set 1.0.4 templates/minimal`, then commit.

  Run `$S/run.sh $S/clone`. Verify: exit 0. The log holds exactly one `WOULD PUBLISH` line, `WOULD PUBLISH ./templates/minimal/`, preceded by `==> minimal: publishing v1.0.4`. Advanced and standard print "already published; skipped".
- [x] 1.7 What vet reads is what publishes (D6). Each case is a branch from the clone's HEAD, run in both modes unless stated. Verify exit 1, no `WOULD PUBLISH`, and the named line:
  - (S1) Add `hack/tplshared/extra.cue`, a second `package minimal` component, and commit `templates/minimal/extra.cue` as a symlink to it (`ln -s ../../hack/tplshared/extra.cue`), with no bump: `symbolic links never publish`, naming `extra.cue`. Vet alone passes it (2 resources); revision 3 called it identical to the published v1.0.3.
  - (S2) Move minimal's `debugValues` block out of `templates/minimal/module.cue` into `hack/tplshared/debug.cue`, commit `templates/minimal/debug.cue` as a symlink to it, and `version set 1.0.4`: the same line, naming `debug.cue`. Revision 3 said GO and published a zip without `debugValues`.
  - S1 and S2 again, dry-run, with `SCRIPT` pointing at a scratch copy of the script whose `layout` call is removed: exit 1 from the listing check alone, `only in the tree: extra.cue` and `only in the tree: debug.cue`.
  - (NM) `templates/minimal/sub/cue.mod/module.cue` (a nested module) and `sub/x.cue`, no bump, dry-run: `a nested cue.mod makes a nested module`, naming `sub/cue.mod`.
  - (HG) `templates/minimal/.hg_archival.txt`, no bump, dry-run: `the module zip does not hold exactly the tree's files`, `only in the tree: .hg_archival.txt`.
  - (FIFO) Uncommitted, on the clone's HEAD: `mkfifo templates/minimal/pipe`, dry-run: `special files never publish`, naming `pipe`. Remove it afterwards.
- [x] 1.8 Identity and verdict (D2). Verify:
  - (M) A branch renaming minimal's module path to `opmodel.dev/templates/minimalx@v1` in `cue.mod/module.cue`, the identity `ModulePath`, the package clause and the identity import, version kept, both modes: exit 1, `ModulePath is opmodel.dev/templates/minimalx@v1; the template in templates/minimal/ at v1.0.3 must be opmodel.dev/templates/minimal@v1`. Revision 3 exited 0 in publish mode, its second probe skipping the renamed template.
  - (V, identity part) A branch with a syntax error in `templates/minimal/identity/identity.cue` (`Version: 1.0.3 +`), dry-run: exit 1, cue's error, `cannot read Version and ModulePath from ./templates/minimal/identity`, and standard and advanced still run their gates. Revision 3 aborted at minimal.
  - (race) `FAULT=race` on `case-D`, publish mode: exit 1, `==> minimal: publishing v1.0.4`, then `publishing v1.0.4 failed`, and no `minimal: ... skipped` line.
  - `grep -n 'published()' .github/scripts/publish-templates.sh` returns nothing.
- [x] 1.9 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `ci(templates): gate template publishing on opm module vet`.

## 2. Changed implies bumped, version order (publish-templates.sh, pr.yml, release.yml comments, AGENTS.md)

- [x] 2.1 `publish-templates.sh`, per design.md D3, D4 and D7:
  - Add `token` (anonymous pull token, never `GHCR_AUTH`), `published_zip` (exactly one `application/zip` layer; digest format checked; blob fetched with `-L` and verified by `sha256sum`) and `same_as_published`, which unzips the published zip and the tree zip from 1.1 and runs `diff -ru published tree`: 0 passes, 1 fails with the diff (header timestamps stripped, at most 60 lines) and the bump message, and anything else fails.
  - Add `stable_versions <repo> <major>`: the anonymous token request (200 reads the token, 403 means no versions, anything else fails), then `GET /v2/<repo>/tags/list?n=1000`, following `Link: <...>; rel="next"` (at most 20 pages). 200 appends the tags, 403 or 404 on the first page means no versions, anything else fails. It prints the bare `<major>.X.Y` stable versions, sorted with `sort -t. -k1,1n -k2,2n -k3,3n`.
  - In `gate`, right after the identity read: `Version` must be strict stable `X.Y.Z`.
  - After the dry-run, with `H` the highest of `stable_versions`:
    - GO: `Version` is not among the versions, and is above `H` when there is one. Verdict `publish`.
    - "already holds": `Version` is among the versions and equals `H`, then `same_as_published`. Verdict `skip`.
  - Every step checks its own status and prints the D4 line for its failure.
  - Header comment: the content check, the version order, and why files and not zip bytes are compared.
  - Do not edit `hack/fixtures.sh`.

  Verify: `shellcheck` is clean.
- [x] 2.2 Comments only. `pr.yml` `template-gates` and `release.yml` `publish-templates` say that a template whose tree differs from its published artifact, or whose version is not the one `opm module init` resolves, fails, in the release job before any publish. Neither checkout changes. Verify: actionlint is clean.
- [x] 2.3 `AGENTS.md`, both lines together:
  - Line 128 (`templates/` entry): replace "Re-pinning a template needs ... (release CI skips a version GHCR already holds)." with: "Any change to a template's files, its `cue.mod` pins and comments included, needs `opm module version set` on that template, to a stable version above its highest published one, before the next release: published versions are immutable, and `.github/scripts/publish-templates.sh` fails the `template-gates` PR job and the release run (before any publish, so the release stays a draft) when a template's tree differs from the artifact GHCR holds at its declared version, holds a file its module zip omits (a symlink, for one), or declares a prerelease or an older version."
  - Line 351 (Template rule): replace it with: "Template rule: any change under `templates/<t>/` bumps that template with `opm module version set` in the same change, direct pushes included; see the `templates/` entry for where it is enforced."

  Verify: `grep -n "never publish" AGENTS.md` returns nothing.
- [x] 2.4 Harness, on this worktree, in both modes. Verify: exit 0, and each template prints `v1.0.3 already published and identical to the tree`. Publish mode prints three "already published; skipped" lines and no `WOULD PUBLISH`.
- [x] 2.5 In the scratch clone from 1.4, put each case on its own branch from the clone's HEAD and run `$S/run.sh $S/clone --dry-run` unless stated otherwise. Verify:
  - (C) Only 1.6's pin change, without the bump: exit 1. The diff shows `-v: "v4.4.4"` and `+v: "v4.4.3"` in `cue.mod/module.cue`, and the message names minimal and `opm module version set`. Publish mode: exit 1, no `WOULD PUBLISH`.
  - (D) 1.6's branch: exit 0, and minimal reports `GO` at v1.0.4 and `passes every gate and is not published yet`.
  - (E) A comment-only edit in `templates/standard/module.cue`: exit 1, and the diff shows the comment line.
  - (F) An unrelated change outside `templates/`: exit 0.
  - (G) C, then an unrelated commit on top: exit 1, minimal differs.
  - (H) C, then `git revert --no-edit HEAD`: exit 0.
  - (I) The current tree with `opm module version set 1.0.2 templates/advanced`, in both modes: exit 1, `v1.0.2 is published, but 'opm module init' fetches the highest published v1.0.3`.
  - (RB) Minimal's whole tree rolled back to its v1.0.2 content (`git rm -r templates/minimal`, `git checkout v1.0.0-alpha.22 -- templates/minimal`), in both modes: exit 1 with the same line for minimal. Its content equals the published v1.0.2, so revision 3 passed it.
  - (P) 1.6's pin change with `version set 1.0.4-rc.1`, in both modes: exit 1, `v1.0.4-rc.1 is not a stable version`, no `WOULD PUBLISH`. Revision 3 published it.
  - (LOW) `FAULT=tags-high` on 1.6's branch, in both modes: exit 1, `v1.0.4 is not above the highest published v1.0.9` for minimal, and `v1.0.3 is published, but 'opm module init' fetches the highest published v1.0.9` for advanced and standard.
  - (V) The branch from 1.8: exit 1, `gates failed for: minimal standard`, standard's diff showing the added comment.
  - (R) C tagged `v1.0.0-beta.5` in the clone, publish mode: exit 1. An unrelated commit on top, dry-run: exit 1. That commit tagged `v1.0.0-beta.6`, publish mode: exit 1. Minimal differs each time, and nothing is published.
- [x] 2.6 History replays, publish mode. Copy this worktree's script to `$S/publish-templates.sh`, run with `SCRIPT=$S/publish-templates.sh`, and check each tag out in the clone. Verify:
  - `v1.0.0-alpha.22`: exit 1; advanced fails vet, and minimal and standard fail with `v1.0.2 is published, but 'opm module init' fetches the highest published v1.0.3`.
  - `v1.0.0-alpha.25`, `v1.0.0-alpha.26` and `v1.0.0-alpha.27`: exit 1 with the same three failures.
  - `v1.0.0-beta.1`: exit 0, all three identical and skipped.
  - Positive control, with `SCRIPT` pointing at a scratch copy whose published-version order check is removed: `v1.0.0-alpha.22` prints `v1.0.2 already published and identical to the tree` for minimal and standard (the content check reproduces a publish made by an older opm), and RB's dry-run exits 0 (RB is caught by the order rule alone).
  - No log has a `WOULD PUBLISH` line.
- [x] 2.7 No history needed. Verify:
  - A `git archive HEAD` export of this worktree (no `.git`): exit 0 in both modes, all three identical.
  - A `git clone --depth 1 file://<this worktree>` (origin removed): `git rev-parse --is-shallow-repository` prints `true`, and the dry-run exits 0 with all three identical.
- [x] 2.8 New template: in a clone branch, copy `templates/minimal` to `templates/zzprobe` and rename its module path, identity `ModulePath`, package clause and identity import. Verify: the dry-run exits 0 with `GO` for zzprobe (GHCR answers its token request with 403, read as no versions). Publish mode exits 0 with exactly one `WOULD PUBLISH ./templates/zzprobe/`.
- [x] 2.9 Faults, on this worktree. Verify:
  - With `FAULT=token` and `FAULT=tags`, each dry-run exits 1 and names every template with `cannot list the published versions on GHCR; a fetch error never counts as unpublished`.
  - With `FAULT=manifest` and `blob`, each dry-run exits 1 and names every template with `cannot fetch the published v1.0.3 from GHCR; a fetch error never counts as unpublished`.
  - With `FAULT=digest`, the dry-run also prints `downloaded blob does not match sha256:...`.
  - With `FAULT=cue-out`, the dry-run prints `cannot build the tree's module zip` for every template.
  - With `FAULT=blob` in publish mode: exit 1, no `WOULD PUBLISH`.
  - `env -i PATH=/nonexistent /bin/bash .github/scripts/publish-templates.sh --dry-run` prints `==> missing tool: cue` and exits 1. (`env -i PATH=/nonexistent bash ...` cannot find `bash` itself and exits 127 without running the script.)
- [x] 2.10 After every run of this section, `$S/docker` and `$S/xdg` are empty, `$S/home` holds only `.opm` (and `.cache`), and no log has `REFUSED CUE PUBLISH`.
- [x] 2.11 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `ci(templates): fail a template whose tree differs from its published version`.

## 3. Publish templates before goreleaser builds the release (release.yml)

- [x] 3.1 `.github/workflows/release.yml`, per design.md D5:
  - `goreleaser` gets `needs: [release-please, publish-templates]`. Its `if:` and concurrency group stay unchanged.
  - Directly after "Check the release is still a draft", it gets a new step:

    ```yaml
          - name: Require the published templates
            env:
              RESULT: ${{ needs.publish-templates.result }}
              TAG: ${{ needs.release-please.outputs.tag_name || inputs.tag }}
            run: |
              if [ "$RESULT" != success ]; then
                echo "::error::publish-templates did not succeed (${RESULT}); release ${TAG} stays a draft."
                echo "publish-templates did not succeed (${RESULT}); release ${TAG} stays a draft and no binary is built."
                echo "A transient failure: re-run the failed jobs. A gate failure: bump the template on main and cut the next patch."
                exit 1
              fi
    ```

  - The `publish-templates` job is otherwise unchanged; it still has no draft check (`adopt-draft-first-release` design D3).
  - Header comment: the first paragraph says the template modules are gated and published before goreleaser fills the draft. The runbook gains: "Template job failed: goreleaser refuses to build, so the release stays a draft. A transient failure: re-run the failed jobs, or run this workflow by hand with the tag. A gate failure: the tag never moves; bump the template on main and cut the next patch; the owner removes the abandoned draft in the browser."
  - The `publish-templates` comment says that it runs first and that goreleaser waits for it.

  Verify: actionlint is clean.
- [x] 3.2 Verify the graph:
  - `yq '.jobs.goreleaser.needs' .github/workflows/release.yml` prints `release-please` and `publish-templates`.
  - `yq '.jobs.goreleaser.steps[].name'` lists the draft check, then "Require the published templates", then the checkout.
  - `yq '.jobs.publish-templates.needs'` prints only `release-please`.
  - Run the new step's `run:` body under `bash -e` with `RESULT` set to `success` (exit 0) and to `failure`, `cancelled` and `skipped` (exit 1 each, with the tag in the message).
- [x] 3.3 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `ci(release): publish templates before goreleaser builds the release`.

Archive-time note (not part of this PR): the `release-workflow` Purpose says goreleaser builds "while a second job publishes the bundled template modules". Deltas cannot change a Purpose, so reword it at archive to say that the template modules are gated and published before goreleaser publishes the release.

## 4. Scaffold test over every template (tests/e2e/mod_init_test.go)

- [ ] 4.1 `TestE2E_ModInit_ThenVet` becomes a loop of `t.Run(name, ...)` over `minimal`, `standard` and `advanced`, each calling the current body as `modInitThenVet(t, name)`. `repoTemplateDir(t, name)` and the init shortcut take `name`. The `components.cue` package-clause assertion runs only when the scaffold has that file (minimal has none). Update the doc comment: every official template, not only standard. Verify: `env -u OPM_CONFIG HOME=$S/e2ehome DOCKER_CONFIG=$S/docker go test -count=1 ./tests/e2e -run 'TestE2E_ModInit_ThenVet$' -v` passes with three subtests. No cluster is needed; the test runs an in-process registry.
- [ ] 4.2 Scratch only (the clone from 1.4, never this worktree): apply case B's template change and run `go test ./tests/e2e -run 'TestE2E_ModInit_ThenVet$/advanced'`. Verify: the advanced subtest fails with `vet failed` and the unresolved disjunction.
- [ ] 4.3 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `test(e2e): scaffold and vet every official template`.
