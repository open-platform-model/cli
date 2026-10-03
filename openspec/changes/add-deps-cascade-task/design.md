## Context

**Sources.** The design source is workspace RELEASING.md, sections "The cascade" (with "The receiver" and "What each repo's task moves"), "Gates", "Cascade files" and "Rollout and changes". The binding interface is the Phase 2 cascade contract, version 1: `p2-cascade-contract.md` in the supervisor's scratchpad, cited as "contract §N". Where the two disagree, RELEASING.md wins, and the conflict goes to the supervisor (contract preamble).

**How the work is split** (contract §1):

- The shared resolver in `.github` owns every version lookup, semver order, hold, frozen check, title and body. `.github` change `add-cascade-resolver` builds it.
- The cli owns its data and its edits:
  - `.tasks/cascade/pins.sh` and `.tasks/cascade/classes`;
  - `.tasks/cascade/cascade.sh`, the mover;
  - the four Taskfile tasks.

**The cli's pins today** (all read 2026-10-04 on `origin/main` `ae60f007`):

| Pin | Location | Value | Class |
| --- | --- | --- | --- |
| library | `go.mod:14` | `v1.0.0-beta.3` | shipped |
| opm-operator | `internal/operator/manifest.go:19` `PinnedOperatorVersion`, plus the image line `internal/operator/dist/install.yaml:1643` | `v1.0.0-beta.5` | shipped |
| opm catalog, core | `templates/{minimal,standard,advanced}/cue.mod/module.cue:9-13` | `v4.4.4`, `v2.0.0-beta.1` | shipped |
| template versions | `templates/*/identity/identity.cue:14` | `1.0.3` | shipped, own version |
| opm catalog, core (plus third-party `cue.dev/x/k8s.io@v0` `v0.12.0`, never moved) | `hack/platform/cue.mod/module.cue:6-13` | `v4.4.4`, `v2.0.0-beta.1` | test |
| opm catalog (bare) | `hack/kind-platform.yaml:21-22` | `4.4.4` | test |
| catalog, core, podinfo `v0.1.11` | `examples/cue.mod/module.cue:9-16` | `v4.4.4`, `v2.0.0-beta.1` | test |
| catalog, core | `tests/fixtures/modules/podinfo/cue.mod/module.cue:9-13` | `v4.4.4`, `v2.0.0-beta.1` | test |
| podinfo version | `tests/fixtures/modules/podinfo/identity/identity.cue:18` | `0.1.11` | test, own version |
| catalog, core, podinfo `v0.1.11` | `tests/e2e/testdata/operator-owned/cue.mod/module.cue:9-16` | `v4.4.4`, `v2.0.0-beta.1` | test |
| catalog, core | `internal/instinit/testdata/initvalues`, `internal/workflow/render/testdata/skip-unprovided`, `tests/e2e/testdata/duplicate-identities`, `tests/integration/module-apply/testdata` (each `cue.mod/module.cue`) | `v4.4.4`, `v2.0.0-beta.1` | test |
| core only | `tests/fixtures/valid/{simple-module,module-with-debug-values}/cue.mod/module.cue:9-10` | `v2.0.0-beta.1` | test |
| frozen | `.cascade-frozen:3-12`: three Go test files | old core and catalog literals | never moved |

**Newest published today** (live, 2026-10-04):

| Upstream | Newest published | Note |
| --- | --- | --- |
| library | `v1.0.0-beta.3` | the proxy `@v/list` lists beta.1, beta.2, beta.3 |
| opm-operator | `v1.0.0-beta.5` | |
| `catalogs/opm@v4` | `v4.5.1` | tags `v4.4.3`, `v4.4.4`, `v4.4.5`, `v4.5.0`, `v4.5.1` and older |
| core | `v2.0.0-beta.2` | |

What the catalog modulefiles pin for core:

| Catalog | Pins core |
| --- | --- |
| `v4.4.3` | `v2.0.0-alpha.13` |
| `v4.4.4` | `v2.0.0-beta.1` |
| `v4.5.1` | `v2.0.0-beta.1` |

## Goals / Non-Goals

**Goals:**

- `task -x deps:cascade` moves every pin in the table above, except the frozen ones, to the target the contract's rules give. It exits 0 if the tree changed, 3 if not, and any other code on error.
- Running it twice in one PR changes nothing the second time. In particular, no second version advance.
- `task -x deps:cascade:title` and `task -x deps:cascade:body` print the resolver's title and body for the cli's diff.
- `task -x deps:cascade:test` proves this with the contract §7 stub. An offline set runs in a required job; the full set runs in a non-required network job.

**Non-Goals:**

- The receive and notify workflows, the branch, the push and the PR. These are Phase 3.
- The catch-up PR that makes a run on `main` exit 3. The supervisor owns it (contract §8 and §9.7).
- Retiring or rewriting the workspace root `.tasks/deps/*.sh`. That is Phase 5, and contract §10 says to make no edits outside this repo.
- Any change to `hack/fixtures.sh`. It is byte-identical with opm-operator's copy, and the workspace `task fixtures:lint` checks that (`AGENTS.md:231`).
- `deps-cascade:breaking`, which is Phase 3 (contract §9.8).

## Decisions

### D1: Tasks live in `Taskfile.yml`, scripts under `.tasks/cascade/`

The cli Taskfile has no `includes:`. Its only `.tasks/` file is `opm-docs.sh`. The four tasks therefore go into `Taskfile.yml` after `deps:release-check` (`Taskfile.yml:520-523`), under the existing "Release tasks" banner.

Each task declares `CASCADE_RESOLVER_PATH` at task level through one YAML anchor, never as a global `vars:` entry (contract §3). Each also exports `CASCADE_RESOLVER: '{{.CASCADE_RESOLVER_PATH}}'` and has the precondition `test -x '{{.CASCADE_RESOLVER_PATH}}'`, with the contract §3 message.

```yaml
deps:cascade:            # .tasks/cascade/cascade.sh
deps:cascade:title:      # "$CASCADE_RESOLVER" title --classes .tasks/cascade/classes --pins .tasks/cascade/pins.sh
deps:cascade:body:       # "$CASCADE_RESOLVER" body  --classes .tasks/cascade/classes --pins .tasks/cascade/pins.sh
deps:cascade:test:       # .tasks/cascade/test.sh
```

**Invocation is always `task -x <name>`** (contract §3 and §9.9). Without `-x`, go-task maps `exit 3` to 201.

**Example, a run with nothing to do:**

```text
$ task -x deps:cascade; echo $?
cascade: library v1.0.0-beta.3 (current)
cascade: opm-operator v1.0.0-beta.5 (current)
cascade: opm catalog v4.4.4 -> v4.5.1
...
0
```

**Example, the second run:**

```text
$ task -x deps:cascade; echo $?
cascade: nothing to move
3
```

**Example, an error.** It exits non-zero and the resolver's own stderr line comes first:

```text
cascade-resolve: https://proxy.golang.org/...: answered `502`; refusing to guess
task: Failed to run task "deps:cascade": exit status 1
```

The `cascade:` progress lines go to stderr. Stdout stays empty, so `deps:cascade:title` and `deps:cascade:body` remain the only tasks with machine output.

### D2: `pins.sh` reports four logical pins

`pins.sh <ref>` prints one TSV row per pin (contract §4.1), each `v`-prefixed. It reads files with `cat` for `WORKTREE` and with `git show <ref>:<path>` otherwise. The representative files follow contract §6.4.

| Key | Display | Class | Read from |
| --- | --- | --- | --- |
| `github.com/open-platform-model/library` | `library` | shipped | `go.mod`: `$1 == "github.com/open-platform-model/library" {print $2}` |
| `github.com/open-platform-model/opm-operator` | `opm-operator` | shipped | `internal/operator/manifest.go`: the `const PinnedOperatorVersion = "..."` value |
| `opmodel.dev/catalogs/opm@v4` | `opm catalog` | shipped | `templates/minimal/cue.mod/module.cue`: the `v:` in the `"opmodel.dev/catalogs/opm@v4": {` block |
| `opmodel.dev/core@v2` | `core` | shipped | `templates/minimal/cue.mod/module.cue`: the `v:` in the `"opmodel.dev/core@v2": {` block |

- The labels column is empty for all four. `need-human-review` is library's.
- A pin missing at a ref is omitted.
- Template and podinfo versions are not upstream pins, so they are not listed (contract §4.1).

### D3: Catalog and core move as one consistent set across all CUE files

The CUE file list, in this order:

1. `templates/minimal`, `templates/standard`, `templates/advanced`;
2. `hack/platform`;
3. `tests/fixtures/modules/podinfo`;
4. `examples`;
5. `tests/e2e/testdata/operator-owned`;
6. `internal/instinit/testdata/initvalues`, `internal/workflow/render/testdata/skip-unprovided`, `tests/e2e/testdata/duplicate-identities`, `tests/integration/module-apply/testdata`;
7. `tests/fixtures/valid/simple-module`, `tests/fixtures/valid/module-with-debug-values`.

Each entry is a directory whose `cue.mod/module.cue` the task reads. The list is explicit and never globbed, because `.claude/worktrees/` and `bin/` can hold other `module.cue` files.

**Rules** (contract §5.2 rule 7; workspace RELEASING.md, "The cascade" › "The receiver", "Consistent set"):

- **Catalog target `T`.**
  - `T` = `newest cue opmodel.dev/catalogs/opm@v4 --current <representative catalog>`, or the representative's current value on exit 3.
  - Every listed file that pins the catalog below `T` moves to `T`.
  - A file above `T` is never lowered.
  - A frozen file for that key is skipped.
- **Core target per file.**
  - `C` is the file's catalog after the move.
  - For the two core-only files, `C` is the representative's catalog after the move (contract §5.2 rule 7).
  - Core goes to `pin-of opmodel.dev/catalogs/opm@v4 C opmodel.dev/core@v2`, but only if that is greater than the file's core.
  - If the file's core is greater, it stays, with the warning "core `<cur>` is ahead of the core `<x>` that catalog `<C>` pins" (contract §9.10; reported to the owner there).
  - Core is never resolved through `newest cue opmodel.dev/core@v2`.
- **Hold on core.**
  - `hold opmodel.dev/core@v2` is read once.
  - If an in-date `max` is below `pin-of T`, the file's catalog stays at its current value, and so does its core. The warning is "catalog `<t>` needs core `<c>`, above the hold `<max>`; catalog held too" (contract §9.11).
- **`hack/kind-platform.yaml`.** The bare `version:` under `opmodel.dev/catalogs/opm@v4:` is rewritten as text to `hack/platform`'s catalog after the move, with no `v`. The file header says it mirrors `hack/platform` (`hack/kind-platform.yaml:4-9`).
- **Writing.**
  - In a file where some key moved, the task runs one `cue mod get` naming only the moved `opmodel.dev/*` keys at their exact versions, then one `cue mod tidy`.
  - A file where nothing moved is never touched.
  - `cue.dev/x/k8s.io@v0` is never named. If `tidy` changes its `v:`, that is a warning, not a revert.

### D4: Three phases; every target is known before the first edit

The phases follow contract §5.2 rule 5.

**Phase A, resolve.** No edits. Calls, in this order:

1. `check-files --repo-root .`
2. `newest go github.com/open-platform-model/library --current <go.mod> --repo-root .`
3. `newest release opm-operator --asset install.yaml --current <manifest.go> --repo-root .`
4. `newest cue opmodel.dev/catalogs/opm@v4 --current <representative> --repo-root .`
5. `hold opmodel.dev/core@v2 --repo-root .`
6. `pin-of opmodel.dev/catalogs/opm@v4 <C> opmodel.dev/core@v2`, once per distinct `C` (D3).
7. `is-frozen <file> <key>` for every (file, key) pair whose target differs from the file's value, and for `hack/kind-platform.yaml`. A frozen pair drops out.
8. For each version-advance module that will differ from the merge base (D6): `published cue <module>@vN v<B>`.
9. `language-of <module> <target>`, for the catalog and for core when either moves (D8).

`newest` gets `--expect <v>` when `CASCADE_EXPECT` names its key (contract §5.4). Each exit code is handled by a `case`:

- 0 means move;
- 3 means stay;
- any other code exits the task with that code.

No `|| true`, no `2>/dev/null ||` and no `set +e` around a resolver call.

**Phase B, tools.** Only when something moves:

```bash
go build -o "$STATE/bin/opm" ./cmd/opm
```

This builds from the unmodified tree, so a library move that breaks compilation cannot stop the task from producing its diff (contract §5.2 rule 11). The binary lives in `$(git rev-parse --git-dir)/cascade/bin`, never under the tree's `bin/`.

**Phase C, edit.** The order follows contract §5.2 rule 12 and §6.4:

1. Shipped pins:
   - **library:** `go get github.com/open-platform-model/library@<v>` and `go mod tidy`. Any other `go.mod` require line that changed is warned under the library key.
   - **operator:** `task operator:sync VERSION=<v>` (`Taskfile.yml:497-518`). Its `curl -sfL ... -o` writes nothing on a 404, so a draft that appears between phases A and C fails the task loudly.
   - **the three templates** (D3).
2. Test pins (D3), then `hack/kind-platform.yaml`.
3. Version advances (D6).
4. Podinfo consumers (D7).
5. Docs-bundle warnings (D9).

The cli has no `.opm-cli-version`, so rule 12's "`.opm-cli-version` last" does not apply.

The contract lists `operator:sync` among the regenerators that rule 12 places last, while §6.4 moves it second. The two never touch the same files, so the order changes nothing; the task follows §6.4.

**Result** (contract §5.2 rule 13): exit 0 when `git status --porcelain --untracked-files=all` is non-empty. Under `CASCADE_ALLOW_DIRTY=1`, it is exit 0 when the snapshot differs from the start. Otherwise it is exit 3.

**Setup** (contract §5.2 rules 1 to 4):

- **Clean start.** A dirty tree without `CASCADE_ALLOW_DIRTY=1` is exit 1.
- **State.** `STATE=$(git rev-parse --git-dir)/cascade`, with `$STATE/warnings` truncated. `CASCADE_WARNINGS` is exported.
- **Registries.** `CUE_REGISTRY` and `OPM_REGISTRY` are exported as `testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`. The task never inherits the `Unit Tests` job's `localhost:5000` mapping (`.github/workflows/pr.yml:66-67`).

### D5: Never touched

- `.cascade-frozen` and `.cascade-hold`.
- The `release-please` files and `CHANGELOG.md`.
- Anything under `.github/`.
- `.opm-docs-version`, `docs-kit.cue`.
- `cue-versions.yml`, if one ever exists.
- Every `language.version`.
- `hack/fixtures.sh` and `tests/fixtures/fixtures.go`.
- The frozen Go literals.
- The task never runs `publish` or `seed` against a real registry (contract §5.2 rules 14 and 15).

### D6: Version advance once per PR (templates and podinfo)

Modules:

| Module | Identity file | Coordinate |
| --- | --- | --- |
| `templates/minimal` | `templates/minimal/identity/identity.cue` | `opmodel.dev/templates/minimal@v1` |
| `templates/standard` | `templates/standard/identity/identity.cue` | `opmodel.dev/templates/standard@v1` |
| `templates/advanced` | `templates/advanced/identity/identity.cue` | `opmodel.dev/templates/advanced@v1` |
| `tests/fixtures/modules/podinfo` | `tests/fixtures/modules/podinfo/identity/identity.cue` | `testing.opmodel.dev/modules/cli/podinfo@v0` |

Per contract §5.2 rule 11:

- `M` = `git merge-base "${CASCADE_BASE:-origin/main}" HEAD`.
- `B` = the bare `^Version:` value of the identity file at `M`.
- `f_changed` is copied verbatim from the contract.
- The target is:
  - `B` when the module is unchanged;
  - `next-patch(v<B>)` when it changed and `published cue <coord> v<B>` answers 0;
  - `B` when it changed and `B` is not yet published.
- It is written bare with `"$STATE/bin/opm" module version set <ver> <dir>` (`internal/cmd/module/version.go:31`, `set <version> [path]`), only when it differs from the file.

In phase A the task predicts "will differ from `M`" as `f_changed` now, or any pin in that module moving. So `published` is called only when needed, and a no-op run (S1) makes no `published` call.

The publish gate (`.github/scripts/publish-templates.sh`; `AGENTS.md:128`) refuses a changed template that keeps a published version. This rule satisfies it on the first run and leaves the version alone on later runs.

### D7: The podinfo consumers follow in the same PR

**Trap T3 is closed.** `TestResolveInstanceArg_RegistryBackedInstancePackage` now resolves through `OPM_REGISTRY` when it is set (`internal/cmdutil/instance_arg_test.go:160-177`). PR CI sets it to the job-local registry seeded from the tree (`.github/workflows/pr.yml:57-89`). The memory note that says the test uses `config.DefaultRegistry` predates that fix.

So `examples/cue.mod/module.cue:15-16` and `tests/e2e/testdata/operator-owned/cue.mod/module.cue:15-16` follow the podinfo target in the same PR (contract §6.4 step 5 and §9.6):

1. Their catalog and core move through D3 (`cue mod get` and `tidy`) while podinfo is still at its published version.
2. Then the `v:` inside their `"testing.opmodel.dev/modules/cli/podinfo@v0": {` block is rewritten as text to the D6 target, only if it differs.
3. No `cue mod get` or `tidy` runs in a consumer after that rewrite, because the new podinfo is not published until merge.

PR CI's `hack/fixtures.sh consumers examples tests/e2e/testdata/operator-owned` (`Taskfile.yml:144`, fixtures job) checks the result against the seeded registry.

A local `go test` without a seeded `OPM_REGISTRY` resolves podinfo from GHCR and fails on the new version until publish. This affects only a human running the gates on a cascade PR, not this change. Use `task test:fixtures`, which seeds.

### D8: `language.version` is compared with the CUE pinned in `pr.yml`

The cli sets no `CUE_VERSION` env. Its required PR CI installs `cuelang.org/go/cmd/cue@v0.17.1` in the `Unit Tests` job (`.github/workflows/pr.yml:81`), and the same literal appears at `:187` and `:268`.

- The task reads the first `cuelang.org/go/cmd/cue@v[0-9.]+` in `.github/workflows/pr.yml`.
- For each moved CUE upstream (catalog `T`, core targets) it compares `language-of` with that version through `semver-cmp`, never with the `cue` on `PATH` (contract §5.2 rule 10).
- Newer upstream: warn.
- Exit 3 from `language-of`: no warning.
- Unreadable pin: warn with key `-`.

### D9: Docs bundles are a warning

This follows contract §6.4 step 6 and §9.5. When library or the operator moved:

- The task runs `go run ./hack/docskit-dump pins` on the edited tree. It prints JSON whose `.pins` maps `library`, `core` and `opm-operator` to bare versions (`hack/docskit-dump/main.go:54-78`).
- For each entry it calls `published oci open-platform-model/docs/<project> <pin>`.
- On exit 3 it warns: "docs bundle for `<project>` `<pin>` is not published; G1 will fail the next release PR until it is".
- If `docskit-dump` fails to build, it warns with key `-` and continues.

These calls decide no target, so they run in phase C.

### D10: The scenario test (contract §8)

`.tasks/cascade/test.sh`:

- **Pre-checks.**
  - The stub's sha256 equals `970130f7d55c07f5b86d4f5b6f392330427ff923eb34f93553656bcd4b893d9c`.
  - `pins.sh WORKTREE` equals `pins.sh HEAD` in a clean sandbox.
- **Sandbox.** It copies the tree into `$(mktemp -d)/r` with `git ls-files ... | tar` and commits it as `base`. It sets `CASCADE_BASE` to the setup SHA, and `CASCADE_RESOLVER` to the absolute stub path, along with `CASCADE_STUB_TABLE`, `CASCADE_STUB_LOG` and `CASCADE_TODAY=2026-10-03`. Cleanup is by `trap`.
- **Stub tables.** These are built at test time from `pins.sh WORKTREE`, plus:
  - one `pin-of` row for the tree catalog;
  - one `published` row per advance module at its tree version.
- **`testdata/older.tsv`**, checked live 2026-10-04:

  | Key | Older version |
  | --- | --- |
  | library | `v1.0.0-beta.2` |
  | opm-operator | `v1.0.0-beta.4` (its `install.yaml` downloads) |
  | `catalogs/opm@v4` | `v4.4.3` |
  | `core@v2` | `v2.0.0-alpha.13` |

  Plus the row `pin-of opmodel.dev/catalogs/opm@v4 v4.4.3 opmodel.dev/core@v2 v2.0.0-alpha.13`. The test asserts each is older than the tree's value.
- **S1, no-op.**
  - Exit 3 and a clean tree.
  - The normalized stub log equals `testdata/s1-calls.txt`: `check-files`, `newest go`, `newest release`, `newest cue` (catalog), `hold`, and one `pin-of`.
- **S3, error.** The library `newest` row is `ERROR`. The exit is neither 0 nor 3, and the tree stays clean.
- **S6, dirty tree.** With an untracked file, the exit is 1.
- **S2, older pins.**
  - **Setup:**
    - `go get library@v1.0.0-beta.2` then `go mod tidy`;
    - `task operator:sync VERSION=v1.0.0-beta.4`;
    - a text edit of the catalog and core `v:` lines in every D3 file, and of `hack/kind-platform.yaml`, to the older values.
    - The `cue.mod` files are left untidy, which `cue mod get` repairs.
  - **First run:** exit 0. The diff against the original tree is exactly this golden list:
    - `templates/{minimal,standard,advanced}/identity/identity.cue` at `1.0.4`;
    - `tests/fixtures/modules/podinfo/identity/identity.cue` at `0.1.12`;
    - `examples/cue.mod/module.cue` and `tests/e2e/testdata/operator-owned/cue.mod/module.cue`, with podinfo at `v0.1.12`.
  - **Second run:** commit, keep `CASCADE_BASE`, run again. Exit 3, and no identity file changes.
  - The version numbers are derived from the tree at test time, not hard-coded.
- **S4, frozen.**
  - The S2 setup, plus an appended `.cascade-frozen` entry for `tests/integration/module-apply/testdata/cue.mod/module.cue` with both OPM keys.
  - That file has catalog and core, no consumers and no version advance, so the freeze isolates one file.
  - That file stays byte-unchanged, and the exit is 0.
- **S5, title and body.** Runs only with `CASCADE_RESOLVER_REAL` set, after S2.
  - The title is `fix(deps): bump 4 upstream pins`.
  - The body has both markers, four table rows and `## Notes` last.
  - It has no `need-human-review` label.
- **Sets.** `CASCADE_TEST_SET=offline` runs the pre-checks plus S1, S3 and S6. `all` (the default) runs everything.

### D11: CI placement

- **Required, offline.** In `pr.yml` job `unit` (name `Unit Tests`), add these steps after `setup-go`:
  - `go-task/setup-task@a00fbb05ce67b35648be3c78cbc9fd85354c757e # v2.2.0`, the pin `pr.yml:99` already uses;
  - `task -x deps:cascade:test`, with `CASCADE_TEST_SET=offline` and `CASCADE_RESOLVER=${{ github.workspace }}/.tasks/cascade/testdata/stub-resolve.sh`.

  The offline set needs no resolver checkout. Setting `CASCADE_RESOLVER` to the stub satisfies the contract §3 precondition, which every cascade task carries. `ubuntu-latest` ships mikefarah `yq` for the stub's `is-frozen`.
- **Not required, network.** New `.github/workflows/cascade-task.yml`:
  - job `Cascade task (network)`, `timeout-minutes: 20`, `permissions: contents: read`;
  - triggers: `pull_request` on `.tasks/cascade/**`, `Taskfile.yml`, `.tasks/*.yaml` and the workflow itself, plus `workflow_dispatch` and a weekly `schedule`;
  - it checks out `open-platform-model/.github` at `main` into `org-github` with `persist-credentials: false`;
  - it sets `CASCADE_RESOLVER_REAL`, so S5 runs, and installs Go 1.26.0 and `cue` v0.17.1, as `pr.yml` does;
  - every `uses:` is SHA-pinned to the pins `pr.yml` already uses.

## Research & Decisions

### Where the consistent set gets its catalog for core-only files

**Context**: `tests/fixtures/valid/*` pin core but no catalog.

**Explored**: contract §5.2 rule 7. RELEASING.md, "What each repo's task moves", lists both files.

**Options considered**:

1. `newest cue opmodel.dev/core@v2`. This is simple, but it breaks the consistent set: core would be `v2.0.0-beta.2` while every catalog pins `v2.0.0-beta.1`.
2. `pin-of` the representative catalog. All trees stay on one core.

**Decision**: option 2.

**Rationale**: the contract mandates it, and one core per tree keeps the render fixtures comparable.

### The `test-fixture-lineage` requirement versus the consistent set

**Context**: the main spec `openspec/specs/test-fixture-lineage/spec.md` requirement "Old test pins are current or frozen with a reason" says each core or catalog test pin "SHALL be moved to the newest published release". Its first scenario fails any older pin outside `.cascade-frozen`. Under the consistent set, core in test trees is `v2.0.0-beta.1` while core `v2.0.0-beta.2` is published, because catalog `v4.5.1` pins `beta.1`. So the requirement is broken permanently, by design.

**Options considered**:

1. Leave the spec. It would then contradict the task that implements it.
2. Restate the requirement as the cascade target (MODIFIED, every scenario kept by name).

**Decision**: option 2. RELEASING.md "The cascade" › "The receiver" wins over a cli main spec.

**Rationale**: this is reported to the supervisor as a spec drift, not a RELEASING.md conflict.

### Older versions for S2

**Context**: S2 needs real published versions older than the tree.

**Explored**: GHCR tags for `catalogs/opm` and the modulefile core pins of `v4.4.3`, `v4.4.4` and `v4.5.1`; the proxy list for library; the operator release asset. All were checked live on 2026-10-04.

**Decision**: library `v1.0.0-beta.2`, opm-operator `v1.0.0-beta.4`, catalog `v4.4.3`, core `v2.0.0-alpha.13`.

**Rationale**:

- The cli built against library `v1.0.0-beta.1` until cli PR 291 moved it to `beta.3` with no Go edits (`bd1efe15` touches only `go.mod`, `go.sum` and the operator embed). So phase B can build the S2 setup tree.
- `v4.4.3` pins `alpha.13`, which gives the `pin-of` row.

### Placing `published` and `is-frozen` in phase A

**Context**: contract §5.2 rule 5 requires every target-deciding call to run before the first edit. Rule 11's `f_changed`, however, is defined on the edited tree.

**Options considered**:

1. Call `published` for every advance module, always. This puts four extra calls into S1.
2. Predict the change in phase A: `f_changed` now, or a pin in that module moving.

**Decision**: option 2.

**Rationale**: the prediction is exact. The task edits nothing in a module where no pin moves, and S1 stays the minimal call list the contract describes.

## Risks / Trade-offs

- **[Catalog `v4.5.x` breaks a template]** The first real run (the catch-up PR) is the first time templates render against `v4.5.1`. → The PR's CI shows it, and `.cascade-hold` or a human commit handles it (RELEASING.md "Runbook"). This change's tests use the stub and real older versions, never `v4.5.x`.
- **[A template's declared version is below its highest published one]** `next-patch(B)` could then collide with a published version. → `publish-templates.sh` refuses it in the PR's `template-gates` job. This does not happen today: every template declares its newest published `1.0.3`.
- **[The network job is not required]** A regression only the network set catches can merge. → The job runs on every PR that touches the cascade files and weekly. The supervisor checks it before merging cascade-touching PRs (contract §9.12).
- **[Proxy or GHCR lag]** This belongs to the resolver (`--expect`, contract §2.9). The task only forwards `CASCADE_EXPECT`.
- **[`go mod tidy` raises a third-party module]** It is warned, not reverted (contract §5.2 rule 6).

## Open Questions

- **Core versus catalog ranking.** Contract §9.10 has the owner choose whether core may be lowered to a catalog's pin. This design keeps never-backwards, as the contract does. If the owner reverses it, only D3's "core is greater" branch changes.
- **Docs-pin mismatch.** The cli's docs pins report core `2.0.0-beta.2` (library's `DefaultSchemaModule`), while the templates ship `v2.0.0-beta.1` (catalog `v4.5.1`'s pin). No gate compares the two. Should the body warn about it? The contract does not ask for it, so this change does not add it.
