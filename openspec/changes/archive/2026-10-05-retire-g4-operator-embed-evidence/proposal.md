## Why

G4 (`G4 operator-embed evidence`, `.github/workflows/release-evidence.yml`) makes a human run `task test:e2e` locally and add the label `e2e-verified` to a release-please PR whose pinned operator module moved since the last cli tag. It was built by `prepare-release-cascade` as an interim gate until a cluster-backed e2e job ran in CI, and retires only when all three conditions in workspace RELEASING.md, section "Gates" ("G4 retirement", owner decision 2026-10-02) hold. All three hold on 2026-10-05:

1. `add-embedded-operator-e2e-job` merged (cli PR 272, 2026-10-02).
2. Its check "E2E (kind, embedded operator)" ran the whole suite and passed on cli release-please PRs, about ten times since 2026-10-02, most recently run 37272176296 on the `1.0.0-beta.10` release PR (step "Run E2E tests (cluster required)" green).
3. That check is required by the cli `main` ruleset (id 24450746), next to `Lint` and `G4 operator-embed evidence`.

The CI job is a strictly stronger gate: it runs on every release PR, not only when the module pin moved, and it cannot be satisfied by a stale label. Keeping G4 only adds a manual step to releases.

## What Changes

- **Remove G4.** Delete `.github/workflows/release-evidence.yml` and `.github/scripts/release-evidence.sh`, and remove the `release-gates` requirement "A moved operator module pin on a release PR needs e2e evidence".
- **Remove the label.** Drop `e2e-verified` from `.github/labels.yml` (the label sync then deletes it from the repository) and from the `repo-automation` requirement that lists the cascade labels (six become five).
- **Docs.** `AGENTS.md`'s release sentence drops the `e2e-verified` clause.
- **Outside this change, done by the supervisor:** the owner-delegated edit of the cli `main` ruleset (AGENTS.md "One narrow exception") removes `G4 operator-embed evidence` from the required checks before this PR merges, because a PR that deletes the workflow never reports that check; and a workspace PR updates RELEASING.md (the G4 rows in "Gates", "The cascade" › Labels, "Runbook" › "Merging release PRs", "Owner settings" and the rollout table).

Not changed: `.github`'s wiring test asserts `e2e-verified` is not a bot label, which stays true.

Release class: none. CI, labels, specs and docs; no command, flag, output or shipped file changes. PR title `ci(release): retire the G4 operator-embed evidence check`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-gates`: the G4 requirement is removed.
- `repo-automation`: the label sync no longer keeps `e2e-verified`.

## Impact

- **Files:** `.github/workflows/release-evidence.yml`, `.github/scripts/release-evidence.sh` (deleted), `.github/labels.yml`, `AGENTS.md`, `openspec/specs/{release-gates,repo-automation}/spec.md`.
- **Risk:** if the ruleset still requires `G4 operator-embed evidence` when this merges, every later PR waits on a check that never reports. The ruleset edit comes first.
- **Open release PR:** the `1.0.0-beta.10` release PR no longer needs the label once the ruleset edit lands.
