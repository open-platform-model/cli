## Why

Enhancement [0002](../../../../enhancements/0002/) renames the OPM `Release` family to `Instance`. Slice **X3** renames the user-facing CLI surface: the `release` command group becomes `instance`. X1 (already committed on this branch) renamed the underlying types/loader to `Instance`, X2 removed the dead bundle path — but the command a user actually types is still `opm release …`, the cobra group still lives in `internal/cmd/release/`, and the command-target plumbing in `internal/cmdutil` and `internal/workflow` still carries the `Release` name. Leaving the command verb at `release` while everything beneath it is `Instance` is the most visible inconsistency of the whole rename; X3 closes it.

This is slice X3 of the bundled, atomic per-repo CLI PR (X1–X4): co-implemented with X1 (module-instance types/loader), X2 (bundle removal), and X4 (label domain + inventory), and bulk-archived together. It is a **BREAKING** change to the user-facing command name; per D8 (hard rename, no alias) and the established fact that the CLI has no external consumers, the old `release` verb and `rel` alias are dropped outright — no compatibility alias is retained.

## What Changes

- **BREAKING** — `git mv internal/cmd/release/ → internal/cmd/instance/`; package `release` → `instance`; `release.go → instance.go`; `release_test.go → instance_test.go`.
- **BREAKING** — cobra group: `Use: "release"` → `"instance"`, `Aliases: ["rel"]` → `["inst"]` (the old `rel` is **not** kept as a back-compat alias, per D8), and the group's `Short`/`Long` help/examples reworded from "release" to "instance".
- **BREAKING** — rename the group constructor `NewReleaseCmd` → `NewInstanceCmd` and all nine subcommand constructors `NewRelease{Vet,Build,Apply,Diff,Status,Tree,Events,Delete,List}Cmd` → `NewInstance*Cmd`; update `internal/cmd/root.go` import alias (`cmdrelease` → `cmdinstance`) and `AddCommand` wiring.
- **BREAKING** — `git mv internal/cmdutil/release_arg.go → instance_arg.go` and `release_target.go → instance_target.go`; rename the command-facing identifiers: `ReleaseArg`, `ResolveReleaseArg`, `isReleasePath`, `resolveReleaseArgFromFile`, `extractReleaseFileIdentity`, `ResolvedReleaseTarget`, `ResolveReleaseTarget` to their `Instance` forms, plus their call sites (`internal/cmd/instance/{delete,events,status,tree}.go`, `internal/workflow/query/status.go`, and their tests). `ReleaseSelectorFlags` (`cmdutil/flags.go`, the `--release-name`/`--release-id` flag bundle used by the `mod-*` deprecated aliases) is **left to X4** — it belongs to the selector/inventory domain, not the command-target plumbing.
- Rename the X2-deferred test `TestReleaseVetCmd_RejectsBundleRelease` → `TestInstanceVetCmd_*` (X2 left it in place by design — it tests arg-count, not bundle code) and drop the now-obsolete `BundleRelease file is rejected with clear error` scenario X2 handed off (bundle support and its message were removed in X2).
- Update **live user docs** — `README.md` (`### Release Operations`, the `opm release` prose and `vet`/`build`/`apply` invocations) and `QUICKSTART.md` (the `opm release …` command lines only; the `examples/releases/**/release.cue` *paths* are X4) — to `opm instance …`, with `// Was:`-style breadcrumbs per D11. Historical `docs/roadmap.md`, `docs/rfc/0007-*`, `docs/design/render-pipeline-*` are frozen records and left as-is.

## Capabilities

### New Capabilities

_None._ This slice renames an existing capability and edits the content of four others.

### Modified Capabilities

- `inst-commands` _(**RENAMED** from `rel-commands`)_: the command group and its nine subcommands move from `release`/`rel` to `instance`/`inst`; constructor and help-text requirements restate the instance verb. The X2-handed-off `BundleRelease file is rejected with clear error` scenario is dropped here.
- `cmd-structure`: the command-package-organisation and root-wiring requirements update the registered group from `release` to `instance` (package `internal/cmd/instance/`, import alias `cmdinstance`, constructor `NewInstanceCmd`), and the `opm release build` argument-branching requirement and the module-group cross-refs (`point user to opm release build <file>`) flip to the `instance` verb. Only the command *verb* changes here; the `--release-name`/`--release-id` flag names are left for X4.

### Coordination seams (documented, not delta'd here)

- **`mod-status` / `mod-events` / `mod-list` / `mod-apply` are deferred entirely to X4.** Their content is dominated by the deployed-"release"/inventory domain noun (`--release-name`/`--release-id` flags, "release inventory record", `ReleaseSelectorFlags`), which is X4's `release-inventory` / `release-identity-labeling` / `inventory-ownership` territory. To keep single-ownership of each capability (no requirement restated by two slices — a silent archive-order revert the atomic PR cannot catch), X3 authors **no** `mod-*` delta, and leaves `ReleaseSelectorFlags` (the `--release-name`/`--release-id` flag bundle those aliases use) to X4. See D-X3.6.
- The physical rename of `openspec/specs/rel-commands/` → `inst-commands/` is a **spec-sync / archive-time** operation, not a manual `git mv` in this change — consistent with X1, which left its main-spec folders (`module-release-type`, …) release-named pending the bulk-archive. X3 authors the `inst-commands` delta; the folder rename rides `openspec archive`. **Hygiene risk:** several `cli` main specs carry malformed `## ADDED Requirements` headers in a main-spec position (e.g. `release-workflow/spec.md`) — the same pathology that forced the opm-operator O-wave to archive with `--skip-specs`; if cli hits it, the folder renames become a manual post-archive hygiene pass (tracked in `design.md`).
- `release-workflow` is the **CI/release-please** capability ("Release triggers on version tags"), unrelated to the `release` command — explicitly **not** touched by X3.
- Label domain (`module-release.opmodel.dev/*`), `internal/inventory` record types, and `examples/releases/**` paths are **X4**, even where they share files (e.g. `QUICKSTART.md`, `internal/workflow/query/inventory_test.go`).

## Impact

- **Affected packages**: `internal/cmd/release` (→ `instance`), `internal/cmd/root.go`, `internal/cmdutil` (`release_{arg,target}.go`, `ReleaseSelectorFlags`), `internal/workflow/query` (command-target call sites only), `internal/cmd/module` (cross-ref prose).
- **Affected docs**: `README.md`, `QUICKSTART.md` (live user-facing command refs).
- **Out of scope (later slice, same PR)**: X4 — label domain, inventory record types, `examples/releases/**` → `examples/instances/**`, `tests/integration/rel-*` → `inst-*`, `openspec/specs` physical dir renames.
- **Upstream dependency**: none new — pure intra-`cli` rename on top of X1/X2. Depends on X1 (committed); not independently gated on `library`.
- **API/UX**: user-visible command rename — `opm release <cmd>` and `opm rel <cmd>` stop working; users invoke `opm instance <cmd>` / `opm inst <cmd>`. Internal Go API renamed (no external consumers).
- **SemVer**: MAJOR (breaking command rename); ships on enhancement 0002's `v1.0.0-alpha.N` line per D13, bundled in the atomic CLI PR.
