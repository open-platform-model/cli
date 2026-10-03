## Why

The cli is the last tier of the release cascade (workspace RELEASING.md, section "Release order"). It ships four upstream pins:

- library as a Go module (`go.mod:14`, `v1.0.0-beta.3`);
- the opm-operator release embedded as `internal/operator/dist/install.yaml` and pinned by `PinnedOperatorVersion` (`internal/operator/manifest.go:19`, image line at `install.yaml:1643`);
- the opm catalog and core in the three templates (`templates/{minimal,standard,advanced}/cue.mod/module.cue:9-13`), each with its own version (`templates/*/identity/identity.cue:14`).

It also pins the catalog and core in the test trees, the kind Platform and the podinfo fixture.

Today a human moves these pins by hand, or through the workspace root tasks (`deps:update:templates`, `deps:pins:platform-pins`, `deps:pins:fixtures`; `AGENTS.md:128`, `AGENTS.md:231`). Those root tasks do not follow the cascade rules:

- they move pins to a bare major instead of an exact published version;
- they ignore `.cascade-frozen` and `.cascade-hold`;
- they advance the fixture version on every run.

Phase 2 of the rollout gives each repo a `task deps:cascade` that the Phase 3 receiver runs (workspace RELEASING.md, sections "The cascade" › "The receiver" and "Rollout and changes"). It also gives each repo the title and body tasks the receiver uses for its one rolling PR. This change adds the cli's task.

## What Changes

- **New tasks** in `Taskfile.yml`, next to `deps:release-check` (`Taskfile.yml:520-523`):
  - `deps:cascade` moves every cli pin listed in workspace RELEASING.md, section "What each repo's task moves", in the working tree only. It exits 0 when the tree changed, 3 when there was nothing to do, and any other code on error.
  - `deps:cascade:title` and `deps:cascade:body` are thin wrappers over the shared resolver's `title` and `body`.
  - `deps:cascade:test` is the task's own scenario test.
  - The task interface, exit codes and rules follow the Phase 2 cascade contract §5 (`p2-cascade-contract.md` in the supervisor's scratchpad, cited below as "contract §N").
- **New scripts** under `.tasks/cascade/`:
  - `cascade.sh`, the mover;
  - `pins.sh`, which reports the four logical pins: library, opm-operator, opm catalog and core;
  - `classes`, the path-class map, verbatim from contract §5.3;
  - `test.sh`;
  - `testdata/stub-resolve.sh`, byte-identical to contract §7;
  - `testdata/older.tsv` and `testdata/s1-calls.txt`.
- **What `deps:cascade` moves** (contract §6.4):
  - **library:** `go get github.com/open-platform-model/library@<exact>`, then `go mod tidy`.
  - **operator:** `task operator:sync VERSION=<v>` to the newest published, non-draft operator release that carries `install.yaml`.
  - **catalog and core** as a consistent set, in:
    - the three templates;
    - `hack/platform`, plus `hack/kind-platform.yaml`;
    - `examples`;
    - the podinfo fixture;
    - `tests/e2e/testdata/operator-owned`;
    - the six testdata `cue.mod` files.
  - **version advances**, once per PR, for the three templates and the podinfo fixture.
  - **podinfo consumers:** the podinfo pin in `examples` and `operator-owned` follows the fixture in the same PR.
  - **warnings:** for an unpublished docs bundle and for a `language.version` newer than the cli's pinned CUE.
- **CI:**
  - The offline test set runs as one step in `pr.yml`'s required `Unit Tests` job (`.github/workflows/pr.yml:57-89`). That job gains a SHA-pinned `go-task/setup-task` step.
  - A new, non-required workflow `.github/workflows/cascade-task.yml` (job `Cascade task (network)`) runs the full set against GHCR, the Go proxy and GitHub releases.
- **`AGENTS.md`** names `task deps:cascade` where it now names the workspace root tasks for templates and fixtures.
- **Main spec `test-fixture-lineage`:** its "Old test pins are current or frozen with a reason" requirement says core moves to "the newest published release". That contradicts the consistent-set rule (workspace RELEASING.md, "The cascade" › "The receiver"): core follows the version the catalog pins. The requirement is restated to match. Its three scenarios are kept.

Release class: none. This is CI and repo tooling, titled `ci(cascade): add the deps:cascade tasks` (contract §10). Users get nothing: no command, flag or shipped file changes. After GA it would still cut no release.

## Depends on / gates

- **`.github` `add-cascade-resolver`** (Phase 2) must be merged on `.github` `main` before this PR merges.
  - The tasks find the resolver at `../.github/.github/scripts/cascade/cascade-resolve.sh` from the main checkout, or at `CASCADE_RESOLVER` (contract §3).
  - The network job checks out `.github` at `main`.
  - Scenario S5 (title and body) runs only against the real resolver.
  - Sections 1 to 3 of this change build and test against the contract §7 stub, so they do not wait for the resolver.
- **cli `prepare-release-cascade`** is merged and archived (`openspec/changes/archive/2026-10-02-prepare-release-cascade`). It provides the cascade labels in `.github/labels.yml` and G1 (`task deps:release-check`).
- **cli `bump-stale-testdata-pins`** is merged and archived (`openspec/changes/archive/2026-10-02-bump-stale-testdata-pins`). It provides `.cascade-frozen` and brings the six testdata trees current.
- **Not part of this change:**
  - **The Phase 2 gate** (workspace RELEASING.md, "Rollout and changes" › "Phases": "a run on `main` exits 3") is met by a separate catch-up PR (contract §8). The supervisor opens it after the operator's catch-up release is published. Today `main` is behind: the catalog is `v4.4.4` and the newest published is `v4.5.1`.
  - **Wiring the receiver** is Phase 3 (`join-release-cascade`).
  - **Rewiring the workspace `task deps:update`** is Phase 5.
- **Leaves stay manual** (owner decision 15). The cascade stops at the cli, and this task never touches another repo.

## Capabilities

### New Capabilities

- `deps-cascade`: `task deps:cascade` and its title, body and test tasks. This covers:
  - what the task moves, and how it resolves and orders the moves;
  - what it never touches;
  - its exit codes;
  - the scenario test and where CI runs it.

### Modified Capabilities

- `test-fixture-lineage`: "Old test pins are current or frozen with a reason" now names the consistent set the cascade applies (catalog to the newest published, core to the version that catalog pins, never backwards), instead of the newest published core.

## Impact

- **Commands:** none. No user-facing opm command, flag or output changes.
- **Packages:** none. No Go code changes.
- **Files:**
  - new: `.tasks/cascade/{cascade.sh,pins.sh,classes,test.sh}`, `.tasks/cascade/testdata/{stub-resolve.sh,older.tsv,s1-calls.txt}` and `.github/workflows/cascade-task.yml`;
  - edited: `Taskfile.yml`, `.github/workflows/pr.yml` (`Unit Tests` job) and `AGENTS.md`.
- **Network:** `deps:cascade` reads GHCR, `proxy.golang.org` and GitHub release downloads anonymously, through the resolver. It never publishes, pushes or commits.
- **Risks:**
  - The first real run (the catch-up PR) moves the catalog from `v4.4.4` to `v4.5.1` in shipped templates. If `v4.5.x` breaks a template or fixture, that PR needs hand fixes. This change does not.
  - The docs pins the cli reports (`hack/docskit-dump pins`) take core from library's `DefaultSchemaModule` (`v2.0.0-beta.2` at library `v1.0.0-beta.3`). The templates take core from the catalog's pin: `v4.5.1` pins `v2.0.0-beta.1`, checked live 2026-10-04. The two can differ. That is expected under the consistent-set rule and fails no gate.
