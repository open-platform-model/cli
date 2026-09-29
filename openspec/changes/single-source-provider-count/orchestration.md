# Orchestration: single-source-provider-count

This file is the brief for the worker agent that implements this change and for the supervisor that stitches the change set together.

## Change set

Four OpenSpec changes, planned together on 2026-09-29, each in its own repo. Together they make the render build's provider count the only count: core computes it once as `#contracts.providedBy`, the library render reads it instead of its own guard, and the operator and cli name the providing catalogs from it. No enhancement entry backs them (they correct the delivery of 0015:D2/D18, and 0015 is being closed as delivered), so none carries an `enhancement.yaml`.

| ID | Repo | Change | Branch | Wave | Starts when | Merges when |
| --- | --- | --- | --- | --- | --- | --- |
| A | core | `count-providers-per-registry-entry` | `feat/count-providers-per-registry-entry` | 1 | now | first; release cut after |
| B | library | `read-provider-count-from-core` | `feat/read-provider-count-from-core` | 2 | A released | after A is released |
| C | opm-operator | `single-source-provider-count` | `fix/single-source-provider-count` | 3 | B's branch pushed | after B is released |
| D | cli | `single-source-provider-count` | `fix/single-source-provider-count` | 3 | B's branch pushed | after B is released and the cli `deps:update` commit is on main |

What each hands on:

- **A** publishes `#ContractInventory.providedBy: [#ContractFQNType]: [...#ModulePathType]` (sorted registry keys), recounted `overSubscribed`/`unfulfilled`, in core 2.0.0-alpha.12 (actual tag reported under `surface`).
- **B** reads it in the render build (single source), decodes it, floors core, and publishes the interface below in a library release.
- **C** and **D** consume B's release.

## Interface B → C, D

The contract C and D consume. B may refine names only by reporting the change under `deviations`; C and D code against what B reports under `surface`.

```go
package platform // github.com/open-platform-model/library/opm/platform

type ContractInventory struct {
	// ...existing fields unchanged...

	// ProvidedBy maps every provider-fulfilled contract FQN some enabled
	// transformer requires (defined by an enabled catalog or not) to the
	// sorted registry keys (path@major) of the enabled entries whose
	// transformers require it. OverSubscribed is exactly its keys with two
	// or more entries; a key a defined provider contract lacks is Unfulfilled.
	ProvidedBy map[string][]string `json:"providedBy"`
}

// Contracts() refuses a platform whose #contracts lacks providedBy, naming
// the field and core 2.0.0-alpha.12.
```

```go
package errors // github.com/open-platform-model/library/opm/errors

// Returned (wrapped) by Kernel.Render before staging, and by Contracts(),
// when the platform module pins a core release predating a field the kernel reads.
type PlatformCoreTooOldError struct { Platform, Field, Since string }
```

`schema.DefaultSchemaModule` = `opmodel.dev/core@v2.0.0-alpha.12`. `OverSubscribedContract{Key, Catalogs}` and the render refusal text are unchanged.

The rule behind it (A computes it, B reads it, C and D print it):

1. Which transformers count: every transformer of every enabled registry entry, iterated per entry. Only required demands (`requiredResources`, `requiredTraits`) count; optional demands and `requiredLabels` never do.
2. Fulfilment is read from the transformer's own requirement (`req.fulfilment == "provider"`), never from the defining catalog's member; counting never depends on whether an enabled entry defines the contract.
3. The key is the registry key, path with major. Two adapters in one entry are one provider; two majors of one catalog are two; two entries are two whether or not any enabled entry defines the contract.
4. `overSubscribed` is every `providedBy` key with more than one entry. `unfulfilled` is every defined provider-fulfilled resource or trait with no `providedBy` key; a contract no enabled catalog defines is never unfulfilled.
5. The render reads the same values: `match.#providers = platform.#contracts.providedBy`; the over-subscription rows are `{key, catalogs: providedBy[key]}` for every key in `overSubscribed`, iterating `providedBy` unconditionally (a presence fallback is fail-open on older cores, measured); `gate: match.resolved & platform.#contracts.routable`.

## Worker protocol

One worker agent per change, in its own git worktree of the change's repo.

1. Create the worktree from fresh `origin/main`, on the branch named in the change set: from the repo root, `git fetch origin`, then `git worktree add .claude/worktrees/<change> -b <branch> origin/main`, then work inside that directory. Read the repo's `AGENTS.md` and `openspec/config.yaml` first; they bind. The repo-specific setup in this file comes next.
2. Run the repo's apply workflow on the change: `openspec instructions apply --change <change> --json`, then `tasks.md` section by section. The commit task that closes each section is the only commit you make.
3. After the last section is green, push the branch: `git push -u origin <branch>`. Do not archive the change, open a PR, merge, tag or release; the supervisor does.
4. On a blocker, stop and report it. Do not widen scope, and do not edit another repo; a design question goes in the report.
5. End with exactly this block as your final message:

```text
change:      <ID> <repo>/<change>
branch:      <branch> @ <head sha> (pushed: yes|no)
sections:    <n>/<total> committed
commits:     <sha> <subject>   (one line per commit)
gates:       <command> -> pass|fail   (one line each; name every failing or skipped test)
surface:     <what the next change consumes: exported symbols, flags, SPEC sections, output strings>
deviations:  <every departure from design.md or this file, with the reason; "none">
supervisor:  <what only the supervisor can do: a release, a pin bump, merge order, a decision; "none">
follow-ups:  <work found outside this change's repo; "none">
```

## Supervisor protocol

1. **Wave 1.** Launch the worker for A, with this change's `orchestration.md` as its brief.
2. **Check each report.** Compare `deviations` and `surface` against `design.md` and the interface above. Send a worker back with a precise ask rather than fixing its branch yourself.
3. **Finalize each change.** For each, in the worktree: `openspec archive <change> --yes`, commit the archive (`chore(openspec): archive <change>`), push, open the PR per the repo's `AGENTS.md`. Merge order: A, then B, then C and D. After A merges, merge the core release PR that release-please opens and note the released version; after B merges, do the same for the library release PR.
4. **Wave 2.** Once A is released, launch B's worker with the core version. Then run the workspace root `task deps:update` and land its per-repo output (`fix(deps)` / `test(fixtures)` per the workspace commit skill); cli's must be on main before D merges.
5. **Wave 3.** Once B's branch is pushed, launch C's and D's workers. Until B is released they develop against B's pushed head as a Go pseudo-version. Once B is released, tell each worker the version; it rebases on `origin/main`, pins the release, reruns its gates and pushes. Then finalize C and D as in step 3.
6. **Follow-ups.** Run the out-of-repo follow-ups each change's `orchestration.md` lists, after the merge they wait on.

Every PR follows its repo's `AGENTS.md`: a body of at most 250 words, no bare `@name`, only the plain co-author trailer. Merging, releasing and pushing to `main` need the user's go-ahead.

## This change (D)

**Release class.** `fix(platform)!` overall, pre-GA. Section 1 is `fix(platform)`, section 2 `fix(render)`, section 3 `fix(deps)!` with a `BREAKING CHANGE:` footer: through the library, `opm` refuses platform modules pinning core older than `2.0.0-alpha.12`, and the two bug shapes now fail `opm platform check`. Every commit releases (none is `chore` or `test`). The PR title carries `fix(platform)!`.

**Worktree setup (cli).**

- Branch `fix/single-source-provider-count` from `origin/main`, worktree at `cli/.claude/worktrees/single-source-provider-count`. The planning commit (`chore(openspec): plan single-source-provider-count`) is already on this branch; rebase it onto fresh `origin/main` before section 1 and again before section 3.
- Export the workspace registry mapping in two lines (a one-line `export A=x B="$A"` leaves `OPM_REGISTRY` empty):

  ```bash
  export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
  export OPM_REGISTRY="$CUE_REGISTRY"
  ```

- Never point a test or a manual `opm` run at the real `~/.opm`. Copy `hack/opm-config.cue` into the session scratchpad and `export OPM_CONFIG=<that copy>`, so the config and the platform cache (which moves with the config) stay out of both the home directory and the tree. The e2e suite seeds its own home (`seedRenderHome`); pass `--config` on every manual run.
- `task test` runs unit, integration and e2e. Integration and the operator-owned e2e tests need the `kind-opm-dev` cluster with a reconciling operator: `task cluster:create` if it is absent, then `task cluster:operator` (installs the pinned operator, seeds the cluster Platform, applies `hack/kind-operator-rbac.yaml`). No local registry is needed; never start one for this change (Registry Policy rule 3 is not triggered: no fixture changes).
- Library: develop against B's pushed head as a Go pseudo-version (`go get github.com/open-platform-model/library@<B head sha>`); never a `replace` to a local path in a commit.

**Hazards.**

- **Red first, no red commit.** Tasks 1.4 and 1.6 run the three bug-shape command tests red twice: on library alpha.34 they must exit 0 (the inventory bug), on B's head they must exit 2 but fail on the report (vacuous wording, no `provided by`). Record both runs under `gates` as `<test> on library alpha.34 -> fail (expected)` and `<test> on B head -> fail (expected)`. A test skipped for registry reachability is not a red run. The section's only commit is after the report change, green.
- **Bug-shape fixtures require ONLY the provider contract.** A provider transformer that also requires the catalog-fulfilled container forms comparable pairs and changes the readout (the same hazard change A recorded for its pins).
- **No count in the CLI.** The report prints `ProvidedBy`, `OverSubscribed` and `Routable` as the inventory carries them. Do not derive provider keys from `RequiredBy`, transformer FQNs or registry iteration, and do not add a presence fallback for `ProvidedBy` (B refuses an inventory lacking it with `PlatformCoreTooOldError`).
- **The merge gate.** `hack/platform/` and `examples/` pin core `v2.0.0-alpha.10` on `origin/main` as planned (e9a9e4e). B's floor refuses every `--platform hack/platform` render (`tests/e2e/instance_build_test.go`, `internal/config/platform_test.go`, `tests/integration/platform-build`, `tests/integration/render-parity`) until the supervisor's workspace `task deps:update` `fix(deps)` commit re-pins them on `main`. Task 1.1 stops if it is not there. This change never edits `hack/platform/`, `examples/` or `templates/`.
- **The last section waits on B's release.** Task 3.1 stops after section 2, with nothing edited, when the release is not out. `go.mod` never reaches a PR on a pseudo-version.
- Commit messages are not Markdown: write every path major glued to its path (`opmodel.dev/core@v2`, `testing.opmodel.dev/catalogs/k8up@v3`), never a bare at-sign followed by a name, and no body line starting with a word followed by an opening parenthesis (the squash body reaches release-please).

**Waits on:**

- **B's pushed head** (`feat/read-provider-count-from-core`) for sections 1 and 2, and the supervisor's cli `fix(deps)` commit from the workspace `task deps:update` (core `v2.0.0-alpha.12` in `hack/platform` and `examples/`) on `origin/main` before section 1.
- **B's release** for section 3 (expected library `v1.0.0-alpha.35`; use the version the supervisor reports). If B's report under `surface` renames any of `ContractInventory.ProvidedBy`, `errors.PlatformCoreTooOldError` (fields `Platform`, `Field`, `Since`), `schema.ProvidedBySince` or the `render refused before staging` wrapping, code against B's names and list the difference under `deviations`.

**Hands off:** nothing to another change. Under `surface`, report: the `opm platform check` report lines (`provided by  <keys>`, `(defined by no enabled catalog)`, the `defined contracts: 0` note), the re-pin hint text, and the library version pinned.

**Follow-ups outside this repo (supervisor):**

1. After D merges and the cli release is cut: regenerate the `opmodel.dev` CLI reference (`task generate` there), which picks up the changed `opm platform check --help`.
2. The personal `opm-kind-demo` platform re-pin to core `v2.0.0-alpha.12` (hand-written; refused by any `opm` built on this change until re-pinned).
3. `opm-suite-installer`: whoever moves its checksum-pinned `opm` (`OPM_VERSION` in its `Containerfile`, `v1.0.0-alpha.20` as of planning) past this change's release must re-pin its hand-written `platform/` module to core `v2.0.0-alpha.12` and re-vendor core (`task vendor:sync`) in the same change, or its installs are refused with the re-pin error.
