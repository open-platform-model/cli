Branch: `0002-cli-x1-rename-module-instance` (stacked on X1 `7a9175d` + X2; atomic-PR / bulk-archive model — **not** a fresh branch). Per the model, full `task test` is **not** expected green at X3 alone; integration/e2e fixtures reconcile in X4 within the same PR.

## 1. Move + rename the command-group package (`inst-commands`, `cmd-structure`)

- [x] 1.1 `git mv internal/cmd/release/ internal/cmd/instance/`; `git mv internal/cmd/instance/release.go internal/cmd/instance/instance.go`; `git mv internal/cmd/instance/release_test.go internal/cmd/instance/instance_test.go`.
- [x] 1.2 Change `package release` → `package instance` across all files in the moved dir.
- [x] 1.3 `instance.go`: rename `NewReleaseCmd` → `NewInstanceCmd`; cobra `Use: "release"` → `"instance"`, `Aliases: []string{"rel"}` → `[]string{"inst"}` (drop `rel`, no back-compat — D-X3.1); reword `Short`/`Long`/examples from "release" to "instance" (keep the "starting from an instance definition … for module source use `opm module`" steer). Add a `// Was: opm release / rel` breadcrumb (D11).
- [x] 1.4 Rename the nine subcommand constructors `NewRelease{Vet,Build,Apply,Diff,Status,Tree,Events,Delete,List}Cmd` → `NewInstance*Cmd` (definitions in `vet.go`/`build.go`/`apply.go`/`diff.go`/`status.go`/`tree.go`/`events.go`/`delete.go`/`list.go`, and the `c.AddCommand(...)` calls in `instance.go`).
- [x] 1.5 Update each subcommand's user-facing strings (`Use`, `Short`, `Long`, `Example`) from "release" to "instance" where they name the command; keep behavior, flags, and arg contracts unchanged.

## 2. Root wiring (`cmd-structure`)

- [x] 2.1 `internal/cmd/root.go`: import alias `cmdrelease "…/internal/cmd/release"` → `cmdinstance "…/internal/cmd/instance"`; `rootCmd.AddCommand(cmdrelease.NewReleaseCmd(&cfg))` → `cmdinstance.NewInstanceCmd(&cfg)`. Grep-confirm no other `cmdrelease` reference survives.

## 3. cmdutil arg/target plumbing (`cmd-structure` support; **not** flags.go)

- [x] 3.1 `git mv internal/cmdutil/release_arg.go internal/cmdutil/instance_arg.go`; rename `ReleaseArg` → `InstanceArg`, `ResolveReleaseArg` → `ResolveInstanceArg`, `isReleasePath` → `isInstancePath`, `resolveReleaseArgFromFile` → `resolveInstanceArgFromFile`, `extractReleaseFileIdentity` → `extractInstanceFileIdentity`, and the methods (`ToSelectorFlags`, `EffectiveNamespace`) receivers. Leave the `*ReleaseSelectorFlags` return type of `ToSelectorFlags` **unchanged** (X4 renames the bundle — D-X3.6); it is a compiling cross-type reference.
- [x] 3.2 `git mv internal/cmdutil/release_target.go internal/cmdutil/instance_target.go`; rename `ResolvedReleaseTarget` → `ResolvedInstanceTarget`, `ResolveReleaseTarget` → `ResolveInstanceTarget`.
- [x] 3.3 Update call sites: `internal/cmd/instance/{delete,events,status,tree}.go` and `internal/workflow/query/status.go` (`ResolveReleaseArg`/`ResolveReleaseTarget` → `Instance` forms). Add `// Was:` breadcrumbs at the moved files' heads (D11).
- [x] 3.4 Update affected tests: `internal/cmdutil/{flags_test,path_guard_test}.go` and `internal/workflow/query/status_test.go` (X3 identifiers only). **Do not** touch `internal/workflow/query/inventory_test.go` (X4 — inventory record types + label keys) or `cmdutil/flags.go` `ReleaseSelectorFlags` (X4).

## 4. X2-deferred test (`inst-commands` — D-X3.2)

- [x] 4.1 In `internal/cmd/instance/instance_test.go`, rename `TestReleaseVetCmd_RejectsBundleRelease` → `TestInstanceVetCmd_*` and align its assertion to post-X2 reality (a stray `kind: "BundleRelease"` now errors via `DetectInstanceKind`'s default → `unknown instance kind: "BundleRelease"`, not the removed friendly message). Rename any other `TestRelease*` test funcs in this file to `TestInstance*`.

## 5. Live user docs (D-X3.5, D11)

- [x] 5.1 `README.md`: `### Release Operations (`opm release`)` → `### Instance Operations (`opm instance`)`; the steer prose `use `opm release`` → `use `opm instance``; the `./bin/opm release vet|build|apply` invocations → `instance`. Add a one-line "Renamed from `opm release` (enhancement 0002)" note near the section head.
- [x] 5.2 `QUICKSTART.md`: change only the `opm release …` command **verbs** → `opm instance …`. **Leave** the `./examples/releases/**/release.cue` example *paths* untouched (X4 owns the example-tree move).
- [x] 5.3 Leave `docs/roadmap.md`, `docs/rfc/0007-*`, `docs/design/render-pipeline-*` untouched (frozen historical records — D-X3.5).

## 6. Spec deltas (authored in this change)

- [x] 6.1 `specs/inst-commands/spec.md` — renamed from `rel-commands` (ADDED-restatement + "Renamed from" breadcrumb), bundle scenario dropped (D-X3.3). _Authored._
- [x] 6.2 `specs/cmd-structure/spec.md` — MODIFIED: package-org, root registration, cluster-query migration, `opm instance build` branching, module build/apply cross-refs, `--name` flag. _Authored._
- [x] 6.3 Do **not** hand-rename `openspec/specs/rel-commands/` → `inst-commands/` (rides the archive — D-X3.4).

## 7. Verification gate (this slice)

- [x] 7.1 `task fmt` clean on all touched files; `// Was:`/"Renamed from" breadcrumbs present at every rename site (D11/D12).
- [x] 7.2 `go build ./...` + `go vet ./...` green.
- [x] 7.3 Targeted tests green: `go test ./internal/cmd/instance/... ./internal/cmdutil/... ./internal/workflow/query/...` (the X3-owned subset; X4-owned `inventory_test.go` assertions may still reference release vocabulary — expected).
- [x] 7.4 `task lint` introduces 0 new issues vs the X2 HEAD (verify any flagged item reproduces pre-X3 with `git stash`).
- [x] 7.5 Grep sweep: no surviving `opm release`/`opm rel`, `cmdrelease`, `NewReleaseCmd`, `internal/cmd/release`, `ReleaseArg`/`ResolveReleaseTarget` outside `// Was:` breadcrumbs. Remaining `Release` tokens are expected X4 surface (`ReleaseSelectorFlags`, `--release-name`, inventory record types, label keys, `examples/releases`) and X1-gap core-type prose (`#ModuleRelease`).
- [x] 7.6 `openspec validate rename-release-command-group-to-instance --strict` passes.
- [x] 7.7 Manual smoke: `./bin/opm instance --help` lists nine subcommands; `./bin/opm inst --help` resolves; `./bin/opm release` errors as unknown command.

## Implementation notes / deviations

- **Held from archive.** Per the atomic-PR / bulk-archive model, X3 stays active alongside X1/X2/X4 and is bulk-archived with them; the `enhancements/0002/config.yaml` history event is recorded at bulk-archive time, not here.
- **mod-\* and `ReleaseSelectorFlags` deferred to X4 (D-X3.6).** A documented deviation from `planned-changes.md`'s literal X3 wording — single-ownership over double-delta.
- **Not green alone (by design).** `task test` (integration/e2e) reconciles with X4 in the same PR.

### Deviations / findings discovered during implementation

- **D-X3.7 rename depth applied.** The `internal/cmd/release/` package carried ~137 `release` tokens, most of them cross-package references X3 does not own. Per D-X3.7, X3 renamed the command-structure surface + user-facing help/output only and **left** the cross-package symbols (`cmdutil.ReleaseFileFlags`, `cmdutil.ReleaseSelectorFlags`, `ResolveReleaseIdentifier`, inventory `Release*` types, `render.FromReleaseFile`/`ReleaseFilePath`, `output.ReleaseLogger`/`releaseLog`, identity vars `releaseName`/`releaseID`). The renamed package therefore still references many `Release`-named symbols by design — resolves as X4 / the X1-gap land in the same PR.
- **Broader cross-ref footprint than tasks 1–3 enumerated (in scope, cmd-structure).** The `opm release` verb is gone, so every reference to it had to flip. Beyond the moved package this touched `internal/cmd/module/{build,apply,mod}.go` (+ their tests), `internal/cmdutil/path_guard.go` (+ test), and a `internal/workflow/render/render.go` doc comment — all `opm release …` → `opm instance …`. The `cmd-structure` spec delta already specified the module-group cross-refs. The `release.cue`/"release file" *nouns* in those same messages (D9 instance-file convention) were **left** for X4/X1-gap, producing accepted mixed-vocabulary seams (e.g. "use `opm instance`" + "release package").
- **Pre-existing X1-gap surfaced, NOT fixed here.** `internal/workflow/render/render_test.go::TestRenderFromReleaseFile_ValidValuesDoNotPanicAcrossRuntimes` fails on a stale `kind: "ModuleRelease"` fixture. Confirmed pre-existing: the file is byte-identical to X2 HEAD `a68a368` (`git diff` clean) and the test fails with X3 fully stashed. It is the X1 wire-kind gap (already noted in X2's tasks) and reconciles in X4 / the X1-gap fixup. Not X3's capability (`workflow/render`).
- **Pre-existing lint, NOT introduced by X3.** 3 `goconst` issues in `internal/cmd/module/{init,mod,vet}.go` (`"module"`/`"debugValues"`). Confirmed reproducing at clean X2 HEAD (X3 stashed). X3 adds 0 new lint issues; `internal/cmd/instance` + `cmdutil` lint clean.
- **Smoke (7.7) verified on a built binary:** `opm instance --help` lists all nine subcommands; `opm inst --help` resolves to the group; `opm release` → `unknown command "release" for "opm"`.
- **Removed orphaned testdata fixture.** `internal/cmd/instance/testdata/jellyfin_release.cue` came along with the package `git mv` but is referenced nowhere repo-wide (X2 noted these command tests build fixtures inline); deleted it and the now-empty `testdata/` dir (cleanliness, consistent with X2's orphaned-fixture removal).
- **Verify-pass fix: wrong article in help text.** The `release`→`instance` seds left 12 "a instance" strings (should be "an instance") across the cobra help in `diff/events/status/tree/apply/build/delete/vet.go`. Fixed `\ba instance\b` → `an instance`; rebuilt + re-tested + gofmt clean. Caught by `/opsx:verify` (Quality dimension).
- **Verify-pass fix: user-facing runtime strings were over-deferred to X4 (reviewer REQUEST_CHANGES).** Implementation initially left `list.go` ("No releases found" ×2, "listing releases" error), the `delete.go` confirmation prompt ("…for release %q"/"…for release-id %q"), and a `tree.go` structured-log key (`"release"`) as "X4 inventory domain." The reviewer correctly flagged these as **X3**: they are the English text a user reads when running `opm instance list`/`delete`/`tree`, and contradicted the command name. Refined the boundary — **user-facing output/prompt/log strings are X3; only the Go symbols (`releaseName`/`releaseID` vars, `--release-name`/`--release-id` flags, `ReleaseSelectorFlags`, inventory types) remain X4.** Fixed all five strings → "instance"/"instance-id"/"instances"; rebuilt, re-tested, lint 0 issues, smoke re-verified.
