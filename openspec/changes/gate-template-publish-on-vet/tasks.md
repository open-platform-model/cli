# Tasks: gate-template-publish-on-vet

Delivery and the PR title are set in proposal.md. Workers never push, publish, tag or release. Every experiment that changes a template runs in a scratch clone, never in this worktree's `templates/`.

Environment for every `opm` and `cue` command, exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

No Go file changes in this change. Each section's gates are `task fmt`, `task lint`, `task test:unit` and `task openspec:check`; `task test:integration` and `task test:e2e` are left to CI and reported as skipped.

Never run the script without `--dry-run` on a tree whose gates pass and whose template versions GHCR lacks: publish mode pushes to GHCR when the caller has credentials. Publish mode is exercised only on a tree whose gates fail (task 1.5).

## 1. Vet gate, gate every template before publishing any (publish-templates.sh)

- [ ] 1.1 `.github/scripts/publish-templates.sh`: restructure into the two phases of design.md D2. A `gate <template> <dir> <tag>` function runs, in order, `opm module tidy --check`, `opm module vet "./$dir"` and `opm module publish --dry-run "./$dir"`; GO passes, an output whose only refusal is "already holds" passes, anything else fails. The gate loop records each failing template and continues; after it, any failure prints `==> gates failed for: <list>; nothing published` and exits 1. `--dry-run` exits 0 after the gate loop. Publish mode then runs the existing loop (GHCR `published` filter, then `opm module publish`) with the tidy call removed from it. Keep the `CUE_REGISTRY`/`OPM_REGISTRY` export block and the `published` function unchanged. Update the header comment: the gate list, the two phases, and that vet renders `debugValues` against the template's own pins. Verify: `shellcheck .github/scripts/publish-templates.sh` is clean.
- [ ] 1.2 Comments only: `release.yml` `publish-templates` (lines 147-151) and `pr.yml` `template-gates` (lines 99-101) name the vet gate and "every gate on every template before any publish". No other workflow change in this section. Verify: `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/pr.yml .github/workflows/release.yml` is clean.
- [ ] 1.3 `task build`, then `OPM_BIN=./bin/opm .github/scripts/publish-templates.sh --dry-run` in this worktree. Verify: exit 0; for each of minimal, standard, advanced the output shows `Module valid (<n> resources)` from vet (1, 2 and 8 at v1.0.3) and an "already holds" dry-run accepted.
- [ ] 1.4 Scratch clone (`git clone <this worktree> <scratch>/clone`, user identity set locally in the clone): delete the two `updateStrategy: type: "RollingUpdate"` lines from `templates/advanced/components.cue`, `./bin/opm module version set 1.0.4 templates/advanced`, commit. Run the script with `--dry-run` there (`OPM_BIN` absolute). Verify: exit 1, output contains `unresolved disjunction "RollingUpdate" | "Recreate" | "OnDelete"`, the vet failure line for advanced, and `gates failed for: advanced; nothing published`; minimal and standard still report their gates.
- [ ] 1.5 Same scratch tree, publish mode (no flag). Verify: exit 1 with the same failure line and no `publishing` line, so nothing reached the push.
- [ ] 1.6 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `ci(templates): gate template publishing on opm module vet`.

## 2. Changed implies bumped (publish-templates.sh, pr.yml, AGENTS.md)

- [ ] 2.1 `publish-templates.sh`: add `BASE_REF=${BASE_REF:-}` and a `changed_since <ref> <dir>` function with the body of `hack/fixtures.sh` lines 118-123 (merge-base of `<ref>` and `HEAD`, `git diff --quiet` on the directory), failing the run with a fetch hint when `<ref>` does not resolve. In `gate`, when the only refusal is "already holds" and `BASE_REF` is set and the template changed since it, fail with the design.md D3 message. Do not edit `hack/fixtures.sh` (it must stay byte-identical with opm-operator's copy). Verify: `shellcheck` clean.
- [ ] 2.2 `pr.yml` `template-gates`: the checkout gains `with: fetch-depth: 0` (comment: the script diffs each template against the base branch), the script step gains `env: BASE_REF: origin/${{ github.base_ref || 'main' }}`, matching the `fixtures` job. Verify: actionlint clean.
- [ ] 2.3 `AGENTS.md` line 128 (`templates/`): replace "or the new pins never publish (release CI skips a version GHCR already holds)" with a statement that any change to a template needs `opm module version set` on it in the same PR, and the `template-gates` PR job fails otherwise. No other AGENTS.md edit.
- [ ] 2.4 In this worktree: `OPM_BIN=./bin/opm BASE_REF=origin/main .github/scripts/publish-templates.sh --dry-run`. Verify: exit 0, each template reports already published and unchanged. Then with `BASE_REF=does-not-exist`: exit non-zero with the fetch hint.
- [ ] 2.5 Fresh scratch clone, `git tag base` at its HEAD, each case on its own branch from `base`, run with `BASE_REF=base --dry-run`. Verify:
  - (C) change `opmodel.dev/catalogs/opm@v4` `v: "v4.4.4"` to `"v4.4.3"` in `templates/minimal/cue.mod/module.cue`, commit: exit 1, the D3 message names minimal and `opm module version set`.
  - (D) C plus `./bin/opm module version set 1.0.4 templates/minimal`, commit: exit 0, minimal reports `GO` at v1.0.4.
  - (E) a comment-only edit in `templates/standard/module.cue`, commit: exit 1, the D3 message names standard.
  - (F) on `base`, commit an unrelated file change outside `templates/`: exit 0.
- [ ] 2.6 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `ci(templates): fail a changed template at an already-published version`.
