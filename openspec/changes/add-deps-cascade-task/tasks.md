# Tasks: add-deps-cascade-task

One PR, titled `ci(cascade): add the deps:cascade tasks` (Phase 2 cascade contract §10). Workers never tag, never create, edit or delete a GitHub Release or label by hand, and never touch rulesets or repo settings. They never create, delete or apply to a Kubernetes cluster. Items marked **SUPERVISOR** are not worker tasks.

The binding interface is the Phase 2 cascade contract (`p2-cascade-contract.md` in the supervisor's scratchpad, cited as "contract §N"), together with workspace RELEASING.md, sections "The cascade", "What each repo's task moves", "Gates", "Cascade files" and "Rollout and changes". Where the two disagree, stop and report the conflict to the supervisor. Do not pick one.

Environment for every Go and CUE command, exported on two lines:

```bash
export CUE_REGISTRY='testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

**The resolver.** Until `.github` `add-cascade-resolver` is merged and checked out beside this repo, export `CASCADE_RESOLVER` as the absolute path of this worktree's `.tasks/cascade/testdata/stub-resolve.sh` (contract §3). This satisfies the precondition every cascade task carries.

**The local test gate** is `task test:unit`, `task openspec:check`, `shellcheck .tasks/cascade/*.sh .tasks/cascade/testdata/stub-resolve.sh` and, from section 2 on, `CASCADE_TEST_SET=offline task -x deps:cascade:test`.

- `task test` also runs `test:integration` and `test:e2e`, which need a cluster. This change adds no Go code, so they are reported as not run.
- Always run tasks with `task -x` when the exit code matters (contract §3).

## Gates (SUPERVISOR ticks)

- [ ] G-resolver: before merge, `.github` `add-cascade-resolver` is merged on `open-platform-model/.github` `main`. Check: `git -C <ws>/.github fetch && git -C <ws>/.github cat-file -e origin/main:.github/scripts/cascade/cascade-resolve.sh && git -C <ws>/.github show origin/main:.github/scripts/cascade/stub-resolve.sh | sha256sum` succeeds and prints contract §7's `970130f7d55c07f5b86d4f5b6f392330427ff923eb34f93553656bcd4b893d9c`.
- [ ] G-network-run: before merge, on this change's PR, `Cascade task (network)` is green, and its log shows S5 ran against the real resolver (not skipped). Also, `Lint` shows the offline cascade step passing.
- [ ] G-catch-up: after merge, not a merge precondition. The cli catch-up PR (contract §8: `task -x deps:cascade` on `main`, titled by `task -x deps:cascade:title`) is a separate PR. It waits until the operator's catch-up release is published. After it merges, `task -x deps:cascade` on `main` exits 3. That is the Phase 2 gate (workspace RELEASING.md, "Rollout and changes" › "Phases").

## 1. Spike: confirm the assumptions in design.md

Every check runs in scratch copies under `$(git rev-parse --git-dir)/cascade-spike/`, never in the tree. Record each result as an indented "Done:" note, then write a short "Spike findings" subsection at the end of design.md's "Research & Decisions".

- [x] 1.1 `task --version`. In a scratch Taskfile with one task running `exit 3`, confirm that `task -x` exits 3 and plain `task` exits 201 (contract §3).
  Done: task 3.52.0; `task -x` exits 3, plain `task` exits 201.
- [x] 1.2 Copy `templates/minimal` to scratch and run `cue mod get opmodel.dev/catalogs/opm@v4.5.1 opmodel.dev/core@v2.0.0-beta.1 && cue mod tidy`. Record the diff. Confirm it names no third-party key and leaves `language.version` alone.
  Done: the only diff is the catalog `v:` (`v4.4.4` -> `v4.5.1`); no third-party key, `language.version` unchanged.
- [x] 1.3 In that scratch copy, run `go build -o <scratch>/opm ./cmd/opm`, then `<scratch>/opm module version set 1.0.4 <scratch copy>`. Confirm it rewrites only the `Version:` line of `identity/identity.cue` and needs no registry.
  Done: only `Version:` in `identity/identity.cue` changes (1.0.3 -> 1.0.4), with the registry env pointed at an unreachable host; the log line goes to stderr.
- [x] 1.4 The D10 S2 assumption, for every D3 file. Copy each of the thirteen D3 directories to scratch. Text-edit their catalog and core `v:` to `v4.4.3` and `v2.0.0-alpha.13` (core only in the two core-only files), then run `cue mod get` of the keys each file has at `v4.4.4` and `v2.0.0-beta.1`, and `cue mod tidy`. Each `module.cue` must come back byte-identical to the committed file (`internal/instinit/testdata/initvalues` carries `default: true` with an aligned `v:`). Also, in a scratch copy of the repo: `go get github.com/open-platform-model/library@v1.0.0-beta.2 && go mod tidy`, `go build ./cmd/opm`, then `go get ...@v1.0.0-beta.3 && go mod tidy`; `go.mod` and `go.sum` must come back byte-identical. Record any difference: it decides the S2 golden list.
  Done: all thirteen D3 `module.cue` files come back byte-identical, `initvalues`' `default: true` included; the Go round trip gives byte-identical `go.mod` and `go.sum`, and `./cmd/opm` builds at library `v1.0.0-beta.2`. The S2 golden list stands as written.
- [x] 1.5 Confirm `testdata/older.tsv`'s versions are published. Use these anonymous checks, the same ones the contract §2.4 kinds make:
  - library `v1.0.0-beta.2`: `proxy.golang.org/.../@v/v1.0.0-beta.2.info` answers 200;
  - opm-operator `v1.0.0-beta.4`: the `install.yaml` download `HEAD -L` answers 200;
  - catalog `v4.4.3`: the GHCR manifest `HEAD` answers 200;
  - the catalog `v4.4.3` modulefile pins core `v2.0.0-alpha.13`.
  Done: library `v1.0.0-beta.2` `.info` 200; opm-operator `v1.0.0-beta.4` `install.yaml` 200; catalog `v4.4.3` and `v4.4.2` published; `v4.4.3` pins core `v2.0.0-alpha.13`, `v4.4.2` pins `v2.0.0-alpha.12`; podinfo `v0.1.10` is published and pins catalog `v4.0.1` and core `v2.0.0-alpha.6`; podinfo `v0.1.12` is not published.
- [x] 1.6 Run `go run ./hack/docskit-dump pins` and record its JSON shape (`.pins` keys and bare values) for D9.
  Done: `{"schema": "docs.opmodel.dev/pins/v1", "pins": {"core": "2.0.0-beta.2", "library": "1.0.0-beta.3", "opm-operator": "1.0.0-beta.5"}}`; values bare.
- [x] 1.7 `task openspec:check` green, then commit `docs(openspec): record the add-deps-cascade-task spike findings`. The commit touches only `openspec/changes/add-deps-cascade-task/`.
  Done: `task openspec:check` green.

## 2. Pins, classes, stub, and the title and body tasks

- [x] 2.1 Create `.tasks/cascade/testdata/stub-resolve.sh` (mode 0755). Extract the fenced block of contract §7 byte for byte. Verify with `sha256sum`: it must equal `970130f7d55c07f5b86d4f5b6f392330427ff923eb34f93553656bcd4b893d9c`. Never edit it.
  Done: sha256 matches.
- [x] 2.2 Create `.tasks/cascade/classes` with exactly the cli block of contract §5.3, plus a one-line `#` header citing workspace RELEASING.md "Title from diff class".
- [x] 2.3 Create `.tasks/cascade/pins.sh <ref>` (executable, `set -euo pipefail`, following design.md D2).
  - Read with `cat` for `WORKTREE`, and with `git show <ref>:<path>` otherwise.
  - Print `key<TAB>display<TAB>shipped<TAB>v...<TAB>` rows in the D2 order. Omit a pin missing at the ref.
  - Exit 1 on a bad argument.
  - Verify: `.tasks/cascade/pins.sh WORKTREE` prints four rows matching `go.mod:14`, `internal/operator/manifest.go:19` and `templates/minimal/cue.mod/module.cue:10,13`. `pins.sh HEAD` prints the same.
  Done: four rows; `WORKTREE` and `HEAD` agree; a bad ref or no argument exits 1.
- [x] 2.4 `Taskfile.yml`, after `deps:release-check`: add `deps:cascade:title`, `deps:cascade:body` and `deps:cascade:test`.
  - Each carries the task-level `CASCADE_RESOLVER_PATH` var through one YAML anchor (contract §3, design.md D1), `env: CASCADE_RESOLVER` and the `test -x` precondition with the contract §3 message.
  - `desc` lines say each task is part of the release cascade and cite workspace RELEASING.md "The cascade".
  - Do not add `deps:cascade` yet; section 3 adds it.
- [x] 2.5 Create `.tasks/cascade/test.sh` with the shared sandbox helper, the `CASCADE_TEST_SET` switch and the two pre-checks from design.md D10: the stub checksum, and `pins.sh WORKTREE` equal to `pins.sh HEAD` in a sandbox. It prints `PASS`/`FAIL` lines and exits 0 or 1. Scenarios come in sections 3 and 4.
  - Verify `CASCADE_TEST_SET=offline task -x deps:cascade:test` exits 0, and exits 1 after a throwaway one-byte edit of the stub (restore it).
  Done: offline set exits 0; a one-byte stub edit makes it exit 1. Sandbox git runs with auto gc off, after one cleanup race on the sandbox's `.git/objects`.
- [x] 2.6 Verify the precondition. With `CASCADE_RESOLVER=relative/path`, `task -x deps:cascade:title` fails with "must be absolute". With an absolute non-existent path, it fails with the contract §3 message.
  Done: a relative path stops the var with "must be absolute"; an absolute missing path fails the precondition with the contract §3 message.
- [x] 2.7 `task fmt`, `task lint`, `task test:unit`, `task openspec:check`, shellcheck, and the offline test set green, then commit `ci(cascade): add the cascade pins, classes and title and body tasks`.
  Done: all green.

## 3. `task deps:cascade` and the offline scenarios

- [x] 3.1 Create `.tasks/cascade/cascade.sh` (executable, `set -euo pipefail`). It covers:
  - the setup: clean-tree check with `CASCADE_ALLOW_DIRTY` and `snapshot`, `$STATE`, `CASCADE_WARNINGS`, the registry exports, and `check-files` (design.md D4; contract §5.2 rules 1 to 4);
  - `f_changed`, copied verbatim from contract §5.2 rule 11;
  - `CASCADE_EXPECT` parsing (contract §5.4).

  No `|| true`, no `2>/dev/null ||` and no `set +e` around resolver or tool calls.
- [x] 3.2 Phase A in `cascade.sh` (design.md D4): the nine call groups in order, each exit handled by a `case` on 0, 3 or other. Implement the consistent set across the D3 file list (an explicit list, never a glob), including:
  - the core-only files;
  - the hold rule;
  - the "core ahead" warning;
  - the phase A prediction for the version advance (D6).
  Done: also caught in S1 during the build: a core-only file must never get a catalog move.
- [x] 3.3 Phases B and C in `cascade.sh`:
  - `go build -o "$STATE/bin/opm" ./cmd/opm`, only if something moves;
  - library `go get` and `go mod tidy`, with the third-party require warning;
  - `task operator:sync VERSION=<v>`;
  - per-file `cue mod get` of the moved `opmodel.dev` keys at exact versions, then `cue mod tidy`, and the frozen-key check after tidy (exit 1 naming file and key);
  - the `cue.dev/x/k8s.io@v0` warning;
  - the `hack/kind-platform.yaml` text rewrite;
  - version advances through `"$STATE/bin/opm" module version set`;
  - the podinfo consumer text rewrite (D7);
  - `language-of` warnings against `.github/workflows/pr.yml`'s `cue@v...` (D8);
  - docs-bundle warnings (D9);
  - the exit 0 or 3 result (contract §5.2 rule 13).
- [x] 3.4 `Taskfile.yml`: add `deps:cascade` running `.tasks/cascade/cascade.sh`, with the same anchor, env and precondition as section 2. Its `desc` names the exit codes and `task -x`.
- [x] 3.5 Create `.tasks/cascade/testdata/s1-calls.txt`: the normalized S1 call list from design.md D10 (`check-files`, `newest go`, `newest release`, `newest cue` for the catalog, `hold`, one `pin-of`).
- [x] 3.6 Add the offline scenarios to `test.sh`: S1 no-op, S3 error and S6 dirty tree, exactly as contract §8 and design.md D10 describe, plus S7 and S8.
  - Build the stub table from `pins.sh WORKTREE` at test time; nothing current is committed.
  - Assert that every `newest` line in the log carries `--current` and `--repo-root`.
  - Assert that the real checkout's `git status --porcelain` is unchanged after the run.
  - S3 also asserts that the normalized stub log ends at the `newest go` call and that `$STATE/bin/opm` does not exist.
  - Add S7 (core ahead) and S8 (a hold on core holds the catalog) from design.md D10 to the offline set.
- [x] 3.7 Verify on the tree:
  - `CASCADE_TEST_SET=offline task -x deps:cascade:test` exits 0 with `PASS` for the pre-checks and S1, S3, S6, S7 and S8;
  - a throwaway `|| true` added after the library `newest` call in `cascade.sh` makes S3 fail (restore it).
  Done: offline set all PASS. Every resolver call goes through the helper `r`, which exits on any code but 0 or 3, so a literal `|| true` after the library call cannot swallow the error (S3 still passes, correctly). The equivalent swallow, the helper's error branch turned into "stay", makes S3 fail (`exit 3, want neither 0 nor 3`). Both restored.
- [x] 3.8 `task fmt`, `task lint`, `task test:unit`, `task openspec:check`, shellcheck, and the offline test set green, then commit `ci(cascade): add task deps:cascade`.
  Done: all green.

## 4. Network scenarios, CI and docs

- [ ] 4.1 Create `.tasks/cascade/testdata/older.tsv` with the `older` and `oldest` rows and the two `pin-of` rows from design.md D10, as confirmed in 1.5. `test.sh` asserts each is strictly older than the tree's value with the stub's `semver-cmp`, failing with the contract §8 message otherwise.
- [ ] 4.2 Add S2 (older pins) to `test.sh`.
  - Setup:
    - `go get github.com/open-platform-model/library@<older>` and `go mod tidy`;
    - `task operator:sync VERSION=<older>`;
    - a text edit of the catalog and core `v:` in every D3 file and of `hack/kind-platform.yaml`.
  - First run: exit 0. The diff against the original tree equals the golden list from design.md D10, as adjusted by 1.4. Its versions are derived from the tree's identity files.
  - The warnings file holds the docs-bundle warning.
  - Second run: `git add -A && git commit`, keep `CASCADE_BASE`, run again. Exit 3, and every identity file is unchanged.
- [ ] 4.3 Add S4 (frozen) to `test.sh`. Run the S2 setup, then append an entry to the sandbox copy's `.cascade-frozen`, keeping its real entries, for `tests/integration/module-apply/testdata/cue.mod/module.cue` with both `opmodel.dev` keys and a reason. Assert that file is byte-unchanged, `tests/e2e/testdata/duplicate-identities/cue.mod/module.cue` moved, and the exit is 0.
- [ ] 4.3a Add S9 (a second move on the branch) to `test.sh` per design.md D10, and the D7 second-run rule it backs, if section 3 did not already implement it.
- [ ] 4.4 Add S5 (title and body) to `test.sh`. `CASCADE_RESOLVER_REAL` holds the absolute path of the real `cascade-resolve.sh`; S5 runs only when it is set, and is skipped with a `SKIP` line otherwise. It asserts:
  - the title `fix(deps): bump 4 upstream pins`;
  - both markers, four table rows, `## Notes` last, and no `need-human-review`.
- [ ] 4.5 Verify `task -x deps:cascade:test` (the full set, with network) exits 0 locally, with `CASCADE_RESOLVER_REAL` set to the real resolver's absolute path (beside the repo, or the `add-cascade-resolver` worktree until it merges), so S5 runs.
- [ ] 4.6 `.github/workflows/pr.yml`, job `lint` (`Lint`): after `setup-go`, before `golangci-lint`, add `go-task/setup-task@a00fbb05ce67b35648be3c78cbc9fd85354c757e # v2.2.0` (`version: 3.x`) and a step `Cascade task (offline)`.
  - The step runs `task -x deps:cascade:test` with `CASCADE_TEST_SET: offline` and `CASCADE_RESOLVER: ${{ github.workspace }}/.tasks/cascade/testdata/stub-resolve.sh` (design.md D11).
  - Add a short comment citing workspace RELEASING.md "Rollout and changes" and "Rulesets on main".
- [ ] 4.7 Create `.github/workflows/cascade-task.yml` per design.md D11:
  - job `Cascade task (network)`, `timeout-minutes: 20`, `permissions: contents: read`;
  - the path-filtered `pull_request`, `workflow_dispatch` and weekly `schedule` triggers;
  - checkout of `open-platform-model/.github` at `main` into `org-github` with `persist-credentials: false`;
  - `CASCADE_RESOLVER_REAL` and `CASCADE_RESOLVER` set to `$GITHUB_WORKSPACE/org-github/.github/scripts/cascade/cascade-resolve.sh`;
  - Go 1.26.0, `cue` v0.17.1 and setup-task, all SHA-pinned as in `pr.yml`;
  - `task -x deps:cascade:test`.

  Verify both workflows with `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/pr.yml .github/workflows/cascade-task.yml`.
- [ ] 4.8 `AGENTS.md`:
  - at `AGENTS.md:128` (templates) and `AGENTS.md:231` (fixtures), say that `task deps:cascade` moves the template and fixture pins and advances their versions once per PR, replacing the workspace root task names there;
  - add one bullet next to the release-pin bullet (`AGENTS.md:356`) naming the four cascade tasks, `task -x`, and workspace RELEASING.md "The cascade";
  - say: do not run the workspace root `deps:update`, `deps:update:templates` or `deps:pins:*` against the cli; Phase 5 rewires them (design.md Risks).
- [ ] 4.9 `task fmt`, `task lint`, `task test:unit`, `task openspec:check`, shellcheck, the offline test set, and the full set from 4.5 green, then commit `ci(cascade): test the cascade task in CI`.

## 5. Archive (rides this PR)

The archive rides the implementing PR, never a push to main (owner decision 4). This section runs only after the supervisor's review of sections 1 to 4, and never in the planning run.

- [ ] 5.1 `openspec verify` for this change; record its result.
- [ ] 5.2 `openspec archive add-deps-cascade-task --yes`. This creates `openspec/specs/deps-cascade/spec.md` with its Purpose, and restates "Old test pins are current or frozen with a reason" in `openspec/specs/test-fixture-lineage/spec.md`, keeping its three scenarios. Verify: `task openspec:check` is green.
- [ ] 5.3 `task openspec:check` green, then commit `chore(openspec): archive add-deps-cascade-task`. The commit touches only `openspec/`.
