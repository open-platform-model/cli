## Context

The release cascade (workspace RELEASING.md) moves upstream pins into each repo through one rolling `deps/cascade` PR and leaves releases to humans. Before any of that runs, each repo needs its release-PR gates and its settings files ready. For the cli that means:

- **CI layout today.** `pr.yml` runs seven independent jobs on `pull_request` (`lint` at `.github/workflows/pr.yml:29-45`); `ci.yml` runs `lint` and `unit` on every branch push (`.github/workflows/ci.yml:16-32`). Both `lint` jobs are named `Lint` and already set up Go 1.26.0. release-please runs as the opm-release-please App (`release.yml:68-81`), so its PRs and pushes to `release-please--branches--main--components--opm` trigger both workflows. No status check is required on `main`; active branch rulesets are `mention-guard` (required workflow on `main`) and `release-branches` (`release/*` only); `Protected` is disabled (read with `gh api repos/open-platform-model/cli/rulesets` on 2026-10-01; the tag rulesets `tags-create-app-only` and `tags-immutable` do not gate PRs).
- **Pins G1 inspects.** `go.mod:13` requires `github.com/open-platform-model/library v1.0.0-beta.1` (the only OPM Go module; no `replace`). The templates pin core `v2.0.0-beta.1` and the opm catalog `v4.4.4` (`templates/*/cue.mod/module.cue`). `PinnedOperatorVersion = "v1.0.0-beta.2"` (`internal/operator/manifest.go:19`) matches the digest-pinned image at `internal/operator/dist/install.yaml:1611`; `task operator:sync` writes both (`Taskfile.yml:390-408`). No `cue.mod/local-module.cue` is tracked. On `origin/main` (f3569b24) every G1 check passes, verified by hand on 2026-10-01 with the commands in D1.
- **Last released version.** At this branch's base (`f3569b24`) `.release-please-manifest.json` holds `1.0.0-beta.4` and tag `v1.0.0-beta.4` pins operator `v1.0.0-beta.2`. Since then `main` gained cli PR 269 (`PinnedOperatorVersion = "v1.0.0-beta.4"`) and released `v1.0.0-beta.5` (PR 267), which embeds `v1.0.0-beta.4`. This change touches neither operator file, so the merged tree carries `main`'s pin.
- **Labels.** `.github/workflows/labels.yml:18-22` runs `crazy-max/ghaction-github-labeler` v6.0.0 with `skip-delete: false`, as a dry run on PRs and for real on `main`, triggered only when `.github/labels.yml` changes. The repo holds five labels the file does not list (`gh label list`, 2026-10-01): `autorelease: pending` and `autorelease: tagged` (color `ededed`, no description), `dependencies` (`0366d6`, "Pull requests that update a dependency file"), `go` (`16e2e2`, "Pull requests that update go code") and `github_actions` (`000000`, "Pull requests that update GitHub Actions code"). The plan named only the first three; `go` and `github_actions` are Dependabot's ecosystem labels and would be deleted just the same.
- **Merge settings.** `squash_merge_commit_title` is `COMMIT_OR_PR_TITLE` and `squash_merge_commit_message` is `COMMIT_MESSAGES` (`gh api repos/open-platform-model/cli`, 2026-10-01). `pr-title.yml:3-10` describes the `PR_TITLE` behavior. The owner decided on 2026-10-02 that `squash_merge_commit_message` becomes `BLANK` (not `PR_BODY`): the ruleset-required mention-guard ignores `edited`, so a PR body edited after a green run would reach `main` unchecked. Squash commits then carry only the PR title; a breaking change is a `!` in the title, and a forced version is a `release-as` key in `release-please-config.json`, landed by a normal PR.

## Goals / Non-Goals

**Goals:**
- G1 and G4 exist and run on release-please PRs, ready to be made required by the owner's ruleset.
- The next label sync deletes nothing a tool relies on, and the cascade labels exist before the cascade runs.
- Dependabot stops proposing library bumps.
- Docs-only commits stop releasing (owner decision 2026-10-01 (RELEASING.md, Pin classes)).
- No comment in the repo describes the squash setting wrongly.

**Non-Goals:**
- The cascade task, the receiver and the notify job (`add-deps-cascade-task`, `join-release-cascade`).
- G2 freshness and G3 settled statuses (set by the receiver in later phases).
- The cluster-backed e2e CI job (`add-embedded-operator-e2e-job`), which retires G4.
- Moving the embedded operator (cli PR 269 already moved it to `v1.0.0-beta.4` on `main`).
- Rulesets, merge settings, the cascade App and Environments (owner actions, RELEASING.md "Owner settings").

## Decisions

### D1: G1 is one script, called from two `lint` jobs and one task

`.github/scripts/release-pin-check.sh` (bash, `set -euo pipefail`, collects failures into an array and prints them all before exiting 1) performs, from the repo root, the five failure kinds of the shared G1 rule (workspace RELEASING.md, section "Gates"). Every probe below runs inside an `if` or behind `||`, so under `set -e` no probe's exit status ends the script; only the collected failures decide the exit code. The probes:

```bash
go mod edit -json | jq -e '.Replace == null'                          # no replace
go mod edit -json | jq -r '.Require[] | select(.Path | startswith("github.com/open-platform-model/")) | "\(.Path) \(.Version)"'
  # each version: grep -E '([-.]0\.|-)[0-9]{14}-[0-9a-f]{12}$' must not match (all three pseudo-version shapes,
  # checked on 2026-10-01 against v0.0.0-…, v1.2.4-0.…, v1.2.3-pre.0.…, and not matching v1.0.0-beta.1)
  # repo = path with any /vN major suffix removed; git ls-remote --exit-code --tags "https://$repo.git" "refs/tags/$version"
  # exit 0 = tag exists; exit 2 = no such tag (violation); any other exit = lookup failure (violation, never a pass)
grep -HnE 'v: "[^"]*-0\.dev\.' templates/*/cue.mod/module.cue        # must not match
git ls-files '*cue.mod/local-module.cue'                              # must be empty
sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p' internal/operator/manifest.go
grep -E '^\s*image: ghcr\.io/open-platform-model/opm-operator:' internal/operator/dist/install.yaml  # exactly one line (zero or several is a violation); tag before '@'
```

The workflow step, added to the `lint` job of both `pr.yml` and `ci.yml` after `setup-go`:

```yaml
- name: Release-pin gate (G1)
  if: startsWith(github.head_ref || github.ref_name, 'release-please--')
  run: .github/scripts/release-pin-check.sh
```

`task deps:release-check` runs the same script with no branch condition. Failure lines name the file, the pin and the fix (for the operator: `task operator:sync VERSION=<tag>`); exit codes are 0 (pass) and 1 (any violation or any lookup failure; a failed `git ls-remote` that is not "no such ref" is reported as a lookup failure, never as a pass).

### D2: G4 is its own workflow, because it must re-run on label events

`.github/workflows/release-evidence.yml`, name `Release Evidence`, job `operator-e2e-evidence` (check name `G4 operator-embed evidence`), triggered by `pull_request` on `main` with types `[opened, synchronize, reopened, labeled, unlabeled]`, `permissions: contents: read`, a concurrency group per PR number. The job has no job-level `if`: it always runs and decides in its step, so a skipped job can never stand in for a pass. It checks out with `fetch-depth: 0` (tags and `origin/main`), then:

```bash
branch="${HEAD_REF:-$REF_NAME}"
case "$branch" in release-please--*) ;; *) echo "G4 applies to release-please PRs only"; exit 0 ;; esac
tag="v$(git show "origin/${BASE_REF}:.release-please-manifest.json" | jq -er '."."')" || fail "cannot read the last cli version from origin/${BASE_REF}"
git rev-parse -q --verify "refs/tags/$tag" >/dev/null || fail "last cli tag $tag not found"
old=$(git show "${tag}:internal/operator/manifest.go" | sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p')
new=$(sed -n 's/^const PinnedOperatorVersion = "\(.*\)"$/\1/p' internal/operator/manifest.go)
[ -n "$old" ] && [ -n "$new" ] || fail "cannot read PinnedOperatorVersion at $tag or in the PR"
[ "$old" = "$new" ] && exit 0
jq -e 'index("e2e-verified")' <<<"$LABELS" >/dev/null || fail "operator moved $old -> $new since $tag; run task test:e2e and add e2e-verified (interim until add-embedded-operator-e2e-job)"
```

`fail` prints its message and exits 1; the tag is verified before `git show "${tag}:…"` runs, so a missing tag is reported by name instead of aborting the pipeline under `set -euo pipefail`. `HEAD_REF`, `REF_NAME`, `BASE_REF` and `LABELS` (`toJSON(github.event.pull_request.labels.*.name)`) reach the script through `env:`, never inline. The logic lives in `.github/scripts/release-evidence.sh` so it can be exercised locally with those variables set.

### D3: G4's label is not bound to a commit

The label proves a human ran the suite at some point on the release PR. If the cascade later moves the operator again and release-please refreshes the PR, the label stays and G4 passes. Accepted for an interim gate: a human removes `e2e-verified` whenever the operator pin moves after labeling. Workspace RELEASING.md, section "Runbook", must carry that line under "Merging release PRs" (on a cli release PR whose `PinnedOperatorVersion` moved since the last cli tag, run `task test:e2e` and add `e2e-verified`; remove it if the pin moves again before merge). Its current step 8 under "Handling a cascade PR" puts the label on the cascade PR, which G4 never reads; flagged to the workspace item, and the real fix is the CI job that replaces G4.

### D4: Bot labels are listed with their live color and description

The labeler rewrites color and description to match the file, so listing a bot label with a different color would edit it on every sync. The five bot labels go in a new `# Managed by bots` group with exactly the values in Context. The six cascade labels go in a new `# Release cascade` group. The shared definition is workspace RELEASING.md, section "The cascade" › Labels; the cli copies the values from that table (workspace commit `57bf4a8`, task 3.1). The cascade receiver (`add-release-cascade-workflows`) must not create labels in the cli, where `labels.yml` owns them. Copied values: `deps-cascade` (`0366d6`, "Rolling upstream-pin PR opened by the release cascade"), `deps-cascade:conflict` (`b60205`, "The bot could not merge main into this cascade PR; a human resolves it"), `deps-cascade:hold` (`fbca04`, "A human is working on this cascade PR; the bot does not push"), `deps-cascade:breaking` (`d93f0b`, "An upstream changelog in this PR announces a breaking change"), `need-human-review` (`e99695`, "Glue edits a human must review before merging"), `e2e-verified` (`0e8a16`, "A human ran task test:e2e against the embedded operator (G4)").

### D5: Hiding docs delays cli, library and operator docs fixes on the site

Owner decision 2026-10-01 (RELEASING.md, Pin classes) hides `docs` in cli. opmodel.dev's line versions build cli docs from the newest cli tag and library and opm-operator docs from exactly what that tag pins (`opmodel.dev/site/versions.conf` header; core and catalog_opm docs come from their `release/<prefix>vX.Y` branch or `main`, so they are unaffected). A docs-only fix therefore reaches the published site only after the next releasing commit in the cli (and, for library or operator docs, after the cascade has carried their next release into a cli release). The owner resolved this on 2026-10-02 (RELEASING.md, Rollout and changes): a new opmodel.dev change, `build-docs-from-branch-head`, builds library, opm-operator and cli docs from their branch head like core and catalog_opm. Section 4 (the docs-hiding commit) MUST NOT merge before that change merges; if it does, a docs-only fix in this repo reaches opmodel.dev only with the next release.

A second effect concerns templates. The AGENTS.md Template rule forces a template version bump for any change under `templates/<t>/`, and templates publish only from the release workflow. A `docs`-typed edit there would bump the template version yet publish nothing until the next releasing commit (PR #161, `docs(config): propose seeding both catalogs…`, touched `templates/`). AGENTS.md therefore states that edits under `templates/` are never typed `docs` (task 4.2).

### D6: What `retire-g4-operator-embed-evidence` removes

G4 is interim. It retires only when all of these hold: `add-embedded-operator-e2e-job` has merged, its e2e check ("E2E (kind, embedded operator)") has passed on at least one cli release PR, and the owner has made that check required (owner-approved 2026-10-02; RELEASING.md, Gates). Retiring it is its own later cli OpenSpec change, working name `retire-g4-operator-embed-evidence`. G4 is a whole workflow, not a step inside one, so that change removes, in this order:

1. **Owner first:** the owner adds "E2E (kind, embedded operator)" to the cli ruleset's required checks and removes `G4 operator-embed evidence` from them. Deleting the workflow while its check is still required would leave every PR pending.
2. A REMOVED delta for the `release-gates` requirement "A moved embedded operator on a release PR needs e2e evidence".
3. `.github/workflows/release-evidence.yml` (the `G4 operator-embed evidence` check) and `.github/scripts/release-evidence.sh`, deleted.
4. The `e2e-verified` entry in `.github/labels.yml` (the label sync then deletes the label from the repo) and the G4 sentence in `AGENTS.md`.
5. A workspace PR updates RELEASING.md: the G4 row in "Gates", the `e2e-verified` row in "The cascade" › Labels, the G4 step in "Runbook" › "Merging release PRs", and the cli line in "Owner settings" › "Rulesets on main".

Steps 2 to 4 are one `ci(release)` commit riding that change's PR, archive included. G1, including its operator-embed check, stays.

### D7: The local test gate skips the cluster half of `task test`

`task test` runs `task test:integration` (needs the `kind-opm-dev` cluster) and `task test:e2e` (destructive on that shared cluster). This change adds no Go code, so the local gate is `task test:unit` plus `go test ./tests/e2e/...` with no kubeconfig, so the cluster tests skip (tasks.md, integration note), together with `task fmt`, `task lint` and `task openspec:check`. The supervisor accepted this explicit local gate in place of `task test` on 2026-10-02.

## Research & Decisions

### Where G1 runs
**Context**: G1 must not show as passing when it did not run.
**Explored**: `pr.yml` and `ci.yml` job layout; GitHub's treatment of skipped jobs (a skipped required job counts as success).
**Options considered**:
1. A new job with `if: startsWith(...)` - skipped on ordinary PRs, which is fine, but a skipped job also reads as a pass on a release PR if the condition is ever wrong.
2. A step in the `pr.yml` `lint` job only - simplest, but both workflows report a check named `Lint` on a release-please head commit, so a required `Lint` could be satisfied by the push workflow's run, which would lack the step.
3. A step in both `lint` jobs - both `Lint` runs enforce it.
**Decision**: option 3.
**Rationale**: matches the plan's "step inside an existing required job" rule and removes the duplicate-name hole. The `ref_name` fallback covers the push workflow, where `head_ref` is empty.

### PinnedOperatorVersion check: script or unit test
**Context**: `task operator:sync` writes the pin and the manifest together, so a mismatch is only possible by hand-editing one of them.
**Options considered**:
1. A G1 script check on release PRs - as scoped.
2. A unit test in `internal/operator` asserting the image tag equals the pin - runs on every PR, so a mismatch never reaches `main`.
**Decision**: option 1 in this change.
**Rationale**: keeps the change inside its scope and makes G1 self-contained across repos. Option 2 is cheap and stricter; it is open question Q2.

### Last cli tag for G4
**Context**: G4 compares against "the last cli tag".
**Options considered**:
1. Newest tag by semver from `git tag`.
2. Newest published GitHub Release.
3. The base branch's `.release-please-manifest.json` version.
**Decision**: option 3.
**Rationale**: it is the version release-please itself counts from, needs no API call, and on the release PR the manifest is already bumped, so reading it from the base branch gives exactly the previous release.

### go.work and the two extra Dependabot labels
**Context**: the plan listed neither.
**Options considered**: (1) G1 also refuses a tracked `go.work`; (2) G1 keeps exactly the shared rule's five failure kinds.
**Decision**: option 2 for `go.work`; `labels.yml` does list `go` and `github_actions`.
**Rationale**: no sibling `prepare-release-cascade` checks `go.work`, so the cli's G1 would differ from opm-operator's, which is also a Go module; and a tracked `go.work` pointing at `../library` already breaks CI builds, so the check adds little. If the workspace wants it, it belongs in the shared G1 rule first. The two Dependabot labels are a real hole: the sync would delete them like the three the plan named (flagged to the workspace item for RELEASING's Labels paragraph).

## Risks / Trade-offs

- [Gates are advisory until the ruleset requires them] → Owner action S0; RELEASING.md "Owner settings" › "Rulesets on main" names the cli checks `Lint` and `G4 operator-embed evidence` (gate G-workspace checks it on workspace `main`).
- [G1's `git ls-remote` makes the `lint` job depend on github.com] → Only on release PRs; a network failure fails closed and a re-run clears it.
- [G4 label not tied to a SHA] → D3.
- [A typo in the G1 step's `if:` would skip G1 forever while every local check passes] → gate G-release-pr-run reads the step on real release-please runs.
- [Docs-only fixes no longer release, so they reach opmodel.dev late] → D5; gated on opmodel.dev `build-docs-from-branch-head` merging before section 4 merges. Until it merges, a docs-only fix in this repo reaches opmodel.dev only with the next release.
- [The labels PR's dry run is the only pre-merge proof of D4] → Task 3.1 compares the file against `gh label list` before commit; gate G-labels-dry-run reads the dry-run log on the PR.

## Open Questions

- **Q1 (owner, resolved 2026-10-02)**: opmodel.dev builds library, opm-operator and cli docs from the branch head like core and catalog_opm (change `build-docs-from-branch-head`); see D5.
- **Q2 (owner)**: add the unit test of the Research section as well, so a mismatched embed fails every PR, not only release PRs?
- **Q3 (owner)**: `amannn/action-semantic-pull-request` can also validate a single commit's subject (`validateSingleCommit`, `validateSingleCommitMatchesPrTitle`), which closes the `COMMIT_OR_PR_TITLE` gap without the owner setting. It would also reject a one-commit cascade PR whose commit subject is not conventional, so it needs `add-deps-cascade-task` to agree. Not in this change.
