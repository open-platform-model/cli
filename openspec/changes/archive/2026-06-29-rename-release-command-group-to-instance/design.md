## Context

Slice X3 of enhancement [0002](../../../../enhancements/0002/) (`Release` → `Instance` rename). X1 (committed on branch `0002-cli-x1-rename-module-instance`) renamed the `pkg/module` types, the loader, and the instance-file convention; X2 removed the dead bundle path. The user-facing command verb is still `opm release`, the cobra group still lives in `internal/cmd/release/`, and the command-target plumbing (`internal/cmdutil/release_{arg,target}.go`, `internal/workflow/query` call sites) still carries the `Release` name. X3 renames that surface to `instance`.

This lands in the single atomic per-repo CLI PR (X1–X4) and bulk-archives with the other cli slices. It depends on X1 (committed); it is not independently gated on `library`. It is co-implemented with X4 (label domain + inventory + example/fixture moves), which shares some files (`QUICKSTART.md`, `internal/workflow/query/inventory_test.go`) but no identifiers.

Command surface in scope (group + nine subcommands):

```
internal/cmd/release/  →  internal/cmd/instance/
  release.go (NewReleaseCmd, Use:"release", Aliases:["rel"])  → instance.go (NewInstanceCmd, Use:"instance", Aliases:["inst"])
  vet build apply diff        (file-arg subcommands)   NewRelease*Cmd → NewInstance*Cmd
  status tree events delete list  (cluster-query)      NewRelease*Cmd → NewInstance*Cmd
internal/cmd/root.go    cmdrelease alias + AddCommand wiring
internal/cmdutil/release_{arg,target}.go  → instance_{arg,target}.go
internal/workflow/query/status.go         command-target call sites only
```

## Goals / Non-Goals

**Goals:**

- Rename the user-facing command group `release`/`rel` → `instance`/`inst`, including all nine subcommands, the root wiring, and the cobra help/examples.
- Rename the command-target plumbing identifiers (`ReleaseArg`, `ResolveReleaseArg`, `ResolvedReleaseTarget`, `ResolveReleaseTarget`, helpers) and move their files (`cmdutil/release_{arg,target}.go`). `ReleaseSelectorFlags` (`flags.go`) is **not** X3 — see Non-Goals / D-X3.6.
- Update live user docs (`README.md`, `QUICKSTART.md` command lines) to `opm instance …` with `// Was:` breadcrumbs.
- Absorb the X2-handed-off seam: rename `TestReleaseVetCmd_RejectsBundleRelease` and drop the obsolete bundle-rejection scenario from the renamed `inst-commands` capability.
- Author the `inst-commands` (renamed) + `cmd-structure` spec deltas only.

**Non-Goals:**

- Authoring `mod-status` / `mod-events` / `mod-list` / `mod-apply` deltas — deferred to X4 (D-X3.6).

- Retaining `release`/`rel` as a back-compat alias. Dropped per D8 — see D-X3.1.
- The label domain (`module-release.opmodel.dev/*`), `internal/inventory` record types (`ReleaseInventoryRecord`, `ReleaseMetadata`, `ReleaseName`), the `ReleaseSelectorFlags` flag bundle (`cmdutil/flags.go`, `--release-name`/`--release-id`), and `examples/releases/**` → `examples/instances/**` path moves — these are **X4**.
- The physical `git mv` of `openspec/specs/rel-commands/` → `inst-commands/` — that is an archive-time spec-sync operation (D-X3.4).
- Touching `release-workflow` (the CI/release-please capability — unrelated to the command).
- Independent green-ness of X3 alone. Like X1/X2, X3 is one slice of an atomic PR; remaining `"ModuleRelease"`/`release.cue` fixtures in command/integration/e2e tests reconcile across the same PR.

## Decisions

### D-X3.1 — Drop `rel`; do not keep it as a back-compat alias

The cobra group changes `Use: "release"` → `"instance"` and `Aliases: ["rel"]` → `["inst"]`, with no `"release"`/`"rel"` retained for compatibility. Rationale: enhancement D8 is a hard rename (no alias), and the CLI has no external consumers, so there is no migration audience to cushion. Keeping `rel` would preserve the exact vocabulary the rename exists to retire and muddy `--help`. Trade-off accepted: existing muscle memory and any local scripts using `opm release …` break — acceptable for a pre-1.0 alpha with no external users. Source: enhancement 0002 D8; user direction 2026-06-28.

### D-X3.2 — Rename the X2-deferred `TestReleaseVetCmd_RejectsBundleRelease`

X2 left this test in place (per its D-X2.3) because it asserts *arg-count* behavior, not bundle code. With the file `release_test.go → instance_test.go` and the group renamed, X3 renames the test to `TestInstanceVetCmd_*` and aligns its assertions to the post-X2 reality — a `kind: "BundleRelease"` file now errors via `DetectInstanceKind`'s default (`unknown instance kind: "BundleRelease"`), not the removed friendly message. The corresponding `inst-commands` scenario X2 handed off is dropped (D-X3.3).

### D-X3.3 — `inst-commands` is the renamed `rel-commands`, minus the bundle scenario

The delta is authored under the new capability name `inst-commands` (mirroring X1's `module-instance-type` convention) with `## MODIFIED Requirements`. It restates the command-group requirements with the `instance`/`inst` verb and the `NewInstance*Cmd` constructors, and **drops** the `BundleRelease file is rejected with clear error` scenario that X2 explicitly handed to X3 (the quoted message no longer exists). X2 deliberately authored no `rel-commands` delta, so X3 is the sole editor of this capability — no double-edit.

### D-X3.4 — The main-spec folder rename rides the archive, not this change

`openspec/specs/rel-commands/` is **not** hand-renamed in X3. X1 established the pattern: its main-spec folders (`module-release-type`, …) are still release-named on disk, pending the bulk-archive's spec-sync. X3 authors the `inst-commands` delta only; the physical folder rename happens uniformly for X1–X4 at `openspec archive`. Hand-renaming now would desync from the sibling slices and likely collide with the sync.

**Hygiene risk (flagged, not fixed here):** several `cli` main specs carry malformed delta-style headers in main-spec position — e.g. `openspec/specs/release-workflow/spec.md` opens with `## ADDED Requirements`. This is the exact pathology that forced the opm-operator O-wave to run `openspec archive --skip-specs` and defer a spec-hygiene pass. If the cli bulk-archive hits it, the `rel-commands → inst-commands` (and X1's) folder renames become a **manual post-archive hygiene pass**, not an automatic sync. Tracked here so it is not a surprise at archive time; the repair itself is out of scope for X3.

### D-X3.5 — Docs: live user docs in scope, frozen design docs excluded

`README.md` (`### Release Operations`, prose, `opm release vet|build|apply`) and `QUICKSTART.md` (the `opm release …` command lines) are live user-facing and are updated to `opm instance …` with breadcrumbs (D11). In `QUICKSTART.md`, X3 edits only the command *verbs*; the `./examples/releases/**/release.cue` *paths* belong to X4 (the example-tree move) — X3 leaves them so X4 does not double-edit. `docs/roadmap.md`, `docs/rfc/0007-*`, and `docs/design/render-pipeline-*` are historical/frozen records (like enhancement 0001) and are left untouched.

### D-X3.6 — Defer all `mod-*` spec deltas to X4 (single capability ownership)

`mod-status` / `mod-events` / `mod-list` / `mod-apply` are dominated by the deployed-"release" / inventory domain noun (`--release-name`/`--release-id` flags, "release inventory record", "release health", `ReleaseSelectorFlags`), which is X4's territory (`release-inventory`, `release-identity-labeling`, `inventory-ownership`). `planned-changes.md` listed `mod-*` under X3 (for command cross-refs) *and* `mod-apply` again under X4 (for inventory) — but OpenSpec restates each MODIFIED requirement in full, so two slices delta'ing the same requirement means whichever archives second silently reverts the other's wording, and the atomic PR cannot catch it (code compiles; only the synced spec is wrong). To preserve the X2-established invariant "no capability delta'd by both slices," X3 authors **no** `mod-*` delta; X4 owns the entire `mod-*` rename (command cross-refs *and* the inventory/flag/noun rename, since it must restate those requirements anyway).

X3's *code* renames only the cmdutil `release_{arg,target}.go` identifiers it owns (`ReleaseArg`, `ResolveReleaseTarget`, …); `ReleaseSelectorFlags` (`cmdutil/flags.go`, the `--release-name`/`--release-id` bundle consumed by the `mod-*` aliases and inventory selectors) is X4's, alongside its `mod-*` deltas. `ReleaseArg.ToSelectorFlags` will momentarily return `*ReleaseSelectorFlags` after X3 (cross-type reference, compiles fine) until X4 renames the bundle. This is a deliberate deviation from `planned-changes.md`'s literal X3 wording, recorded here per user direction 2026-06-28. Trade-off: at X3 code-time the `mod-*` main specs are momentarily stale (reference the old verb/type) until X4 — acceptable under the atomic-PR model (specs sync at archive, not per-slice).

### D-X3.7 — Rename depth: structural + help text; cross-package symbols left to their slices

Implementation revealed `internal/cmd/release/` carries ~137 `release` tokens, most of them **cross-package references X3 does not own** (`cmdutil.ReleaseFileFlags`, `cmdutil.ReleaseSelectorFlags`, `ReleaseName`/`ReleaseID`/`ReleaseMetadata`/`ReleaseInventoryRecord` from `internal/inventory`, `EvaluateReleaseHealth`/`PrintReleaseStatus`/`RenderReleaseListOutput` from health/output, `FromReleaseFile`/`RenderFromReleaseFile` from `pkg/render`). X3 renames the **command-structure surface and user-facing help only** (user direction 2026-06-28):

- **In scope:** package `release`→`instance`; the 10 `NewRelease*Cmd` constructors; the 9 `runRelease*` funcs; cobra `Use`/`Short`/`Long`/`Example` strings including domain nouns ("release file"→"instance file") and example filenames (`*_release.cue`→`*_instance.cue`, `release.cue`→`instance.cue`); the `releaseFile` path variable; `cmdutil` `release_{arg,target}.go` (`ReleaseArg`, `ResolveReleaseArg`, `ResolvedReleaseTarget`, `ResolveReleaseTarget`, helpers); test func names in the moved test file.
- **Left to X4 / X1-gap (compiling cross-package references):** `cmdutil.ReleaseFileFlags` (render-file flag bundle — deferred to X4 with the other `cmdutil` flag types + the `cmdutil` spec delta), `cmdutil.ReleaseSelectorFlags`, `ResolveReleaseIdentifier`, the inventory/health/output symbols, and the `releaseName`/`releaseID` identity vars + `--release-name`/`--release-id` flags.

**Boundary refinement (post-verify, reviewer-driven):** the X3/X4 line is **Go symbols vs. user-facing strings**, not "command help vs. runtime output." Every string a user *reads* when invoking `opm instance …` — cobra help, `output.Println`/`Prompt` messages, error text, and structured-log attribute keys — is X3 and must say "instance" (else it contradicts the command just typed). Only the underlying Go identifiers (`releaseName`/`releaseID` vars, `--release-name`/`--release-id` flag names, `ReleaseSelectorFlags`, inventory `Release*` types) stay `Release`-named for X4. The first implementation pass over-deferred `list.go`/`delete.go`/`tree.go` runtime strings to X4; the verify pass corrected them.

Consequence: post-X3 the `instance` package still references many `Release`-named symbols from other packages. This is expected under the atomic-PR model and resolves as X4 / the X1-gap land in the same PR. `ReleaseLogger`/`releaseLog` are renamed only if locally defined in this package; if they come from `internal/output`, they are left (X4).

## Risks / Trade-offs

- **User-visible breakage.** `opm release …` and `opm rel …` stop working with no alias. Mitigated by: no external users, alpha line, breadcrumbs in docs, and `unknown command` guidance from cobra. Accepted per D-X3.1.
- **Shared-file edits with X4 in one PR.** `QUICKSTART.md` and `internal/workflow/query/inventory_test.go` are touched by both slices. Mitigated by clean ownership: X3 owns command *verbs* / command-target *identifiers*; X4 owns example *paths* / inventory *types* / label *keys*. No line is owned by both.
- **Archive-time folder-rename / `--skip-specs` risk.** See D-X3.4 — surfaced now; resolution deferred to the bulk-archive step / a hygiene pass.

## Migration Notes

No external consumers (the CLI has no external API surface; D8 hard-rename, no shim). User-visible delta: replace `opm release <cmd>` / `opm rel <cmd>` with `opm instance <cmd>` / `opm inst <cmd>`. The subcommand names, flags, and behavior are unchanged — only the group verb and alias move.
