# Tasks: gate-template-publish-on-vet

Revision 3, 2026-10-01:

- Section 2 compares each published template's GHCR content with the tree, not a git base, and every fetch or comparison error fails.
- New section 3 runs `publish-templates` before goreleaser builds.
- The e2e test moves to section 4.
- The harness strips `<dir>` before forwarding `"$@"`, and puts `cue` (and, for fault cases, `curl`) behind shims.

Delivery and the PR title are set in proposal.md. Workers never push, publish, tag or release. Every experiment that changes a template runs in a scratch clone, never in this worktree's `templates/`.

Environment for every `opm` and `cue` command, exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

Each section's gates are `task fmt`, `task lint`, `task test:unit` and `task openspec:check`. `task test:integration` and the full `task test:e2e` are left to CI and reported as skipped (section 4 runs its one e2e test directly).

## Isolation harness (every script run in this change)

No run of `publish-templates.sh` in this change may be able to write a registry, whatever the gate under test does. Set it up once under a scratch directory `$S`, outside this worktree:

- `$S/home`, `$S/docker`, `$S/xdg`, `$S/cache`: empty directories.
- `$S/bin/opm-real`: `go build -o $S/bin/opm-real ./cmd/opm` from this worktree.
- `$S/bin/opm-shim`, executable. It execs the real opm, except that a real publish is logged and refused:

  ```bash
  #!/usr/bin/env bash
  set -euo pipefail
  real=${OPM_REAL:?OPM_REAL must name the real opm binary}
  if [ "${1:-}" = module ] && [ "${2:-}" = publish ]; then
    for a in "$@"; do [ "$a" = --dry-run ] && exec "$real" "$@"; done
    echo "WOULD PUBLISH ${*: -1}"
    exit 0
  fi
  exec "$real" "$@"
  ```

- `$S/shim/cue`, executable. It execs `$CUE_REAL`, except that `cue mod publish` without `--out` prints `REFUSED CUE PUBLISH` and exits 97. With `FAULT=cue-out`, the `--out` call exits 1.
- `$S/shim/curl`, executable. It execs `$CURL_REAL` unless `FAULT` selects one failure in the content fetch:
  - `token`: the anonymous token request fails (exit 22).
  - `manifest`: the manifest GET with `-o` fails.
  - `blob`: the blob GET fails.
  - `digest`: the blob GET succeeds, then one byte is appended to its output file.
- `$S/run.sh <dir> [--dry-run]` works in four steps:
  1. It asserts that `$S/docker` and `$S/xdg` hold nothing and `$S/home` holds nothing but `.opm` and `.cache` (exit 99 otherwise).
  2. It runs `dir=$1; shift; cd "$dir"`, so `"$@"` is only the mode flag.
  3. It runs:

     ```bash
     env -i PATH="$S/shim:$PATH" HOME="$S/home" DOCKER_CONFIG="$S/docker" \
       XDG_CONFIG_HOME="$S/xdg" CUE_CONFIG_DIR="$S/xdg/cue" CUE_CACHE_DIR="$S/cache" \
       CUE_REGISTRY="$CUE_REGISTRY" OPM_REGISTRY="$OPM_REGISTRY" FAULT="${FAULT:-}" \
       OPM_BIN="$S/bin/opm-shim" OPM_REAL="$S/bin/opm-real" \
       CUE_REAL="$(command -v cue)" CURL_REAL="$(command -v curl)" \
       bash "${SCRIPT:-.github/scripts/publish-templates.sh}" "$@"
     ```

  4. It prints `EXIT <status>`.

  `SCRIPT` defaults to the script in `<dir>`. The history replays in 2.6 point it at a copy of this worktree's script.

A run passes isolation when:
- its log has no `WOULD PUBLISH` line except where a task expects one;
- a `--dry-run` log has no `==> <t>: publishing` line;
- no log has a `REFUSED CUE PUBLISH` line.

No local registry container is used.

## 1. Vet gate, gate every template before publishing any (publish-templates.sh)

- [ ] 1.1 `.github/scripts/publish-templates.sh`: restructure it into the two phases of design.md D2.
  - A `gate <template> <dir> <version>` function runs, in order, `opm module tidy --check`, `opm module vet "./$dir"` and `opm module publish --dry-run "./$dir"`. GO passes, an output whose only refusal is "already holds" passes (section 2 narrows this), and anything else fails.
  - `gate` runs in an `||` list, so every step checks its own status.
  - The gate loop records each failing template and continues. After it, any failure prints `==> gates failed for: <list>; nothing published` and exits 1. `--dry-run` exits 0 after the gate loop.
  - Publish mode then runs the existing loop (the GHCR `published` filter, then `opm module publish`), with the tidy call removed from it.
  - Keep the `CUE_REGISTRY`/`OPM_REGISTRY` export block and the `published` function unchanged.
  - Update the header comment: the gate list, the two phases, and that vet renders `debugValues` against the template's own pins.

  Verify: `shellcheck .github/scripts/publish-templates.sh` is clean.
- [ ] 1.2 Comments only: `release.yml` `publish-templates` (lines 147-151) and `pr.yml` `template-gates` (lines 99-101) name the vet gate and "every gate on every template before any publish". Verify: `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/pr.yml .github/workflows/release.yml` is clean.
- [ ] 1.3 Build the harness, then run `$S/run.sh <this worktree> --dry-run`. Verify: exit 0. For each of minimal, standard and advanced, the output shows `Module valid (<n> resources)` from vet (1, 2 and 8 at v1.0.3) and an accepted "already holds" dry-run.
- [ ] 1.4 Make a scratch clone with `git clone file://<this worktree> $S/clone`, set a user identity locally in the clone, and check that it carries the real tags. Branch `case-B` from the clone's HEAD:
  - delete both `updateStrategy: type: "RollingUpdate"` lines from `templates/advanced/components.cue` (worker and cache);
  - run `$S/bin/opm-real module version set 1.0.4 templates/advanced`, then commit.

  Run `$S/run.sh $S/clone --dry-run` on it. Verify: exit 1. The output contains `unresolved disjunction "RollingUpdate" | "Recreate" | "OnDelete"`, the vet failure line for advanced, and `gates failed for: advanced; nothing published`. Minimal and standard still report their gates.
- [ ] 1.5 Same branch, publish mode: `$S/run.sh $S/clone`. Verify: exit 1, the same failure line, no `publishing` and no `WOULD PUBLISH` line, and the isolation assertions held.
- [ ] 1.6 Publish-mode success path. Branch `case-D` from the clone's HEAD:
  - change `opmodel.dev/catalogs/opm@v4` `v: "v4.4.4"` to `"v4.4.3"` in `templates/minimal/cue.mod/module.cue`;
  - run `$S/bin/opm-real module version set 1.0.4 templates/minimal`, then commit.

  Run `$S/run.sh $S/clone`. Verify: exit 0. The log holds exactly one `WOULD PUBLISH` line, `WOULD PUBLISH ./templates/minimal/`, preceded by `==> minimal: publishing v1.0.4`. Advanced and standard print "already published; skipped".
- [ ] 1.7 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `ci(templates): gate template publishing on opm module vet`.

## 2. Changed implies bumped, by published content (publish-templates.sh, pr.yml, release.yml comments, AGENTS.md)

- [ ] 2.1 `publish-templates.sh`, per design.md D3 and D4:
  - Before anything else, a preflight fails with `==> missing tool: <name>` unless `cue`, `curl`, `jq`, `unzip`, `sha256sum` and `diff` are on `PATH`.
  - A `work` temp directory is removed on `EXIT`.
  - Add `zip_layer`, `published_zip` (anonymous pull token, never `GHCR_AUTH`; exactly one `application/zip` layer; digest format checked; blob fetched with `-L` and verified by `sha256sum`), `tree_zip` (`cue mod publish v<version> --out`, then `index.json`, its single manifest and its zip layer) and `same_as_published`.
  - `same_as_published` unzips both zips and runs `diff -ru published tree`: 0 passes, 1 fails with the diff (header timestamps stripped, at most 60 lines) and the bump message, and anything else fails.
  - In `gate`, when the only refusal is "already holds", the result is `same_as_published`.
  - Every step checks its own status and prints the D4 line for its failure.
  - Header comment: the content check, why the tree zip comes from `cue mod publish --out` (the publisher's `modzip.CreateFromDir` for `source: kind: "self"`, which the gates require), and why files and not zip bytes are compared.
  - Do not edit `hack/fixtures.sh`.

  Verify: `shellcheck` is clean.
- [ ] 2.2 Comments only. `pr.yml` `template-gates` and `release.yml` `publish-templates` say that a template whose tree differs from its published artifact fails, in the release job before any publish. Neither checkout changes. Verify: actionlint is clean.
- [ ] 2.3 `AGENTS.md`, both lines together:
  - Line 128 (`templates/` entry): replace "Re-pinning a template needs ... (release CI skips a version GHCR already holds)." with: "Any change to a template's files, its `cue.mod` pins and comments included, needs `opm module version set` on that template before the next release: published versions are immutable, and `.github/scripts/publish-templates.sh` fails the `template-gates` PR job and the release run (before any publish, so the release stays a draft) when a template's tree differs from the artifact GHCR holds at its declared version."
  - Line 351 (Template rule): replace it with: "Template rule: any change under `templates/<t>/` bumps that template with `opm module version set` in the same change, direct pushes included; see the `templates/` entry for where it is enforced."

  Verify: `grep -n "never publish" AGENTS.md` returns nothing.
- [ ] 2.4 Harness, on this worktree, in both modes. Verify: exit 0, and each template prints `v1.0.3 already published and identical to the tree`. Publish mode prints three "already published; skipped" lines and no `WOULD PUBLISH`.
- [ ] 2.5 In the scratch clone from 1.4, put each case on its own branch from the clone's HEAD and run `$S/run.sh $S/clone --dry-run` unless stated otherwise. Verify:
  - (C) Only 1.6's pin change, without the bump: exit 1. The diff shows `-v: "v4.4.4"` and `+v: "v4.4.3"` in `cue.mod/module.cue`, and the message names minimal and `opm module version set`. Publish mode: exit 1, no `WOULD PUBLISH`.
  - (D) 1.6's branch: exit 0, and minimal reports `GO` at v1.0.4.
  - (E) A comment-only edit in `templates/standard/module.cue`: exit 1, and the diff shows the comment line.
  - (F) An unrelated change outside `templates/`: exit 0.
  - (G) C, then an unrelated commit on top: exit 1, minimal differs.
  - (H) C, then `git revert --no-edit HEAD`: exit 0.
  - (I) The current tree with `opm module version set 1.0.2 templates/advanced`, in both modes: exit 1. The diff against the published v1.0.2 names both `updateStrategy` lines in `components.cue` and the pins in `cue.mod/module.cue`.
  - (R) C tagged `v1.0.0-beta.5` in the clone, publish mode: exit 1. An unrelated commit on top, dry-run: exit 1. That commit tagged `v1.0.0-beta.6`, publish mode: exit 1. Minimal differs each time, and nothing is published.
- [ ] 2.6 History replays, publish mode. Copy this worktree's script to `$S/publish-templates.sh`, run with `SCRIPT=$S/publish-templates.sh`, and check each tag out in the clone. Verify:
  - `v1.0.0-alpha.22`: minimal and standard print `v1.0.2 already published and identical to the tree`, and advanced fails vet.
  - `v1.0.0-alpha.26` and `v1.0.0-alpha.27`: exit 1, minimal and standard differ from v1.0.2, and advanced fails vet.
  - `v1.0.0-beta.1`: exit 0, all three identical and skipped.
  - No log has a `WOULD PUBLISH` line.
- [ ] 2.7 No history needed. Verify:
  - A `git archive HEAD` export of this worktree (no `.git`): exit 0 in both modes, all three identical.
  - `git clone --depth 1 file://<this worktree>`: `git rev-parse --is-shallow-repository` prints `true`, and the dry-run exits 0 with all three identical.
- [ ] 2.8 New template: in a clone branch, copy `templates/minimal` to `templates/zzprobe` and rename its module path, identity `ModulePath`, package clause and identity import. Verify: the dry-run exits 0 with `GO` for zzprobe. Publish mode exits 0 with exactly one `WOULD PUBLISH ./templates/zzprobe/`.
- [ ] 2.9 Faults, on this worktree. Verify:
  - With `FAULT=token`, `manifest` and `blob`, each dry-run exits 1 and names every template with `cannot fetch the published v1.0.3 from GHCR; a fetch error never counts as unpublished`.
  - With `FAULT=digest`, the dry-run also prints `downloaded blob does not match sha256:...`.
  - With `FAULT=cue-out`, the dry-run prints `cannot build the tree's module zip`.
  - With `FAULT=blob` in publish mode: exit 1, no `WOULD PUBLISH`.
  - `env -i PATH=/nonexistent bash .github/scripts/publish-templates.sh --dry-run` prints `==> missing tool: cue` and exits 1.
- [ ] 2.10 After every run of this section, `$S/docker` and `$S/xdg` are empty, `$S/home` holds only `.opm` (and `.cache`), and no log has `REFUSED CUE PUBLISH`.
- [ ] 2.11 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `ci(templates): fail a template whose tree differs from its published version`.

## 3. Publish templates before goreleaser builds the release (release.yml)

- [ ] 3.1 `.github/workflows/release.yml`, per design.md D5:
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
- [ ] 3.2 Verify the graph:
  - `yq '.jobs.goreleaser.needs' .github/workflows/release.yml` prints `release-please` and `publish-templates`.
  - `yq '.jobs.goreleaser.steps[].name'` lists the draft check, then "Require the published templates", then the checkout.
  - `yq '.jobs.publish-templates.needs'` prints only `release-please`.
  - Run the new step's `run:` body under `bash -e` with `RESULT` set to `success` (exit 0) and to `failure`, `cancelled` and `skipped` (exit 1 each, with the tag in the message).
- [ ] 3.3 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `ci(release): publish templates before goreleaser builds the release`.

Archive-time note (not part of this PR): the `release-workflow` Purpose says goreleaser builds "while a second job publishes the bundled template modules". Deltas cannot change a Purpose, so reword it at archive to say that the template modules are gated and published before goreleaser publishes the release.

## 4. Scaffold test over every template (tests/e2e/mod_init_test.go)

- [ ] 4.1 `TestE2E_ModInit_ThenVet` becomes a loop of `t.Run(name, ...)` over `minimal`, `standard` and `advanced`, each calling the current body as `modInitThenVet(t, name)`. `repoTemplateDir(t, name)` and the init shortcut take `name`. The `components.cue` package-clause assertion runs only when the scaffold has that file (minimal has none). Update the doc comment: every official template, not only standard. Verify: `env -u OPM_CONFIG HOME=$S/e2ehome DOCKER_CONFIG=$S/docker go test -count=1 ./tests/e2e -run 'TestE2E_ModInit_ThenVet$' -v` passes with three subtests. No cluster is needed; the test runs an in-process registry.
- [ ] 4.2 Scratch only (the clone from 1.4, never this worktree): apply case B's template change and run `go test ./tests/e2e -run 'TestE2E_ModInit_ThenVet$/advanced'`. Verify: the advanced subtest fails with `vet failed` and the unresolved disjunction.
- [ ] 4.3 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` are green. Then commit `test(e2e): scaffold and vet every official template`.
