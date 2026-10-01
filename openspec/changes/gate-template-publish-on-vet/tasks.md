# Tasks: gate-template-publish-on-vet

Revision 2026-10-01: section 2 compares against the previous release tag in both modes (was: merge-base, PR only); every run that is not `--dry-run` goes through the isolation harness below; a publish-mode success case (D) and the release-mode cases J to L were added; section 3 extends the e2e scaffold test.

Delivery and the PR title are set in proposal.md. Workers never push, publish, tag or release. Every experiment that changes a template runs in a scratch clone, never in this worktree's `templates/`.

Environment for every `opm` and `cue` command, exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

Each section's gates are `task fmt`, `task lint`, `task test:unit` and `task openspec:check`; `task test:integration` and the full `task test:e2e` are left to CI and reported as skipped (section 3 runs its one e2e test directly).

## Isolation harness (every script run in this change)

No run of `publish-templates.sh` in this change may be able to write a registry, whatever the gate under test does. Set up once under a scratch directory `$S` (not in this worktree):

- `$S/home`, `$S/docker`, `$S/xdg`: empty directories.
- `$S/bin/opm-real`: `go build -o $S/bin/opm-real ./cmd/opm` from this worktree.
- `$S/bin/opm-shim`, executable:

  ```bash
  #!/usr/bin/env bash
  # Exec the real opm for everything except a real publish, which is logged and refused.
  set -euo pipefail
  real=${OPM_REAL:?OPM_REAL must name the real opm binary}
  if [ "${1:-}" = module ] && [ "${2:-}" = publish ]; then
    for a in "$@"; do [ "$a" = --dry-run ] && exec "$real" "$@"; done
    echo "WOULD PUBLISH ${*: -1}"
    exit 0
  fi
  exec "$real" "$@"
  ```

- `$S/run.sh <dir> [--dry-run]`: asserts `$S/docker/config.json` does not exist and `$S/xdg` is empty (exit 99 otherwise), then runs, from `<dir>`:

  ```bash
  env -i PATH="$PATH" HOME="$S/home" DOCKER_CONFIG="$S/docker" \
    XDG_CONFIG_HOME="$S/xdg" CUE_CONFIG_DIR="$S/xdg/cue" \
    CUE_REGISTRY="$CUE_REGISTRY" OPM_REGISTRY="$OPM_REGISTRY" \
    OPM_BIN="$S/bin/opm-shim" OPM_REAL="$S/bin/opm-real" \
    bash .github/scripts/publish-templates.sh "$@"
  ```

A run "passes isolation" when its log has no `WOULD PUBLISH` line except where a task expects one. No local registry container is used.

## 1. Vet gate, gate every template before publishing any (publish-templates.sh)

- [ ] 1.1 `.github/scripts/publish-templates.sh`: restructure into the two phases of design.md D2. A `gate <template> <dir> <version>` function runs, in order, `opm module tidy --check`, `opm module vet "./$dir"` and `opm module publish --dry-run "./$dir"`; GO passes, an output whose only refusal is "already holds" passes (section 2 narrows this), anything else fails. The gate loop records each failing template and continues; after it, any failure prints `==> gates failed for: <list>; nothing published` and exits 1. `--dry-run` exits 0 after the gate loop. Publish mode then runs the existing loop (GHCR `published` filter, then `opm module publish`) with the tidy call removed from it. Keep the `CUE_REGISTRY`/`OPM_REGISTRY` export block and the `published` function unchanged. Update the header comment: the gate list, the two phases, and that vet renders `debugValues` against the template's own pins. Verify: `shellcheck .github/scripts/publish-templates.sh` is clean.
- [ ] 1.2 Comments only: `release.yml` `publish-templates` (lines 147-151) and `pr.yml` `template-gates` (lines 99-101) name the vet gate and "every gate on every template before any publish". Verify: `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/pr.yml .github/workflows/release.yml` is clean.
- [ ] 1.3 Build the harness, then `$S/run.sh <this worktree> --dry-run`. Verify: exit 0; for each of minimal, standard, advanced the output shows `Module valid (<n> resources)` from vet (1, 2 and 8 at v1.0.3) and an "already holds" dry-run accepted.
- [ ] 1.4 Scratch clone (`git clone <this worktree> $S/clone`, user identity set locally in the clone; the clone carries the real tags). Branch `case-B` from the clone's HEAD: delete the two `updateStrategy: type: "RollingUpdate"` lines from `templates/advanced/components.cue`, `$S/bin/opm-real module version set 1.0.4 templates/advanced`, commit. `$S/run.sh $S/clone --dry-run` on it. Verify: exit 1, output contains `unresolved disjunction "RollingUpdate" | "Recreate" | "OnDelete"`, the vet failure line for advanced, and `gates failed for: advanced; nothing published`; minimal and standard still report their gates.
- [ ] 1.5 Same branch, `$S/run.sh $S/clone` (publish mode). Verify: exit 1, the same failure line, no `publishing` and no `WOULD PUBLISH` line, and the isolation assertions held.
- [ ] 1.6 Publish-mode success path. Branch `case-P` from the clone's HEAD: change `opmodel.dev/catalogs/opm@v4` `v: "v4.4.4"` to `"v4.4.3"` in `templates/minimal/cue.mod/module.cue`, `$S/bin/opm-real module version set 1.0.4 templates/minimal`, commit. `$S/run.sh $S/clone` (publish mode). Verify: exit 0; the log holds exactly one `WOULD PUBLISH` line, `WOULD PUBLISH ./templates/minimal/`, preceded by `==> minimal: publishing v1.0.4`; advanced and standard print "already published; skipped".
- [ ] 1.7 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `ci(templates): gate template publishing on opm module vet`.

## 2. Changed implies bumped against the previous release tag (publish-templates.sh, pr.yml, release.yml, AGENTS.md)

- [ ] 2.1 `publish-templates.sh`: before the gate loop, set `prev=$(git describe --tags --abbrev=0 --match 'v[0-9]*' HEAD^)`; when it fails, print `==> no release tag reachable from HEAD^; check out full history with tags (fetch-depth: 0)` and exit 1; otherwise print `==> previous release tag: <prev>`. Add `version_at <ref> <dir>` (`git show "<ref>:<dir>identity/identity.cue"` into a temp file, `cue eval <file> --out text -e Version`, empty when absent) and `raised <old> <new>` (0 when `<old>` is empty, or differs from `<new>` and sorts below it under `sort -V`). In `gate`, when the only refusal is "already holds": pass when `git diff --quiet "$prev" HEAD -- "$dir"`; else, in publish mode only, pass when `raised "$(version_at "$prev" "$dir")" "$version"` (message: already published by an earlier run of this release); else fail with the design.md D3 message. Do not edit `hack/fixtures.sh` (it must stay byte-identical with opm-operator's copy). Header comment: why the base is the previous release tag (direct pushes) and why `HEAD^`. Verify: `shellcheck` clean.
- [ ] 2.2 `pr.yml` `template-gates` and `release.yml` `publish-templates`: the checkout gains `fetch-depth: 0` with a comment (the script diffs each template against the previous release tag); no `BASE_REF`. Update both jobs' comments: changed-implies-bumped against the previous release tag, and in the release job, that it fails before any publish. Verify: actionlint clean.
- [ ] 2.3 `AGENTS.md`, both lines together. Line 128 (`templates/` entry): replace "Re-pinning a template needs ... (release CI skips a version GHCR already holds)." with: "Any change to a template's files, its `cue.mod` pins and comments included, needs `opm module version set` on that template before the next release: published versions are immutable, and `.github/scripts/publish-templates.sh` fails the `template-gates` PR job and the release run (before any publish) when a template changed since the previous release tag sits at a version GHCR already holds." Line 351 (Template rule): replace with "Template rule: any change under `templates/<t>/` bumps that template with `opm module version set` in the same change, direct pushes included; see the `templates/` entry for where it is enforced." Verify: `grep -n "never publish" AGENTS.md` returns nothing.
- [ ] 2.4 Harness, in this worktree: `$S/run.sh <this worktree> --dry-run`. Verify: exit 0, `previous release tag: v1.0.0-beta.4` (or the newest tag at implementation time), each template "already published and unchanged since <tag>". Then in a `git clone --depth 1` of this worktree: exit 1 with the fetch-depth hint.
- [ ] 2.5 Scratch clone from 1.4, each case on its own branch from the clone's HEAD, `$S/run.sh $S/clone --dry-run` unless stated. Verify:
  - (C) the minimal pin change of 1.6 without the bump: exit 1, the D3 message names minimal and `opm module version set`. Publish mode too: exit 1, no `WOULD PUBLISH`.
  - (D) 1.6's branch: exit 0, minimal reports `GO` at v1.0.4.
  - (E) a comment-only edit in `templates/standard/module.cue`: exit 1, the D3 message names standard.
  - (F) an unrelated change outside `templates/`: exit 0.
  - (G) C, then an unrelated commit on top: exit 1, the D3 message names minimal (a direct push followed by an innocent PR).
  - (H) C, then `git revert HEAD`: exit 0.
  - (I) C plus `version set 1.0.2` (older, already published): exit 1 in both modes, the D3 message names minimal.
  - (J) a branch at tag `v1.0.0-beta.1`, publish mode (re-run of the release that raised every template to 1.0.3): exit 0, each template "already published by an earlier run of this release", no `WOULD PUBLISH`.
  - (K) a branch at tag `v1.0.0-alpha.27`, publish mode: exit 1, minimal and standard named by D3 (and advanced by vet), no `WOULD PUBLISH`.
- [ ] 2.6 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `ci(templates): fail a template changed since the last release at a published version`.

## 3. Scaffold test over every template (tests/e2e/mod_init_test.go)

- [ ] 3.1 `TestE2E_ModInit_ThenVet` becomes a loop of `t.Run(name, ...)` over `minimal`, `standard` and `advanced`, each calling the current body as `modInitThenVet(t, name)`; `repoTemplateDir(t, name)` and the init shortcut take `name`. The `components.cue` package-clause assertion runs only when the scaffold has that file (minimal has none). Update the doc comment: every official template, not only standard. Verify: `env -u OPM_CONFIG HOME=$S/e2ehome DOCKER_CONFIG=$S/docker go test -count=1 ./tests/e2e -run 'TestE2E_ModInit_ThenVet$' -v` passes with three subtests (no cluster needed; the test runs an in-process registry).
- [ ] 3.2 Scratch only (clone from 1.4, never this worktree): apply case B's template change and run `go test ./tests/e2e -run 'TestE2E_ModInit_ThenVet$/advanced'`. Verify: the advanced subtest fails with `vet failed` and the unresolved disjunction.
- [ ] 3.3 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `test(e2e): scaffold and vet every official template`.
