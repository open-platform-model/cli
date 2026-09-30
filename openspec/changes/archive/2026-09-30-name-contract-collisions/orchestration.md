# Orchestration: name-contract-collisions

This file is the brief for the worker agent that implements this change and for the supervisor that stitches the change set together. The same text sits in every change of the set; only the title and the "This change" section differ.

## Change set

Five OpenSpec changes, planned together on 2026-09-30, in four repos. Four of them do one job:

- core stops failing to evaluate a platform that enables two majors of one catalog sharing contract keys. It folds only the keys with exactly one enabled definer and reports the rest as `collisions` and `collidingEntries`, with `routable` false.
- The library turns a collision into a typed render refusal and decodes it in `Contracts()`.
- The operator and cli name the collision.

The fifth (E) is independent: it makes the operator's build-compatibility verdict deterministic on a platform carrying two majors of one catalog. The fix is an interim safety net and does not add side-by-side majors; enhancement 0026 D9 later makes two majors legitimate through per-resolution builds (0026 OQ17 and 05-risks recommend both fixes independent of 0026). No change claims a 0026 decision, so none carries an `enhancement.yaml`.

| ID | Repo | Change | Branch | Wave | Starts when | Merges when |
| --- | --- | --- | --- | --- | --- | --- |
| A | core | `fold-colliding-contract-keys` | `feat/fold-colliding-contract-keys` | 1 | now | first; release cut after |
| E | opm-operator | `index-build-compat-by-major` | `fix/index-build-compat-by-major` | 1 | now | any time, independent of A to D |
| B | library | `refuse-colliding-contracts` | `feat/refuse-colliding-contracts` | 2 | A released | after A is released |
| C | opm-operator | `name-contract-collisions` | `fix/name-contract-collisions` | 3 | B's branch pushed | after B is released |
| D | cli | `name-contract-collisions` | `fix/name-contract-collisions` | 3 | B's branch pushed | after B is released |

What each hands on:

- **A** publishes, in core `2.0.0-alpha.13` (actual tag reported under `surface`):
  - `#ContractInventory.collisions: [...#ContractFQNType]`: sorted keys with more than one enabled definer.
  - `#ContractInventory.collidingEntries: [#ContractFQNType]: [...#ModulePathType]`: the sorted registry keys defining each.
  - `routable: len(overSubscribed) == 0 && len(collisions) == 0`.
- **B** reads them in the render build and in `Contracts()`, raises a typed refusal, moves `DefaultSchemaModule` to A's tag, and publishes the interface below in a library release.
- **C** and **D** consume B's release.
- **E** consumes nothing and hands on nothing.

## Interface B -> C, D

The contract C and D consume. B may refine names only by reporting the change under `deviations`; C and D code against what B reports under `surface`.

```go
package platform // github.com/open-platform-model/library/opm/platform

type ContractInventory struct {
	// ...existing fields unchanged...

	// Routable is true exactly when OverSubscribed and Collisions are both empty.
	Routable bool `json:"routable"`

	// Collisions lists, ascending, every contract key more than one enabled
	// registry entry's catalog lists. Such a key is in none of DefinedBy,
	// RequiredBy, Unfulfilled or Comparable, so Fulfilled and Discriminated
	// can read true while Collisions is non-empty; Routable is false.
	Collisions []string `json:"collisions"`

	// CollidingEntries maps each Collisions key to the sorted registry keys
	// (path@major) of the enabled entries listing it.
	CollidingEntries map[string][]string `json:"collidingEntries"`
}

// Contracts() decodes an absent collisions/collidingEntries as empty: every
// core before A's tag fails to evaluate a colliding platform at acquire.
```

```go
package errors // github.com/open-platform-model/library/opm/errors

type ContractCollision struct {
	Key      string   `json:"key"`
	Catalogs []string `json:"catalogs"` // sorted registry keys, path@major
}

// Raised (joined, first) by the render gate, platform-wide, under SkipUnprovided too.
type ContractCollisionsError struct{ Contracts []ContractCollision }

// Raised only when #contracts.routable is false and no over-subscription or collision row explains it.
type NotRoutableError struct{}
```

`kernel.RenderDiagnostics.Collisions []oerrors.ContractCollision` (sorted by key) and `RenderDiagnostics.Routable bool`. `schema.DefaultSchemaModule` = A's tag. `schema.ProvidedBySince` (the floor) is unchanged. `UnresolvedDemand.Colliding []string` is diagnostic only.

The rule behind it (A computes it, B reads it, C and D print it):

1. A definer is an ENABLED registry entry whose catalog lists the key in `#resources`, `#traits` or `#blueprints`. A disabled entry never counts.
2. A key with exactly one definer folds into `defined` and `definedBy` as before. A key with more is a collision: it is in `collisions` and `collidingEntries` and in none of `defined`, `definedBy`, `requiredBy`, `unfulfilled` or `comparable`. `providedBy` and `overSubscribed` are unaffected and can co-occur with a collision.
3. `routable` is false while any collision exists. `fulfilled` and `discriminated` can still read true (the stated limitation), so no consumer reads either as safe while `collisions` is non-empty.
4. The render refuses a colliding platform with `ContractCollisionsError`, whatever the instance and whatever `SkipUnprovided` says. Its rows are `{key, catalogs: collidingEntries[key]}` for every key in `collisions`, guarded on presence. Absence is provably empty: an older core fails to evaluate such a platform. A `routable` false that no row explains raises `NotRoutableError`.
5. The operator words it as reason `ContractCollisions`, ahead of `OverSubscribedContracts` and `ComparablePredicates`. The cli prints a colliding-contracts section and counts collisions in the routable verdict and the exit message.

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

1. **Wave 1.** Launch the workers for A and E, each with its change's `orchestration.md` as its brief.
2. **Check each report.** Compare `deviations` and `surface` against `design.md` and the interface above. Send a worker back with a precise ask rather than fixing its branch yourself.
3. **Finalize each change.** For each, in the worktree: `openspec archive <change> --yes`, commit the archive (`chore(openspec): archive <change>`), push, open the PR per the repo's `AGENTS.md`. Merge order: A, then B, then C and D; E whenever it is green. After A merges, merge the core release PR that release-please opens and note the released version; after B merges, do the same for the library release PR.
4. **Wave 2.** Once A is released, launch B's worker with the core version. Do NOT run the workspace root `task deps:update` yet. Run it once B is released, and land its per-repo output (`fix(deps)` / `test(fixtures)` per the workspace commit skill). This departs from the previous set on purpose: a kernel older than B renders a colliding platform pinned to A's core (duplicate objects, measured), so no workspace platform moves to A's core before a refusing kernel exists.
5. **Wave 3.** Once B's branch is pushed, launch C's and D's workers. Until B is released they develop against B's pushed head as a Go pseudo-version. Once B is released, tell each worker the version; it merges `origin/main`, pins the release, reruns its gates and pushes. Then finalize C and D as in step 3.
6. **Follow-ups.** Run the out-of-repo follow-ups each change's `orchestration.md` lists, after the merge they wait on.

Every PR follows its repo's `AGENTS.md`: a body of at most 250 words, no bare `@name`, only the plain co-author trailer. Merging, releasing and pushing to `main` need the user's go-ahead.

## This change (D)

**Release class.** `fix(platform)` overall, PATCH, pre-GA. Section 1 is `fix(platform)`, section 2 `fix(render)`, section 3 `fix(deps)`. Not breaking: a colliding platform exited non-zero before (it did not build) and still does, now naming why; every platform that evaluates today reads the same report. Every commit releases (none is `chore` or `test`). The PR title carries `fix(platform)`.

**Worktree setup (cli).**

- Branch `fix/name-contract-collisions` from `origin/main`, worktree at `cli/.claude/worktrees/name-contract-collisions`. The planning commit (`chore(openspec): plan name-contract-collisions`) is already on this branch; merge it onto fresh `origin/main` before section 1 and again before section 3 (the release pin).
- Export the workspace registry mapping in two lines (a one-line `export A=x B="$A"` leaves `OPM_REGISTRY` empty):

  ```bash
  export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
  export OPM_REGISTRY="$CUE_REGISTRY"
  ```

- Never point a test or a manual `opm` run at the real `~/.opm`. Copy `hack/opm-config.cue` into the session scratchpad and `export OPM_CONFIG=<that copy>`, so the config and the platform cache (which moves with the config) stay out of both the home directory and the tree. The e2e suite seeds its own home (`seedRenderHome`) and runs with `OPM_CONFIG` unset; pass `--config` on every manual run.
- `task test` runs unit, integration and e2e. Integration and the operator-owned e2e tests need the `kind-opm-dev` cluster with a reconciling operator: `task cluster:create` if it is absent, then `task cluster:operator`. No local registry is needed; never start one for this change. Registry Policy rule 3 is not triggered: the colliding fixtures are inline test platforms and temp-dir copies, nothing is published, and A's core resolves from GHCR.
- Library: develop against B's pushed head as a Go pseudo-version (`go get github.com/open-platform-model/library@<B head sha>`); never a `replace` to a local path in a commit.

**Hazards.**

- **Red first, no red commit.** Task 1.4 runs the colliding command tests on library alpha.35: the collision-only platform must print the vacuous report while exiting with the validation code, and the collision-plus-over-subscription platform must read `1 contract is over-subscribed` with no colliding heading. Task 1.6 reruns them on B's head (still red: the report does not carry `Collisions`). Tasks 2.1 and 2.2 do the same for the render rows. Record each as `<test> on <pin> -> fail (expected)` under `gates`. A test skipped for registry reachability is not a red run.
- **The colliding fixtures pin A's core explicitly**, in their own module file, never through `schema.DefaultSchemaVersion()`: on library alpha.35 the default is alpha.12, where a colliding platform does not build, and the red run would prove nothing.
- **Fixture providers require ONLY the provider contract.** A provider transformer that also requires the catalog-fulfilled container forms comparable pairs and changes the readout.
- **No count in the CLI.** Print `Collisions`, `CollidingEntries`, `OverSubscribed`, `ProvidedBy` and `Routable` as the inventory carries them, and the render's `Collisions` rows as the kernel carries them. Do not derive collisions from `CollidingEntries` lengths, registry iteration or `DefinedBy`, and add no presence fallback (B decodes absence as empty).
- **`fulfilled` and `discriminated` can read yes under a collision** (core's stated limitation). Do not reword or recompute them; the colliding section's note and the exit status carry the refusal.
- **Two additions beyond the approved design** (design.md § 6: the `Colliding` case in `cmdutil.FormatUnresolvedDemands`, and `refusalHint` withholding hints under a collision). Implement them, and list both under `deviations` so the supervisor can confirm or strike them.
- **Never edit** `hack/platform/`, `examples/` or `templates/`; the e2e colliding platform is a temp-dir copy of `hack/platform` with its core pin rewritten in the copy.
- **The last section waits on B's release.** Task 3.1 stops after section 2, with nothing edited, when the release is not out. `go.mod` never reaches a PR on a pseudo-version.
- Commit messages are not Markdown: write every path major glued to its path (`opmodel.dev/core@v2`, `testing.opmodel.dev/catalogs/base@v2`), never a bare at-sign followed by a name, and no body line starting with a word followed by an opening parenthesis (the squash body reaches release-please).

**Waits on:**

- **A's release** (expected core `v2.0.0-alpha.13`) resolving from GHCR, and **B's pushed head** (`feat/refuse-colliding-contracts`) for sections 1 and 2. Task 1.1 stops if either is missing.
- **B's release** for section 3 (use the version the supervisor reports). If B's report under `surface` renames any of `ContractInventory.Collisions`, `CollidingEntries`, `errors.ContractCollision` (fields `Key`, `Catalogs`), `errors.ContractCollisionsError`, `errors.NotRoutableError`, `RenderDiagnostics.Collisions` or `UnresolvedDemand.Colliding`, code against B's names and list the difference under `deviations`.
- Merge origin/main if C or E has merged (no file overlap: they are operator changes).

**Hands off:** nothing to another change. Under `surface`, report: the `opm platform check` report lines (the `colliding contracts: N` section, its `defined by  ` rows and omission note, the routable verdict wording), the exit message, the render collision row text, the unresolved `Colliding` wording, and the library version pinned.

**Follow-ups outside this repo (supervisor):**

1. After D merges and the cli release is cut: regenerate the `opmodel.dev` CLI reference (`task generate` there), which picks up the changed `opm platform check --help`.
2. Hold the workspace `task deps:update` until B is released (Supervisor protocol step 4); `hack/platform` and `examples/` move to A's core only through it.
