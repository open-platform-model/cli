## Why

OpenSpec 1.14.1 raises "Requirement text is very long (>500 characters)" from INFO to WARNING. `openspec validate --strict` fails on any warning, and no config key or flag turns this one off: the limit is a constant in the validator. The warning is right on the merits. A long requirement usually holds several behaviors, and one behavior per requirement makes each one testable by its own scenarios.

The cli has 215 requirements over 500 characters in 59 of its 68 main specs (measured on `origin/main` 88268e46 with 1.14.1: `openspec validate --all --strict --json`, 215 warnings, 0 errors). This is why `task openspec:check` runs without `--strict` since the bump to 1.14.1. This change rewrites those requirements so that `--strict` can pass again, with no rule lost or changed, and then puts `--strict` back. The job names "OpenSpec Specs (strict)" in `ci.yml` and `pr.yml` become true again.

## What Changes

Spec deltas, in eight batches of at most 28 requirements. A batch holds whole specs, and no spec is in two batches. The long requirement counts are in brackets.

- **cli-B1 (28):** `operator-lifecycle` (17), `apply-pruning` (11).
- **cli-B2 (28):** `deploy` (14), `platform-resolution` (9), `artifact-publishing` (5).
- **cli-B3 (28):** `release-workflow` (8), `kernel-render` (7), `deps-cascade` (6), `instance-init` (6), `authoring-commands` (1).
- **cli-B4 (28):** `instance-inventory` (6), `mod-status` (6), `release-cascade-wiring` (6), `cmdutil` (5), `platform-check` (5).
- **cli-B5 (28):** `test-fixture-lineage` (5), `cmd-structure` (4), `e2e-cluster-preconditions` (4), `inst-commands` (4), `kind-cluster-tasks` (4), `mod-vet` (4), `catalog-compatibility` (3).
- **cli-B6 (28):** `published-module-resolution` (4), `template-modules` (4), `dependency-tidy` (3), `e2e-cluster-workflow` (3), `errors-domain` (3), `flag-conventions` (3), `inventory-ownership` (3), `log-output-style` (3), `apply-preflight-gates` (2).
- **cli-B7 (28):** `pr-workflow` (3), `catalog-registry-check` (2), `ci-workflow` (2), `config` (2), `config-commands` (2), `config-types` (2), `docs-bundle` (2), `health-evaluation` (2), `instance-building` (2), `k8s-helpers` (2), `mod-apply` (2), `module-synthetic-instance` (2), `release-gates` (2), `command-reference` (1).
- **cli-B8 (19):** `repo-automation` (2), `resource-conversion` (2), `resource-discovery` (2), `status-verbose` (2), `workflow-security` (2), `instance-identity-labeling`, `mod-instance-optional`, `mod-list`, `pkg-types`, `platform-pull`, `refactoring-requirements`, `registry-login`, `secret-inventory-migration`, `status-wide-output` (1 each).

The delta files are written batch by batch. Each rewrite is a split or a move into scenarios. The rules are in `design.md`.

Repo extras, in the last unit:

- **Gate.** `task openspec:check` in `Taskfile.yml` runs `openspec validate --all --strict --no-interactive` again, and its `desc` loses the "`--strict` is off" text.
- **Text.** The `task openspec:check` row of `AGENTS.md` and the Validation Gates line 4 in `openspec/config.yaml` lose the same text.
- **CI.** `ci.yml` and `pr.yml` already call `task openspec:check`, so they need no edit. Their job names "OpenSpec Specs (strict)" become true.

Affected commands and packages: none. No Go file, flag, exit code or output changes.

Release class: none, as it would be after GA and in beta. Spec text, a task and CI text only; commit types are `docs` for specs and `ci` or `chore` for the gate, so release-please cuts no release.

Delivery: one PR per change, the default. The change has ten task sections, more than the "about five" that the repo rules name. See `design.md` section "Sections" for why this is one change. For the one-sentence checks that each rewritten body needs, see the section "Strength table" of `design.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

The 59 specs named above. Each keeps its capability name. Requirement names may change in splits, and the scenario ledger in `design.md` records every old scenario name.

## Enhancement

None.

## Impact

- **Files:** `openspec/specs/*/spec.md` (59 files, through deltas), `Taskfile.yml`, `AGENTS.md`, `openspec/config.yaml`.
- **Risks:** a rewrite can lose or change a rule without any tool error. Mitigations are the byte-equal check of moved scenarios, the scenario ledger and the gate, all in `design.md`. A reference to an old requirement header can go stale after a split; each unit greps for it. No other open change exists in the cli, so no collision is possible today (checked 2026-10-19: `openspec/changes/` holds only `archive/`).
