# Tasks: adopt-draft-first-release

One PR for sections 1 to 4 (branch `tags/draft-first-release`); section 5 runs after the first real release and lands as its own small PR. Every commit uses a hidden type (`ci`, `chore`), so merging cuts no release by itself. Workers never merge, never tag, never create, edit or delete a GitHub Release, and never touch rulesets or org settings. Items marked **SUPERVISOR** or **OWNER** are not worker tasks.

Environment for every Go/CUE command, exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`task test` includes `task test:integration`, which needs a live kind cluster; when none is available, run `task test:unit` and `task test:e2e` (with `env -u OPM_CONFIG`) and report integration as skipped instead of starting a cluster unasked.

## Gates (SUPERVISOR ticks)

- [ ] G-sandbox `open-platform-model/release-flow-sandbox` proved the draft-first flow end to end with this change's exact keys (release-please `draft` + `force-tag-creation`, goreleaser `use_existing_draft` + `mode: keep-existing` + `replace_existing_artifacts`, the verify step), covering design.md U1 to U5, under the `tags-immutable` ruleset and with immutable releases on for the sandbox; recorded: the sandbox report path and the release-please-action and goreleaser versions it ran. Required before section 1.
- [ ] G-owner OWNER: the org ruleset `tags-immutable` is active on cli (target tag, `~ALL`, update + deletion + non-fast-forward, empty bypass list). Immutable releases for cli stay OFF until section 5 has verified one real draft-first release; enabling them is the owner's step after 5.2, never before this change ships a release. Required before the PR merges.
- [ ] G-first-release the first releasable merge after this change's PR has produced a cli release through the new flow; recorded: its tag. Required before section 5.

## 1. Sandbox findings (spike: design.md U1 to U5)

- [ ] 1.1 Precondition: G-sandbox ticked. Read the sandbox report. Verify: every one of U1 to U5 has an observed result (run link or command output), not an inference.
- [ ] 1.2 Write a `### Sandbox findings` subsection under design.md "Research & Decisions": one line per assumption (U1 to U5) with the observed result and evidence, plus the release-please-action and goreleaser versions. If any finding contradicts D1 to D4 (for example `sha` unset for a draft, or the next release PR not anchoring on the forced tag), stop here, report it, and revise design.md, proposal.md and the spec delta before any further section. Verify: design.md names no assumption as unverified.
- [ ] 1.3 `openspec validate adopt-draft-first-release --strict` and `task openspec:check` green, then commit `chore(openspec): record the draft-first sandbox findings`.

## 2. Tag verification script (.github/scripts/verify-release-tag.sh)

- [ ] 2.1 Write `.github/scripts/verify-release-tag.sh <tag> [expected-sha]` per design.md D3: `set -euo pipefail`; `gh release view` for `isDraft` and `targetCommitish`; peeled tag commit from `git ls-remote "https://github.com/${GITHUB_REPOSITORY:-open-platform-model/cli}" "refs/tags/$TAG" "refs/tags/$TAG^{}"` (prefer the `^{}` line); the D3 failure messages for a missing release, a missing tag and a mismatch; `draft=` and `tag=` appended to `$GITHUB_OUTPUT` when set, printed otherwise. Only read-only `gh` and `git ls-remote` calls. Verify: `shellcheck .github/scripts/verify-release-tag.sh` is clean and the file is executable.
- [ ] 2.2 Exercise it read-only against real cli tags. Verify all three: `.github/scripts/verify-release-tag.sh v1.0.0-beta.3 8cf6b15ff2330925ba702d8fbf871520eed57bba` exits 0 and prints `draft=false`; the same tag with expected `0000000000000000000000000000000000000000` exits non-zero with the mismatch message naming both commits; tag `v0.0.0-does-not-exist` exits non-zero with the missing-release message. The manual-run form (no expected SHA) on `v1.0.0-beta.3` exits 0, comparing against the release's `targetCommitish`.
- [ ] 2.3 `task fmt`, `task lint`, `task test` (see the integration note) and `task openspec:check` green, then commit `ci(release): add the release tag verification script`.

## 3. Draft-first release flow (release-please-config.json, .goreleaser.yml, release.yml)

- [ ] 3.1 `release-please-config.json`: add top-level `"draft": true` and `"force-tag-creation": true` (design.md D1); nothing else changes. Verify: `jq '.draft, ."force-tag-creation"' release-please-config.json` prints `true` twice and `jq . release-please-config.json` parses.
- [ ] 3.2 `.goreleaser.yml` `release:` gains `use_existing_draft: true`, `mode: keep-existing` and `replace_existing_artifacts: true` beside the existing `prerelease: auto` and `extra_files` (design.md D2); no `draft:` and no `name_template:` key. Verify: `go run github.com/goreleaser/goreleaser/v2@latest check` passes and `go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish` produces the five `opm-<os>-<arch>` archives and `checksums.txt` exactly as before (compare file names in `dist/` with a run on `origin/main`); remove `dist/` afterwards.
- [ ] 3.3 `.github/workflows/release.yml`, per design.md D3 and D4: the `release-please` job exposes `sha` (`steps.release.outputs.sha`); a new `verify-release` job (`needs: release-please`, the existing `always() && (...)` condition, `permissions: contents: read`, `GH_TOKEN: ${{ github.token }}`, a sparse checkout of `.github/scripts` at `github.sha`, never the tag, so a stale tag cannot supply its own verifier) runs the script with `tag_name || inputs.tag` and `sha` (empty on a dispatch) and outputs `tag` and `draft`; `goreleaser` and `publish-templates` both `needs: [release-please, verify-release]` with condition `always() && needs.verify-release.result == 'success'` and check out `needs.verify-release.outputs.tag`; `goreleaser`'s first step fails with the D4 message when `needs.verify-release.outputs.draft != 'true'`. Pass every expression into `run:` through `env:`, never inline. Verify: `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/release.yml` is clean.
- [ ] 3.4 Rewrite the `workflow_dispatch` header comment and the `tag` input description as the D4 recovery runbook (draft: re-run or dispatch; published: cut the next patch, never re-tag or delete). Verify: `grep -nE 'git (tag|push)|update-ref|release (delete|edit)|--clobber|git/refs' .github/workflows/*.yml .github/scripts/*.sh` returns no hit.
- [ ] 3.5 `task fmt`, `task lint`, `task test` (see the integration note) and `task openspec:check` green, then commit `ci(release): cut releases as drafts and publish them last`.

## 4. Agent guidance (AGENTS.md)

- [ ] 4.1 Add one line to AGENTS.md "Repository Rules": release tags are immutable per the workspace root `AGENTS.md` rule; a broken cli release is fixed by the next release (a Go `retract` in it when the broken version must not be selected), a draft release is finished with the release workflow's manual run, and no tag or GitHub Release is ever moved, deleted or re-created. Verify: the line names no fixture or release version and adds no other prose.
- [ ] 4.2 `task fmt`, `task lint`, `task test` (see the integration note) and `task openspec:check` green, then commit `chore(agents): point at the workspace release-tag rule`.

## 5. First release verified, archive (after merge, gate G-first-release)

- [ ] 5.1 Precondition: G-first-release ticked. Verify on the recorded tag, read-only: the release workflow run shows `verify-release`, `goreleaser` and `publish-templates` green; `gh release view <tag> --json isDraft,isPrerelease,assets,name` shows `isDraft: false`, `isPrerelease: true`, name equal to the tag and the five archives, `checksums.txt` and `LICENSE`; `gh release list` shows exactly one release for the tag; the peeled tag commit equals the release PR's merge commit; and the next release PR (once one is open) proposes the following version with a changelog that starts after the recorded tag.
- [ ] 5.2 **OWNER** enables immutable releases for cli (org policy, Selected) once 5.1 is green. Not a worker task and not a precondition for 5.3.
- [ ] 5.3 On a fresh branch from `origin/main`, `openspec archive adopt-draft-first-release` (syncs `openspec/specs/release-workflow/spec.md`). Verify: `task openspec:check` green and the main spec carries the two ADDED requirements and every MODIFIED scenario heading unchanged.
- [ ] 5.4 `task fmt`, `task lint`, `task test` (see the integration note) and `task openspec:check` green, then commit `chore(openspec): archive adopt-draft-first-release`.
